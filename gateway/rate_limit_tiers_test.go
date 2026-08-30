package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/config"
)

func TestRateLimitTiers_GetTier(t *testing.T) {
	tiers := NewRateLimitTiers(nil)

	tests := []struct {
		tier    string
		wantReq int
	}{
		{"free", 10},
		{"basic", 100},
		{"premium", 1000},
		{"enterprise", 10000},
		{"unknown", 10},
	}

	for _, tt := range tests {
		got := tiers.GetTier(tt.tier)
		if got.Requests != tt.wantReq {
			t.Errorf("GetTier(%q) = %d requests, want %d", tt.tier, got.Requests, tt.wantReq)
		}
	}
}

func TestRateLimitTiers_GetTierForKey_NoRedis(t *testing.T) {
	tiers := NewRateLimitTiers(nil)

	got := tiers.GetTierForKey("any-key")
	if got != "basic" {
		t.Errorf("GetTierForKey with nil redis should return 'basic', got '%s'", got)
	}

	got = tiers.GetTierForKey("")
	if got != "basic" {
		t.Errorf("GetTierForKey with empty key should return 'basic', got '%s'", got)
	}
}

func TestRateLimitTiers_SetKeyTier_NoRedis(t *testing.T) {
	tiers := NewRateLimitTiers(nil)

	err := tiers.SetKeyTier("some-key", "premium")
	if err != nil {
		t.Errorf("SetKeyTier with nil redis should return nil, got %v", err)
	}
}

func TestRateLimitTiers_ListTiers(t *testing.T) {
	tiers := NewRateLimitTiers(nil)

	list := tiers.ListTiers()
	if len(list) != 4 {
		t.Errorf("expected 4 tiers, got %d", len(list))
	}

	names := make(map[string]bool)
	for _, tier := range list {
		names[tier.Name] = true
	}

	expected := []string{"free", "basic", "premium", "enterprise"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tier: %s", name)
		}
	}
}

func TestTieredRateLimiter_Allow(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)
	tiers := NewRateLimitTiers(nil)
	tlr := NewTieredRateLimiter(rl, tiers)

	for i := 0; i < 10; i++ {
		allowed, err := tlr.Allow("client1", "premium")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed (premium tier=1000)", i+1)
		}
	}

	original := rl.GetLimit()
	if original != 100 {
		t.Errorf("original limit should be restored to 100, got %d", original)
	}
}

func TestTieredRateLimiter_Allow_RespectsTierLimit(t *testing.T) {
	rl := NewLocalRateLimiter(1000, 60)
	tiers := NewRateLimitTiers(nil)
	tlr := NewTieredRateLimiter(rl, tiers)

	tier := tiers.GetTier("basic")
	for i := 0; i < tier.Requests; i++ {
		allowed, _ := tlr.Allow("client1", "any-key")
		if !allowed {
			t.Errorf("request %d should be allowed (basic tier=%d)", i+1, tier.Requests)
		}
	}

	allowed, _ := tlr.Allow("client1", "any-key")
	if allowed {
		t.Errorf("request %d beyond basic tier limit should be blocked", tier.Requests)
	}
}

func TestTieredRateLimiter_Middleware(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)
	tiers := NewRateLimitTiers(nil)
	tlr := NewTieredRateLimiter(rl, tiers)

	router := gin.New()
	router.Use(tlr.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGenerateReferralCode(t *testing.T) {
	code := GenerateReferralCode()

	if len(code) < 12 {
		t.Errorf("referral code too short: %s (expected at least 12 chars)", code)
	}

	if code[:4] != "REF-" {
		t.Errorf("expected prefix 'REF-', got '%s'", code[:4])
	}
}

func TestGenerateReferralCode_Unique(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := GenerateReferralCode()
		if codes[code] {
			t.Errorf("duplicate referral code generated: %s", code)
		}
		codes[code] = true
	}
}

func TestHealthHandler(t *testing.T) {
	gw := &Gateway{}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	gw.healthHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp["status"])
	}

	if resp["version"] == "" {
		t.Error("expected non-empty version")
	}
}

func TestApplyConfig_UpdatesRateLimiterFromStruct(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:      nil,
		config:          &config.GatewayConfig{RateLimitRequests: 100, RateLimitWindowSeconds: 60, CircuitBreakerThreshold: 5},
		redisConfig:     &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:     NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway:    config.GatewayConfig{RateLimitRequests: 50, RateLimitWindowSeconds: 30},
		Compliance: config.ComplianceConfig{},
	}

	if rl.GetLimit() != 100 {
		t.Fatalf("expected initial limit 100, got %d", rl.GetLimit())
	}

	gw.ApplyConfig(newCfg)

	if rl.GetLimit() != 50 {
		t.Errorf("expected rate limiter limit 50 after ApplyConfig, got %d", rl.GetLimit())
	}
}
