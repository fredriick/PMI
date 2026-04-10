package gateway

type RateLimitTier struct {
	Name      string
	Requests  int
	WindowSec int
}

type RateLimitTiers struct {
	Tiers map[string]RateLimitTier
}

func NewRateLimitTiers() *RateLimitTiers {
	return &RateLimitTiers{
		Tiers: map[string]RateLimitTier{
			"free":       {Name: "free", Requests: 10, WindowSec: 60},
			"basic":      {Name: "basic", Requests: 100, WindowSec: 60},
			"premium":    {Name: "premium", Requests: 1000, WindowSec: 60},
			"enterprise": {Name: "enterprise", Requests: 10000, WindowSec: 60},
		},
	}
}

func (r *RateLimitTiers) GetTier(tierName string) RateLimitTier {
	if tier, ok := r.Tiers[tierName]; ok {
		return tier
	}
	return r.Tiers["free"]
}

func (r *RateLimitTiers) GetTierForKey(apiKey string) string {
	return "basic"
}

func (r *RateLimitTiers) ListTiers() []RateLimitTier {
	var tiers []RateLimitTier
	for _, tier := range r.Tiers {
		tiers = append(tiers, tier)
	}
	return tiers
}
