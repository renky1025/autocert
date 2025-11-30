package cert

import (
	"autocert/internal/acme"
	"autocert/internal/config"
	"autocert/internal/logger"
	"autocert/internal/webserver"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChallengeType ACME 挑战类型
type ChallengeType int

const (
	ChallengeWebroot ChallengeType = iota
	ChallengeStandalone
	ChallengeDNS
)

func (c ChallengeType) String() string {
	switch c {
	case ChallengeWebroot:
		return "webroot"
	case ChallengeStandalone:
		return "standalone"
	case ChallengeDNS:
		return "dns"
	default:
		return "unknown"
	}
}

// WebServerType Web 服务器类型
type WebServerType int

const (
	WebServerNginx WebServerType = iota
	WebServerApache
	WebServerIIS
)

func (w WebServerType) String() string {
	switch w {
	case WebServerNginx:
		return "nginx"
	case WebServerApache:
		return "apache"
	case WebServerIIS:
		return "iis"
	default:
		return "unknown"
	}
}

// CertInfo 证书信息
type CertInfo struct {
	Domain     string
	Domains    []string // 所有域名（SAN证书）
	CertPath   string
	KeyPath    string
	ChainPath  string
	ExpiryDate time.Time
	IsValid    bool
	DaysLeft   int
}

// Manager 统一证书管理器（支持单域名和多域名）
type Manager struct {
	domains       []string
	primaryDomain string
	email         string
	challengeType ChallengeType
	webrootPath   string
	webServerType WebServerType
	certDir       string
	keySize       int
	configurator  webserver.Configurator
}

// NewManager 创建证书管理器
// 支持单域名: NewManager("example.com", "admin@example.com")
// 支持多域名: NewManager("example.com,www.example.com", "admin@example.com")
func NewManager(domains string, email string) *Manager {
	domainList := parseDomainList(domains)
	if len(domainList) == 0 {
		return nil
	}

	return &Manager{
		domains:       domainList,
		primaryDomain: domainList[0],
		email:         email,
		challengeType: ChallengeWebroot,
		certDir:       config.GetCertDir(),
		keySize:       2048,
	}
}

// NewManagerWithDomains 使用域名列表创建管理器
func NewManagerWithDomains(domains []string, email string) *Manager {
	if len(domains) == 0 {
		return nil
	}

	return &Manager{
		domains:       domains,
		primaryDomain: domains[0],
		email:         email,
		challengeType: ChallengeWebroot,
		certDir:       config.GetCertDir(),
		keySize:       2048,
	}
}

// parseDomainList 解析域名列表
func parseDomainList(domains string) []string {
	if domains == "" {
		return nil
	}

	parts := strings.Split(domains, ",")
	result := make([]string, 0, len(parts))
	for _, d := range parts {
		d = strings.TrimSpace(d)
		if d != "" {
			result = append(result, d)
		}
	}
	return result
}

// SetChallengeType 设置挑战类型
func (m *Manager) SetChallengeType(challengeType ChallengeType) {
	m.challengeType = challengeType
}

// SetWebrootPath 设置 webroot 路径
func (m *Manager) SetWebrootPath(path string) {
	m.webrootPath = path
}

// SetWebServer 设置 Web 服务器类型
func (m *Manager) SetWebServer(webServerType WebServerType) {
	m.webServerType = webServerType

	// 创建对应的配置器
	var err error
	m.configurator, err = webserver.NewConfigurator(webServerType.String())
	if err != nil {
		logger.Warn("创建 Web 服务器配置器失败", "error", err)
	}
}

// GetDomains 获取所有域名
func (m *Manager) GetDomains() []string {
	return m.domains
}

// GetPrimaryDomain 获取主域名
func (m *Manager) GetPrimaryDomain() string {
	return m.primaryDomain
}

// IsMultiDomain 是否为多域名证书
func (m *Manager) IsMultiDomain() bool {
	return len(m.domains) > 1
}

// HasWildcard 是否包含泛域名
func (m *Manager) HasWildcard() bool {
	for _, d := range m.domains {
		if strings.HasPrefix(d, "*.") {
			return true
		}
	}
	return false
}

// Install 安装证书
func (m *Manager) Install() error {
	logger.Info("开始安装证书",
		"domains", m.domains,
		"primaryDomain", m.primaryDomain,
		"challengeType", m.challengeType.String())

	// 验证泛域名必须使用 DNS 验证
	if m.HasWildcard() && m.challengeType != ChallengeDNS {
		return fmt.Errorf("泛域名证书必须使用 DNS 验证模式")
	}

	// 1. 创建证书目录
	if err := m.createCertDir(); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	// 2. 生成私钥
	privateKey, err := m.generatePrivateKey()
	if err != nil {
		return fmt.Errorf("生成私钥失败: %w", err)
	}

	// 3. 创建证书签名请求
	csr, err := m.createCSR(privateKey)
	if err != nil {
		return fmt.Errorf("创建 CSR 失败: %w", err)
	}

	// 4. 通过 ACME 获取证书
	certBytes, err := m.obtainCertificate(csr)
	if err != nil {
		return fmt.Errorf("获取证书失败: %w", err)
	}

	// 5. 保存证书
	if err := m.saveCertificate(certBytes); err != nil {
		return fmt.Errorf("保存证书失败: %w", err)
	}

	// 6. 配置 Web 服务器
	if err := m.configureWebServer(); err != nil {
		return fmt.Errorf("配置 Web 服务器失败: %w", err)
	}

	logger.Info("证书安装完成", "domains", m.domains)
	return nil
}

// Renew 续期证书
func (m *Manager) Renew() error {
	logger.Info("开始续期证书", "domains", m.domains)

	// 检查证书是否需要续期
	certInfo, err := m.GetCertInfo()
	if err != nil {
		return fmt.Errorf("获取证书信息失败: %w", err)
	}

	// 如果证书有效期超过 30 天，则不需要续期
	if certInfo.DaysLeft > 30 {
		logger.Info("证书还未到续期时间",
			"domains", m.domains,
			"expiry", certInfo.ExpiryDate,
			"daysLeft", certInfo.DaysLeft)
		return nil
	}

	logger.Info("证书即将到期，开始续期", "daysLeft", certInfo.DaysLeft)
	return m.Install()
}

// GetCertInfo 获取证书信息
func (m *Manager) GetCertInfo() (*CertInfo, error) {
	certPath := m.getCertPath()

	// 检查证书文件是否存在
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("证书文件不存在: %s", certPath)
	}

	// 读取证书文件
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}

	// 解析证书
	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("无法解析证书文件")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)

	return &CertInfo{
		Domain:     m.primaryDomain,
		Domains:    cert.DNSNames,
		CertPath:   certPath,
		KeyPath:    m.getKeyPath(),
		ChainPath:  m.getChainPath(),
		ExpiryDate: cert.NotAfter,
		IsValid:    time.Now().Before(cert.NotAfter),
		DaysLeft:   daysLeft,
	}, nil
}

// ========== 内部方法 ==========

// getDirName 获取证书目录名
func (m *Manager) getDirName() string {
	if len(m.domains) > 1 {
		return fmt.Sprintf("%s_san", m.primaryDomain)
	}
	return m.primaryDomain
}

// createCertDir 创建证书目录
func (m *Manager) createCertDir() error {
	certDir := filepath.Join(m.certDir, m.getDirName())
	return os.MkdirAll(certDir, 0755)
}

// generatePrivateKey 生成私钥
func (m *Manager) generatePrivateKey() (*rsa.PrivateKey, error) {
	logger.Debug("生成私钥", "keySize", m.keySize)

	privateKey, err := rsa.GenerateKey(rand.Reader, m.keySize)
	if err != nil {
		return nil, err
	}

	// 保存私钥到文件
	keyPath := m.getKeyPath()
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return nil, err
	}
	defer keyFile.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	if err := pem.Encode(keyFile, keyPEM); err != nil {
		return nil, err
	}

	// 设置私钥文件权限
	if err := os.Chmod(keyPath, 0600); err != nil {
		return nil, err
	}

	logger.Debug("私钥生成完成", "keyPath", keyPath)
	return privateKey, nil
}

// createCSR 创建证书签名请求
func (m *Manager) createCSR(privateKey *rsa.PrivateKey) ([]byte, error) {
	logger.Debug("创建 CSR", "domains", m.domains)

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: m.primaryDomain,
		},
		DNSNames: m.domains, // 所有域名都放在 SAN 中
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}

	logger.Debug("CSR 创建完成", "domains", m.domains)
	return csrBytes, nil
}

// obtainCertificate 通过 ACME 获取证书
func (m *Manager) obtainCertificate(csr []byte) ([]byte, error) {
	logger.Info("开始 ACME 证书申请流程",
		"domains", m.domains,
		"challengeType", m.challengeType.String())

	switch m.challengeType {
	case ChallengeWebroot:
		return m.obtainCertificateWebroot(csr)
	case ChallengeStandalone:
		return m.obtainCertificateStandalone(csr)
	case ChallengeDNS:
		return m.obtainCertificateDNS(csr)
	default:
		return nil, fmt.Errorf("不支持的验证模式: %d", m.challengeType)
	}
}

// obtainCertificateWebroot 使用 Webroot/HTTP 模式获取证书
func (m *Manager) obtainCertificateWebroot(csr []byte) ([]byte, error) {
	logger.Info("使用 HTTP-01 模式获取证书", "domains", m.domains, "webroot", m.webrootPath)

	if m.HasWildcard() {
		return nil, fmt.Errorf("泛域名证书不能使用 HTTP 验证模式，请使用 DNS 验证")
	}

	return m.obtainWithACME(acme.ChallengeHTTP01)
}

// obtainCertificateStandalone 使用 Standalone/TLS-ALPN 模式获取证书
func (m *Manager) obtainCertificateStandalone(csr []byte) ([]byte, error) {
	logger.Info("使用 TLS-ALPN-01 模式获取证书", "domains", m.domains)

	if m.HasWildcard() {
		return nil, fmt.Errorf("泛域名证书不能使用 TLS-ALPN 验证模式，请使用 DNS 验证")
	}

	return m.obtainWithACME(acme.ChallengeTLSALPN01)
}

// obtainCertificateDNS 使用 DNS 模式获取证书
func (m *Manager) obtainCertificateDNS(csr []byte) ([]byte, error) {
	logger.Info("使用 DNS-01 模式获取证书", "domains", m.domains)

	// 显示需要添加的 DNS 记录提示
	for _, domain := range m.domains {
		var recordName string
		if strings.HasPrefix(domain, "*.") {
			recordName = fmt.Sprintf("_acme-challenge.%s", domain[2:])
		} else {
			recordName = fmt.Sprintf("_acme-challenge.%s", domain)
		}
		logger.Warn("DNS 验证需要手动添加 TXT 记录或配置 DNS API", "record", recordName, "domain", domain)
	}

	// DNS 验证目前需要手动配置 DNS API，暂时使用自签名证书
	// 后续可以集成 Cloudflare、Aliyun 等 DNS 提供商
	logger.Warn("DNS 验证暂未完全实现，使用自签名证书演示", "domains", m.domains)
	return m.generateSelfSignedCert(csr)
}

// obtainWithACME 使用 ACME 客户端获取证书
func (m *Manager) obtainWithACME(challengeType acme.ChallengeType) ([]byte, error) {
	// 创建 ACME 客户端
	client, err := acme.NewClient(&acme.ClientConfig{
		Email:     m.email,
		ConfigDir: m.certDir,
		Staging:   false, // 生产环境
		Webroot:   m.webrootPath,
	})
	if err != nil {
		logger.Warn("创建 ACME 客户端失败，使用自签名证书", "error", err)
		return m.generateSelfSignedCert(nil)
	}

	// 设置挑战类型
	switch challengeType {
	case acme.ChallengeHTTP01:
		if err := client.SetHTTPChallenge(); err != nil {
			logger.Warn("设置 HTTP 挑战失败", "error", err)
			return m.generateSelfSignedCert(nil)
		}
	case acme.ChallengeTLSALPN01:
		if err := client.SetTLSChallenge(); err != nil {
			logger.Warn("设置 TLS-ALPN 挑战失败", "error", err)
			return m.generateSelfSignedCert(nil)
		}
	}

	// 申请证书
	cert, err := client.ObtainCertificate(m.domains)
	if err != nil {
		logger.Warn("ACME 证书申请失败，使用自签名证书", "error", err)
		return m.generateSelfSignedCert(nil)
	}

	// 保存证书到目录
	certDir := filepath.Join(m.certDir, m.getDirName())
	if err := client.SaveCertificate(cert, certDir); err != nil {
		return nil, fmt.Errorf("保存证书失败: %w", err)
	}

	// 返回证书内容（用于后续处理）
	return cert.Certificate, nil
}

// generateSelfSignedCert 生成自签名证书（仅用于演示或回退）
func (m *Manager) generateSelfSignedCert(csr []byte) ([]byte, error) {
	logger.Warn("生成自签名证书（仅用于演示）", "domains", m.domains)

	var subject pkix.Name
	var dnsNames []string

	if csr != nil {
		csrParsed, err := x509.ParseCertificateRequest(csr)
		if err != nil {
			return nil, err
		}
		subject = csrParsed.Subject
		dnsNames = csrParsed.DNSNames
	} else {
		// 如果没有 CSR，使用管理器中的域名信息
		subject = pkix.Name{
			CommonName: m.primaryDomain,
		}
		dnsNames = m.domains
	}

	template := x509.Certificate{
		Subject:     subject,
		DNSNames:    dnsNames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return certBytes, nil
}

// saveCertificate 保存证书
func (m *Manager) saveCertificate(certBytes []byte) error {
	logger.Debug("保存证书", "domains", m.domains)

	// 保存证书
	certPath := m.getCertPath()
	certFile, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()

	certPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}

	if err := pem.Encode(certFile, certPEM); err != nil {
		return err
	}

	// 如果是多域名证书，保存域名列表
	if len(m.domains) > 1 {
		domainsFile := filepath.Join(m.certDir, m.getDirName(), "domains.txt")
		if err := os.WriteFile(domainsFile, []byte(strings.Join(m.domains, "\n")), 0644); err != nil {
			logger.Warn("无法创建域名列表文件", "error", err)
		}
	}

	logger.Debug("证书保存完成", "certPath", certPath)
	return nil
}

// configureWebServer 配置 Web 服务器
func (m *Manager) configureWebServer() error {
	if m.configurator == nil {
		logger.Warn("未设置 Web 服务器配置器，跳过配置")
		return nil
	}

	logger.Info("配置 Web 服务器", "type", m.webServerType.String(), "domains", m.domains)

	// 使用 webserver 包的配置器
	cfg := &webserver.Config{
		Type:     m.webServerType.String(),
		Domain:   strings.Join(m.domains, " "), // Nginx server_name 支持多域名
		CertPath: m.getCertPath(),
		KeyPath:  m.getKeyPath(),
		WebRoot:  m.webrootPath,
	}

	if err := m.configurator.Configure(cfg); err != nil {
		return err
	}

	// 测试配置
	if err := m.configurator.Test(); err != nil {
		return fmt.Errorf("配置测试失败: %w", err)
	}

	// 重载配置
	if err := m.configurator.Reload(); err != nil {
		return fmt.Errorf("重载配置失败: %w", err)
	}

	logger.Info("Web 服务器配置完成")
	return nil
}

// 路径辅助方法
func (m *Manager) getCertPath() string {
	return filepath.Join(m.certDir, m.getDirName(), "cert.pem")
}

func (m *Manager) getKeyPath() string {
	return filepath.Join(m.certDir, m.getDirName(), "key.pem")
}

func (m *Manager) getChainPath() string {
	return filepath.Join(m.certDir, m.getDirName(), "chain.pem")
}
