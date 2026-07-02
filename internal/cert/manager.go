package cert

import (
	"autocert/internal/acme"
	"autocert/internal/config"
	"autocert/internal/logger"
	"autocert/internal/webserver"
	"crypto/x509"
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
	webServerName string
	dnsProvider   string
	certDir       string
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
	}
}

// NewManagerFromSiteSpec 从持久化站点配置创建管理器。
func NewManagerFromSiteSpec(spec *SiteSpec) (*Manager, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	manager := NewManagerWithDomains(spec.Domains, spec.Email)
	if manager == nil {
		return nil, fmt.Errorf("创建证书管理器失败")
	}

	manager.SetWebrootPath(spec.WebrootPath)
	manager.SetDNSProvider(spec.DNSProvider)

	switch spec.Challenge {
	case ChallengeWebroot.String():
		manager.SetChallengeType(ChallengeWebroot)
	case ChallengeStandalone.String():
		manager.SetChallengeType(ChallengeStandalone)
	case ChallengeDNS.String():
		manager.SetChallengeType(ChallengeDNS)
	default:
		return nil, fmt.Errorf("不支持的 challenge 类型: %s", spec.Challenge)
	}

	switch strings.ToLower(spec.WebServer) {
	case "", "unknown":
	case "nginx":
		manager.SetWebServer(WebServerNginx)
	case "apache":
		manager.SetWebServer(WebServerApache)
	case "iis":
		manager.SetWebServer(WebServerIIS)
	default:
		return nil, fmt.Errorf("不支持的 Web 服务器类型: %s", spec.WebServer)
	}

	return manager, nil
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

// SetDNSProvider 设置 DNS provider 名称。
func (m *Manager) SetDNSProvider(provider string) {
	m.dnsProvider = provider
}

// SetWebServer 设置 Web 服务器类型
func (m *Manager) SetWebServer(webServerType WebServerType) {
	m.webServerType = webServerType
	m.webServerName = webServerType.String()

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
	if m.email == "" {
		return fmt.Errorf("email 不能为空")
	}
	if m.challengeType == ChallengeWebroot && m.webrootPath == "" {
		return fmt.Errorf("webroot 验证需要指定网站根目录")
	}
	if m.challengeType == ChallengeDNS && m.dnsProvider == "" {
		logger.Warn("DNS 验证将进入手动 TXT 记录校验流程")
	}

	// 1. 创建证书目录
	if err := m.createCertDir(); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	// 2. 通过 ACME 获取证书
	if err := m.obtainCertificate(); err != nil {
		return fmt.Errorf("获取证书失败: %w", err)
	}

	// 3. 配置 Web 服务器
	if err := m.configureWebServer(); err != nil {
		return fmt.Errorf("配置 Web 服务器失败: %w", err)
	}

	if err := SaveSiteSpec(m.SiteSpec()); err != nil {
		return fmt.Errorf("保存站点配置失败: %w", err)
	}

	logger.Info("证书安装完成", "domains", m.domains)
	return nil
}

// Renew 续期证书。
func (m *Manager) Renew(force bool) error {
	logger.Info("开始续期证书", "domains", m.domains)

	// 检查证书是否需要续期
	certInfo, err := m.GetCertInfo()
	if err != nil {
		return fmt.Errorf("获取证书信息失败: %w", err)
	}

	// 如果证书有效期超过 30 天，则不需要续期
	if !force && certInfo.DaysLeft > 30 {
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
	return buildSiteDirName(m.domains)
}

// createCertDir 创建证书目录
func (m *Manager) createCertDir() error {
	certDir := filepath.Join(m.certDir, m.getDirName())
	return os.MkdirAll(certDir, 0755)
}

// obtainCertificate 通过 ACME 获取证书
func (m *Manager) obtainCertificate() error {
	logger.Info("开始 ACME 证书申请流程",
		"domains", m.domains,
		"challengeType", m.challengeType.String())

	switch m.challengeType {
	case ChallengeWebroot:
		return m.obtainCertificateWebroot()
	case ChallengeStandalone:
		return m.obtainCertificateStandalone()
	case ChallengeDNS:
		return m.obtainCertificateDNS()
	default:
		return fmt.Errorf("不支持的验证模式: %d", m.challengeType)
	}
}

// obtainCertificateWebroot 使用 Webroot/HTTP 模式获取证书
func (m *Manager) obtainCertificateWebroot() error {
	logger.Info("使用 HTTP-01 模式获取证书", "domains", m.domains, "webroot", m.webrootPath)

	if m.HasWildcard() {
		return fmt.Errorf("泛域名证书不能使用 HTTP 验证模式，请使用 DNS 验证")
	}

	return m.obtainWithACME(acme.ChallengeHTTP01)
}

// obtainCertificateStandalone 使用 Standalone/TLS-ALPN 模式获取证书
func (m *Manager) obtainCertificateStandalone() error {
	logger.Info("使用 TLS-ALPN-01 模式获取证书", "domains", m.domains)

	if m.HasWildcard() {
		return fmt.Errorf("泛域名证书不能使用 TLS-ALPN 验证模式，请使用 DNS 验证")
	}

	return m.obtainWithACME(acme.ChallengeTLSALPN01)
}

// obtainCertificateDNS 使用 DNS 模式获取证书
func (m *Manager) obtainCertificateDNS() error {
	logger.Info("使用 DNS-01 模式获取证书", "domains", m.domains)
	return m.obtainWithACME(acme.ChallengeDNS01)
}

// obtainWithACME 使用 ACME 客户端获取证书
func (m *Manager) obtainWithACME(challengeType acme.ChallengeType) error {
	// 创建 ACME 客户端
	client, err := acme.NewClient(&acme.ClientConfig{
		Email:     m.email,
		ConfigDir: m.certDir,
		Staging:   false, // 生产环境
		Webroot:   m.webrootPath,
		DNSName:   m.dnsProvider,
	})
	if err != nil {
		return fmt.Errorf("创建 ACME 客户端失败: %w", err)
	}

	// 设置挑战类型
	switch challengeType {
	case acme.ChallengeHTTP01:
		if err := client.SetHTTPChallenge(); err != nil {
			return fmt.Errorf("设置 HTTP 挑战失败: %w", err)
		}
	case acme.ChallengeTLSALPN01:
		if err := client.SetTLSChallenge(); err != nil {
			return fmt.Errorf("设置 TLS-ALPN 挑战失败: %w", err)
		}
	case acme.ChallengeDNS01:
		if err := client.SetDNSChallenge(); err != nil {
			return fmt.Errorf("设置 DNS 挑战失败: %w", err)
		}
	}

	// 申请证书
	cert, err := client.ObtainCertificate(m.domains)
	if err != nil {
		return fmt.Errorf("ACME 证书申请失败: %w", err)
	}

	// 保存证书到目录
	certDir := filepath.Join(m.certDir, m.getDirName())
	if err := client.SaveCertificate(cert, certDir); err != nil {
		return fmt.Errorf("保存证书失败: %w", err)
	}

	return nil
}

// SiteSpec 返回当前管理器的持久化配置。
func (m *Manager) SiteSpec() *SiteSpec {
	return &SiteSpec{
		Domains:     append([]string(nil), m.domains...),
		Email:       m.email,
		Challenge:   m.challengeType.String(),
		WebrootPath: m.webrootPath,
		WebServer:   m.webServerName,
		DNSProvider: m.dnsProvider,
	}
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
		CertPath: m.getPreferredCertPath(),
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

func (m *Manager) getFullChainPath() string {
	return filepath.Join(m.certDir, m.getDirName(), "fullchain.pem")
}

func (m *Manager) getPreferredCertPath() string {
	if _, err := os.Stat(m.getFullChainPath()); err == nil {
		return m.getFullChainPath()
	}
	return m.getCertPath()
}
