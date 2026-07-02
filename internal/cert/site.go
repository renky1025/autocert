package cert

import (
	"autocert/internal/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const siteSpecFileName = "site.json"

// SiteSpec 描述一个可被续期和管理的证书站点。
type SiteSpec struct {
	Domains     []string  `json:"domains"`
	Email       string    `json:"email"`
	Challenge   string    `json:"challenge"`
	WebrootPath string    `json:"webroot_path,omitempty"`
	WebServer   string    `json:"web_server,omitempty"`
	DNSProvider string    `json:"dns_provider,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *SiteSpec) PrimaryDomain() string {
	if len(s.Domains) == 0 {
		return ""
	}
	return s.Domains[0]
}

func (s *SiteSpec) DirName() string {
	return buildSiteDirName(s.Domains)
}

func (s *SiteSpec) MatchesDomain(domain string) bool {
	return slices.Contains(s.Domains, domain)
}

func (s *SiteSpec) SupportsAutoRenewal() bool {
	switch s.Challenge {
	case ChallengeWebroot.String():
		return s.WebrootPath != ""
	default:
		return false
	}
}

func (s *SiteSpec) Validate() error {
	if len(s.Domains) == 0 {
		return fmt.Errorf("域名不能为空")
	}
	if s.Email == "" {
		return fmt.Errorf("email 不能为空")
	}
	if s.Challenge == "" {
		return fmt.Errorf("challenge 不能为空")
	}
	return nil
}

func SaveSiteSpec(spec *SiteSpec) error {
	if err := spec.Validate(); err != nil {
		return err
	}

	certDir := filepath.Join(config.GetCertDir(), spec.DirName())
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("创建站点目录失败: %w", err)
	}

	now := time.Now()
	if existing, err := readSiteSpecFile(filepath.Join(certDir, siteSpecFileName)); err == nil && !existing.InstalledAt.IsZero() {
		spec.InstalledAt = existing.InstalledAt
	}
	if spec.InstalledAt.IsZero() {
		spec.InstalledAt = now
	}
	spec.UpdatedAt = now

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("编码站点配置失败: %w", err)
	}

	if err := os.WriteFile(filepath.Join(certDir, siteSpecFileName), data, 0644); err != nil {
		return fmt.Errorf("写入站点配置失败: %w", err)
	}

	return nil
}

func LoadSiteSpec(domain string) (*SiteSpec, error) {
	specs, err := ListSiteSpecs()
	if err != nil {
		return nil, err
	}

	for _, spec := range specs {
		if spec.MatchesDomain(domain) {
			return spec, nil
		}
	}

	return nil, fmt.Errorf("未找到域名 %s 的站点配置", domain)
}

func ListSiteSpecs() ([]*SiteSpec, error) {
	baseDir := config.GetCertDir()
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*SiteSpec{}, nil
		}
		return nil, fmt.Errorf("读取证书目录失败: %w", err)
	}

	specs := make([]*SiteSpec, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		spec, err := readSiteSpecFile(filepath.Join(baseDir, entry.Name(), siteSpecFileName))
		if err != nil {
			continue
		}
		specs = append(specs, spec)
	}

	slices.SortFunc(specs, func(a, b *SiteSpec) int {
		switch {
		case a.PrimaryDomain() < b.PrimaryDomain():
			return -1
		case a.PrimaryDomain() > b.PrimaryDomain():
			return 1
		default:
			return 0
		}
	})

	return specs, nil
}

func buildSiteDirName(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	if len(domains) > 1 {
		return fmt.Sprintf("%s_san", domains[0])
	}
	return domains[0]
}

func readSiteSpecFile(path string) (*SiteSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var spec SiteSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}
