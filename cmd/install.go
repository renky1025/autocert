package cmd

import (
	"autocert/internal/cert"
	"autocert/internal/logger"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装和配置 HTTPS 证书",
	Long: `为指定域名安装 Let's Encrypt HTTPS 证书，并自动配置 Web 服务器。

示例:
  # 单域名证书
  autocert install --domain example.com --email admin@example.com --nginx
  
  # 多域名证书（SAN证书）
  autocert install --domains "example.com,www.example.com,api.example.com" --email admin@example.com --nginx
  
  # 泛域名证书（需要DNS验证）
  autocert install --domain "*.example.com" --email admin@example.com --nginx --dns
  
  # 二级域名
  autocert install --domain sub.example.com --email admin@example.com --nginx
  
  # 混合域名（主域名+泛域名）
  autocert install --domains "example.com,*.example.com" --email admin@example.com --nginx --dns`,
	RunE: runInstall,
}

var (
	domain       string
	domains      string // 多域名，逗号分隔
	email        string
	webroot      string
	standalone   bool
	dnsChallenge bool // DNS 验证模式
	nginx        bool
	apache       bool
	iis          bool
)

func init() {
	rootCmd.AddCommand(installCmd)

	// 域名参数
	installCmd.Flags().StringVarP(&domain, "domain", "d", "", "要申请证书的单个域名")
	installCmd.Flags().StringVar(&domains, "domains", "", "多个域名，用逗号分隔 (例: example.com,www.example.com,*.example.com)")
	installCmd.Flags().StringVarP(&email, "email", "e", "", "用于 Let's Encrypt 账户的邮箱地址 (必需)")

	// 验证模式
	installCmd.Flags().StringVarP(&webroot, "webroot", "w", "", "Webroot 模式的网站根目录路径")
	installCmd.Flags().BoolVar(&standalone, "standalone", false, "使用 Standalone 模式验证")
	installCmd.Flags().BoolVar(&dnsChallenge, "dns", false, "使用 DNS 验证模式（泛域名证书必需）")

	// Web 服务器类型
	installCmd.Flags().BoolVar(&nginx, "nginx", false, "配置 Nginx")
	installCmd.Flags().BoolVar(&apache, "apache", false, "配置 Apache")
	installCmd.Flags().BoolVar(&iis, "iis", false, "配置 IIS")

	// 标记必需参数
	installCmd.MarkFlagRequired("email")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// 解析域名列表
	domainList, err := parseDomains()
	if err != nil {
		return fmt.Errorf("域名参数解析失败: %w", err)
	}

	logger.Info("开始安装证书", "domains", domainList, "email", email)

	// 验证参数
	if err := validateInstallFlags(domainList); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 使用统一的证书管理器
	return installCertificate(domainList)
}

// installCertificate 安装证书（统一处理单域名和多域名）
func installCertificate(domainList []string) error {
	// 创建统一的证书管理器
	certManager := cert.NewManagerWithDomains(domainList, email)
	if certManager == nil {
		return fmt.Errorf("创建证书管理器失败")
	}

	// 设置验证模式
	if dnsChallenge || certManager.HasWildcard() {
		certManager.SetChallengeType(cert.ChallengeDNS)
		logger.Info("使用 DNS 验证模式")
	} else if standalone {
		certManager.SetChallengeType(cert.ChallengeStandalone)
	} else if webroot != "" {
		certManager.SetChallengeType(cert.ChallengeWebroot)
		certManager.SetWebrootPath(webroot)
	} else {
		certManager.SetChallengeType(cert.ChallengeWebroot)
	}

	// 设置 Web 服务器类型
	if nginx {
		certManager.SetWebServer(cert.WebServerNginx)
	} else if apache {
		certManager.SetWebServer(cert.WebServerApache)
	} else if iis {
		certManager.SetWebServer(cert.WebServerIIS)
	}

	// 申请并安装证书
	if err := certManager.Install(); err != nil {
		logger.Error("证书安装失败", "domains", domainList, "error", err)
		return fmt.Errorf("证书安装失败: %w", err)
	}

	logger.Info("证书安装成功", "domains", domainList)
	if len(domainList) == 1 {
		fmt.Printf("✓ 域名 %s 证书安装成功\n", domainList[0])
	} else {
		fmt.Printf("✓ 多域名证书安装成功，包含 %d 个域名: %s\n", len(domainList), strings.Join(domainList, ", "))
	}
	return nil
}

// parseDomains 解析域名列表
func parseDomains() ([]string, error) {
	var domainList []string

	// 如果指定了 domains 参数，优先使用
	if domains != "" {
		domainList = strings.Split(domains, ",")
		for i, d := range domainList {
			domainList[i] = strings.TrimSpace(d)
		}
	} else if domain != "" {
		// 否则使用单个 domain 参数
		domainList = []string{strings.TrimSpace(domain)}
	} else {
		return nil, fmt.Errorf("必须指定 --domain 或 --domains 参数")
	}

	// 验证域名格式
	for _, d := range domainList {
		if d == "" {
			return nil, fmt.Errorf("域名不能为空")
		}
		if err := validateDomainName(d); err != nil {
			return nil, fmt.Errorf("域名 %s 格式无效: %w", d, err)
		}
	}

	return domainList, nil
}

// validateDomainName 验证域名格式
func validateDomainName(domain string) error {
	// 基本的域名格式验证
	if len(domain) == 0 {
		return fmt.Errorf("域名不能为空")
	}

	// 检查泛域名格式
	if strings.HasPrefix(domain, "*.") {
		if len(domain) < 4 { // *.x 至少4个字符
			return fmt.Errorf("泛域名格式无效")
		}
		// 验证泛域名后面的部分
		baseDomain := domain[2:]
		if strings.Contains(baseDomain, "*") {
			return fmt.Errorf("泛域名只能有一个通配符在开头")
		}
	}

	// 检查域名中是否包含无效字符
	if strings.Contains(domain, " ") {
		return fmt.Errorf("域名不能包含空格")
	}

	return nil
}

func validateInstallFlags(domainList []string) error {
	// 验证至少指定了一种 Web 服务器
	if !nginx && !apache && !iis {
		return fmt.Errorf("必须指定至少一种 Web 服务器类型: --nginx, --apache, 或 --iis")
	}

	// 验证只指定了一种 Web 服务器
	count := 0
	if nginx {
		count++
	}
	if apache {
		count++
	}
	if iis {
		count++
	}
	if count > 1 {
		return fmt.Errorf("只能指定一种 Web 服务器类型")
	}

	// 检查泛域名是否使用了 DNS 验证
	hasWildcard := false
	for _, d := range domainList {
		if strings.HasPrefix(d, "*.") {
			hasWildcard = true
			break
		}
	}

	if hasWildcard && !dnsChallenge {
		return fmt.Errorf("泛域名证书必须使用 DNS 验证模式，请添加 --dns 参数")
	}

	// 验证验证模式不能同时指定多个
	challengeCount := 0
	if standalone {
		challengeCount++
	}
	if webroot != "" {
		challengeCount++
	}
	if dnsChallenge {
		challengeCount++
	}

	if challengeCount > 1 {
		return fmt.Errorf("只能指定一种验证模式: --standalone, --webroot, 或 --dns")
	}

	return nil
}
