package cert

import (
	"autocert/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildSiteDirName(t *testing.T) {
	if got := buildSiteDirName([]string{"example.com"}); got != "example.com" {
		t.Fatalf("unexpected dir name: %s", got)
	}
	if got := buildSiteDirName([]string{"example.com", "www.example.com"}); got != "example.com_san" {
		t.Fatalf("unexpected SAN dir name: %s", got)
	}
}

func TestSaveAndListSiteSpecs(t *testing.T) {
	tmpDir := t.TempDir()
	config.AppConfig = &config.Config{CertDir: tmpDir}
	t.Cleanup(func() { config.AppConfig = nil })

	spec := &SiteSpec{
		Domains:     []string{"example.com", "www.example.com"},
		Email:       "admin@example.com",
		Challenge:   ChallengeWebroot.String(),
		WebrootPath: "/var/www/example.com",
		WebServer:   "nginx",
	}

	if err := SaveSiteSpec(spec); err != nil {
		t.Fatalf("SaveSiteSpec() error = %v", err)
	}

	specPath := filepath.Join(tmpDir, "example.com_san", siteSpecFileName)
	if _, err := os.Stat(specPath); err != nil {
		t.Fatalf("site spec not written: %v", err)
	}

	list, err := ListSiteSpecs()
	if err != nil {
		t.Fatalf("ListSiteSpecs() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 site spec, got %d", len(list))
	}
	if list[0].PrimaryDomain() != "example.com" {
		t.Fatalf("unexpected primary domain: %s", list[0].PrimaryDomain())
	}
	if !list[0].SupportsAutoRenewal() {
		t.Fatalf("expected webroot site to support auto renewal")
	}
}

func TestLoadSiteSpecByContainedDomain(t *testing.T) {
	tmpDir := t.TempDir()
	config.AppConfig = &config.Config{CertDir: tmpDir}
	t.Cleanup(func() { config.AppConfig = nil })

	if err := SaveSiteSpec(&SiteSpec{
		Domains:     []string{"example.com", "api.example.com"},
		Email:       "admin@example.com",
		Challenge:   ChallengeDNS.String(),
		DNSProvider: "cloudflare",
	}); err != nil {
		t.Fatalf("SaveSiteSpec() error = %v", err)
	}

	spec, err := LoadSiteSpec("api.example.com")
	if err != nil {
		t.Fatalf("LoadSiteSpec() error = %v", err)
	}
	if spec.PrimaryDomain() != "example.com" {
		t.Fatalf("unexpected primary domain: %s", spec.PrimaryDomain())
	}
	if spec.SupportsAutoRenewal() {
		t.Fatalf("expected manual DNS site to be excluded from auto renewal")
	}
}
