package cmd

import "testing"

func TestParseDomains(t *testing.T) {
	t.Cleanup(resetInstallFlags)

	domains = " example.com, www.example.com "
	got, err := parseDomains()
	if err != nil {
		t.Fatalf("parseDomains() error = %v", err)
	}
	if len(got) != 2 || got[0] != "example.com" || got[1] != "www.example.com" {
		t.Fatalf("unexpected domains: %#v", got)
	}
}

func TestValidateInstallFlagsRequiresSkipScheduleForWildcardDNS(t *testing.T) {
	t.Cleanup(resetInstallFlags)

	domain = "*.example.com"
	email = "admin@example.com"
	nginx = true
	dnsChallenge = true

	domainList, err := parseDomains()
	if err != nil {
		t.Fatalf("parseDomains() error = %v", err)
	}

	if err := validateInstallFlags(domainList); err == nil {
		t.Fatal("expected wildcard DNS install to require skip-schedule")
	}
}

func TestValidateInstallFlagsAllowsScheduledWebrootInstall(t *testing.T) {
	t.Cleanup(resetInstallFlags)

	domain = "example.com"
	email = "admin@example.com"
	nginx = true
	webroot = "/var/www/example.com"

	domainList, err := parseDomains()
	if err != nil {
		t.Fatalf("parseDomains() error = %v", err)
	}

	if err := validateInstallFlags(domainList); err != nil {
		t.Fatalf("validateInstallFlags() error = %v", err)
	}
}

func resetInstallFlags() {
	domain = ""
	domains = ""
	email = ""
	webroot = ""
	standalone = false
	dnsChallenge = false
	skipSchedule = false
	nginx = false
	apache = false
	iis = false
}
