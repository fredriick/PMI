package gateway

import (
	"strings"

	"proxymesh/internal/config"
)

type ComplianceService struct {
	BlockedDomains []string
	KYCRequired    bool
}

func NewComplianceService(cfg *config.ComplianceConfig) *ComplianceService {
	return &ComplianceService{
		BlockedDomains: cfg.BlockedDomains,
		KYCRequired:    cfg.KYCRequired,
	}
}

func (c *ComplianceService) IsBlocked(target string) bool {
	domain := extractDomain(target)
	if domain == "" {
		return false
	}

	domain = strings.ToLower(domain)

	for _, blocked := range c.BlockedDomains {
		blocked = strings.ToLower(blocked)
		blocked = strings.TrimPrefix(blocked, "*.")

		if strings.HasPrefix(blocked, "*.") {
			suffix := strings.TrimPrefix(blocked, "*.")
			if strings.HasSuffix(domain, suffix) {
				return true
			}
		}

		if domain == blocked || domain == "."+blocked {
			return true
		}
	}

	return false
}

func extractDomain(target string) string {
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	if idx := strings.Index(target, "/"); idx != -1 {
		target = target[:idx]
	}

	return target
}

func (c *ComplianceService) IsGovernmentDomain(domain string) bool {
	domains := []string{
		".gov",
		".mil",
		".gov.uk",
		".gov.au",
		".gc.ca",
	}

	domain = strings.ToLower(domain)
	for _, d := range domains {
		if strings.HasSuffix(domain, d) {
			return true
		}
	}
	return false
}

func (c *ComplianceService) IsFinancialDomain(domain string) bool {
	financialPatterns := []string{
		"bank",
		"chase",
		"wellsfargo",
		"bankofamerica",
		"citibank",
	}

	domain = strings.ToLower(domain)
	for _, pattern := range financialPatterns {
		if strings.Contains(domain, pattern) {
			return true
		}
	}
	return false
}

func (c *ComplianceService) ValidateKYC(userID string) bool {
	return !c.KYCRequired
}
