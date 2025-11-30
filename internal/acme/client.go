package acme

import (
	"autocert/internal/logger"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// ChallengeType ACME 挑战类型
type ChallengeType string

const (
	ChallengeHTTP01    ChallengeType = "http-01"
	ChallengeTLSALPN01 ChallengeType = "tls-alpn-01"
	ChallengeDNS01     ChallengeType = "dns-01"
)

// User 实现 lego 的 User 接口
type User struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *User) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// Client ACME 客户端
type Client struct {
	user      *User
	client    *lego.Client
	configDir string
	staging   bool // 是否使用测试环境
	webroot   string
	httpPort  string
	tlsPort   string
}

// ClientConfig 客户端配置
type ClientConfig struct {
	Email     string
	ConfigDir string
	Staging   bool   // 使用 Let's Encrypt 测试环境
	Webroot   string // Webroot 路径
	HTTPPort  string // HTTP 挑战端口
	TLSPort   string // TLS-ALPN 挑战端口
}

// NewClient 创建 ACME 客户端
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg.Email == "" {
		return nil, fmt.Errorf("email 不能为空")
	}

	client := &Client{
		configDir: cfg.ConfigDir,
		staging:   cfg.Staging,
		webroot:   cfg.Webroot,
		httpPort:  cfg.HTTPPort,
		tlsPort:   cfg.TLSPort,
	}

	// 加载或创建用户
	user, err := client.loadOrCreateUser(cfg.Email)
	if err != nil {
		return nil, fmt.Errorf("加载用户失败: %w", err)
	}
	client.user = user

	// 创建 lego 配置
	config := lego.NewConfig(user)
	config.Certificate.KeyType = certcrypto.RSA2048

	// 设置 ACME 服务器
	if cfg.Staging {
		config.CADirURL = lego.LEDirectoryStaging
		logger.Info("使用 Let's Encrypt 测试环境")
	} else {
		config.CADirURL = lego.LEDirectoryProduction
		logger.Info("使用 Let's Encrypt 生产环境")
	}

	// 创建 lego 客户端
	legoClient, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建 ACME 客户端失败: %w", err)
	}
	client.client = legoClient

	// 注册用户（如果尚未注册）
	if user.Registration == nil {
		reg, err := legoClient.Registration.Register(registration.RegisterOptions{
			TermsOfServiceAgreed: true,
		})
		if err != nil {
			return nil, fmt.Errorf("注册 ACME 账户失败: %w", err)
		}
		user.Registration = reg

		// 保存用户信息
		if err := client.saveUser(user); err != nil {
			logger.Warn("保存用户信息失败", "error", err)
		}
		logger.Info("ACME 账户注册成功", "email", cfg.Email)
	}

	return client, nil
}

// SetHTTPChallenge 设置 HTTP-01 挑战
func (c *Client) SetHTTPChallenge() error {
	if c.webroot != "" {
		// 使用 webroot 模式
		provider := http01.NewProviderServer("", c.httpPort)
		// 注意：lego 的 webroot provider 需要额外配置
		return c.client.Challenge.SetHTTP01Provider(provider)
	}

	// 使用内置 HTTP 服务器
	port := c.httpPort
	if port == "" {
		port = "80"
	}
	return c.client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", port))
}

// SetTLSChallenge 设置 TLS-ALPN-01 挑战
func (c *Client) SetTLSChallenge() error {
	port := c.tlsPort
	if port == "" {
		port = "443"
	}
	return c.client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", port))
}

// ObtainCertificate 获取证书
func (c *Client) ObtainCertificate(domains []string) (*certificate.Resource, error) {
	logger.Info("开始申请证书", "domains", domains)

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := c.client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("获取证书失败: %w", err)
	}

	logger.Info("证书申请成功", "domains", domains)
	return certificates, nil
}

// RenewCertificate 续期证书
func (c *Client) RenewCertificate(cert *certificate.Resource) (*certificate.Resource, error) {
	logger.Info("开始续期证书", "domains", cert.Domain)

	newCert, err := c.client.Certificate.Renew(*cert, true, false, "")
	if err != nil {
		return nil, fmt.Errorf("续期证书失败: %w", err)
	}

	logger.Info("证书续期成功", "domains", cert.Domain)
	return newCert, nil
}

// SaveCertificate 保存证书到指定目录
func (c *Client) SaveCertificate(cert *certificate.Resource, certDir string) error {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	// 保存证书
	certPath := filepath.Join(certDir, "cert.pem")
	if err := os.WriteFile(certPath, cert.Certificate, 0644); err != nil {
		return fmt.Errorf("保存证书失败: %w", err)
	}

	// 保存私钥
	keyPath := filepath.Join(certDir, "key.pem")
	if err := os.WriteFile(keyPath, cert.PrivateKey, 0600); err != nil {
		return fmt.Errorf("保存私钥失败: %w", err)
	}

	// 保存证书链（如果有）
	if cert.IssuerCertificate != nil {
		chainPath := filepath.Join(certDir, "chain.pem")
		if err := os.WriteFile(chainPath, cert.IssuerCertificate, 0644); err != nil {
			return fmt.Errorf("保存证书链失败: %w", err)
		}
	}

	// 保存完整链（证书 + 中间证书）
	fullchainPath := filepath.Join(certDir, "fullchain.pem")
	fullchain := append(cert.Certificate, cert.IssuerCertificate...)
	if err := os.WriteFile(fullchainPath, fullchain, 0644); err != nil {
		return fmt.Errorf("保存完整证书链失败: %w", err)
	}

	// 保存证书元数据
	metaPath := filepath.Join(certDir, "cert.json")
	meta := map[string]interface{}{
		"domain":        cert.Domain,
		"certURL":       cert.CertURL,
		"certStableURL": cert.CertStableURL,
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		logger.Warn("保存证书元数据失败", "error", err)
	}

	logger.Info("证书保存完成",
		"certPath", certPath,
		"keyPath", keyPath,
		"domain", cert.Domain)

	return nil
}

// loadOrCreateUser 加载或创建用户
func (c *Client) loadOrCreateUser(email string) (*User, error) {
	userDir := filepath.Join(c.configDir, "accounts", sanitizeEmail(email))
	userFile := filepath.Join(userDir, "account.json")
	keyFile := filepath.Join(userDir, "account.key")

	// 尝试加载现有用户
	if _, err := os.Stat(userFile); err == nil {
		return c.loadUser(userFile, keyFile)
	}

	// 创建新用户
	logger.Info("创建新 ACME 账户", "email", email)

	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成私钥失败: %w", err)
	}

	user := &User{
		Email: email,
		key:   privateKey,
	}

	// 保存私钥
	if err := os.MkdirAll(userDir, 0700); err != nil {
		return nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return nil, err
	}

	return user, nil
}

// loadUser 加载用户
func (c *Client) loadUser(userFile, keyFile string) (*User, error) {
	// 读取用户信息
	userData, err := os.ReadFile(userFile)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}

	// 读取私钥
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("无法解析私钥文件")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	user.key = privateKey
	logger.Info("加载已有 ACME 账户", "email", user.Email)

	return &user, nil
}

// saveUser 保存用户信息
func (c *Client) saveUser(user *User) error {
	userDir := filepath.Join(c.configDir, "accounts", sanitizeEmail(user.Email))
	userFile := filepath.Join(userDir, "account.json")

	if err := os.MkdirAll(userDir, 0700); err != nil {
		return err
	}

	userData, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(userFile, userData, 0600)
}

// sanitizeEmail 清理邮箱地址用于文件名
func sanitizeEmail(email string) string {
	// 简单替换特殊字符
	result := ""
	for _, c := range email {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
}
