package gateway

import (
	"testing"

	"proxymesh/internal/config"
)

func TestIsBlocked_WildcardSubdomain(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{
		BlockedDomains: []string{"*.bankofamerica.com", "*.gov", "*.mil"},
		KYCRequired:    false,
	})

	tests := []struct {
		target  string
		blocked bool
	}{
		{"https://bankofamerica.com/login", true},
		{"https://www.bankofamerica.com", true},
		{"https://sub.bankofamerica.com/path", true},
		{"https://deep.sub.bankofamerica.com", true},
		{"https://bankofamerica.org", false},
		{"https://notbankofamerica.com", false},
		{"https://example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		got := cs.IsBlocked(tt.target)
		if got != tt.blocked {
			t.Errorf("IsBlocked(%q) = %v, want %v", tt.target, got, tt.blocked)
		}
	}
}

func TestIsBlocked_GovMil(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{
		BlockedDomains: []string{"*.gov", "*.mil"},
		KYCRequired:    false,
	})

	tests := []struct {
		target  string
		blocked bool
	}{
		{"https://irs.gov/taxes", true},
		{"https://www.irs.gov", true},
		{"https://army.mil", true},
		{"https://example.gov.uk", false},
		{"https://example.com", false},
	}

	for _, tt := range tests {
		got := cs.IsBlocked(tt.target)
		if got != tt.blocked {
			t.Errorf("IsBlocked(%q) = %v, want %v", tt.target, got, tt.blocked)
		}
	}
}

func TestIsBlocked_ExactMatch(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{
		BlockedDomains: []string{"illegal-content.com", "badsite.net"},
		KYCRequired:    false,
	})

	tests := []struct {
		target  string
		blocked bool
	}{
		{"https://illegal-content.com", true},
		{"https://badsite.net/path", true},
		{"https://sub.illegal-content.com", false},
		{"https://legal-content.com", false},
	}

	for _, tt := range tests {
		got := cs.IsBlocked(tt.target)
		if got != tt.blocked {
			t.Errorf("IsBlocked(%q) = %v, want %v", tt.target, got, tt.blocked)
		}
	}
}

func TestIsBlocked_CaseInsensitive(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{
		BlockedDomains: []string{"*.BankOfAmerica.COM"},
		KYCRequired:    false,
	})

	if !cs.IsBlocked("https://www.bankofamerica.com") {
		t.Error("expected case-insensitive match for *.BankOfAmerica.COM")
	}
	if !cs.IsBlocked("https://BANKOFAMERICA.COM") {
		t.Error("expected case-insensitive match for BANKOFAMERICA.COM")
	}
}

func TestIsGovernmentDomain(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{})

	tests := []struct {
		domain string
		gov    bool
	}{
		{"irs.gov", true},
		{"army.mil", true},
		{"example.gov.uk", true},
		{"service.gov.au", true},
		{"agency.gc.ca", true},
		{"gov.uk", false},
		{"example.com", false},
		{"government.example.com", false},
	}

	for _, tt := range tests {
		got := cs.IsGovernmentDomain(tt.domain)
		if got != tt.gov {
			t.Errorf("IsGovernmentDomain(%q) = %v, want %v", tt.domain, got, tt.gov)
		}
	}
}

func TestIsFinancialDomain(t *testing.T) {
	cs := NewComplianceService(&config.ComplianceConfig{})

	tests := []struct {
		domain string
		fin    bool
	}{
		{"chase.com", true},
		{"wellsfargo.com", true},
		{"bankofamerica.com", true},
		{"citibank.com", true},
		{"mybank.example.com", true},
		{"example.com", false},
		{"news.google.com", false},
	}

	for _, tt := range tests {
		got := cs.IsFinancialDomain(tt.domain)
		if got != tt.fin {
			t.Errorf("IsFinancialDomain(%q) = %v, want %v", tt.domain, got, tt.fin)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		target string
		domain string
	}{
		{"https://example.com/path", "example.com"},
		{"http://example.com", "example.com"},
		{"example.com/path", "example.com"},
		{"example.com?query=1", "example.com?query=1"},
		{"example.com", "example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractDomain(tt.target)
		if got != tt.domain {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.target, got, tt.domain)
		}
	}
}
