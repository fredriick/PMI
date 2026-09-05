package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"proxymesh/internal/config"
)

func TestApplyConfig_UpdatesRateLimiter(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:    NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:     nil,
		config:         &config.GatewayConfig{RateLimitRequests: 100, RateLimitWindowSeconds: 60},
		redisConfig:    &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:     NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			RateLimitRequests:        200,
			RateLimitWindowSeconds:   30,
			CircuitBreakerThreshold:  10,
			CooldownSeconds:          1800,
			RequestIDPrefix:          "req",
			RequestIDFormat:          "unix",
		},
		Redis:         config.RedisConfig{},
		Matchmaker:    config.MatchmakerConfig{CooldownTTLMinutes: 30},
		Compliance:    config.ComplianceConfig{BlockedDomains: []string{"*.gov"}, KYCRequired: false},
	}

	gw.ApplyConfig(newCfg)

	if rl.GetLimit() != 200 {
		t.Errorf("expected rate limiter limit 200, got %d", rl.GetLimit())
	}
}

func TestApplyConfig_UpdatesCircuitBreakerThreshold(t *testing.T) {
	gw := &Gateway{
		rateLimiter:             NewTieredRateLimiter(NewLocalRateLimiter(100, 60), NewRateLimitTiers(nil)),
		matchmaker:              nil,
		config:                  &config.GatewayConfig{CircuitBreakerThreshold: 5},
		redisConfig:             &config.RedisConfig{},
		matchmakerConfig:        &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		circuitBreakerThreshold: 5,
		compliance:              NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			CircuitBreakerThreshold: 10,
			RateLimitRequests:       100,
			RateLimitWindowSeconds:  60,
		},
		Redis:      config.RedisConfig{},
		Matchmaker: config.MatchmakerConfig{CooldownTTLMinutes: 15},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(newCfg)

	if gw.circuitBreakerThreshold != 10 {
		t.Errorf("expected circuit breaker threshold 10, got %d", gw.circuitBreakerThreshold)
	}
}

func TestApplyConfig_UpdatesCompliance(t *testing.T) {
	gw := &Gateway{
		rateLimiter: NewTieredRateLimiter(NewLocalRateLimiter(100, 60), NewRateLimitTiers(nil)),
		matchmaker:  nil,
		config:      &config.GatewayConfig{},
		redisConfig: &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance: NewComplianceService(&config.ComplianceConfig{
			BlockedDomains: []string{"*.gov"},
			KYCRequired:    false,
		}),
	}

	newCfg := &config.Config{
		Gateway:    config.GatewayConfig{},
		Redis:      config.RedisConfig{},
		Matchmaker: config.MatchmakerConfig{CooldownTTLMinutes: 15},
		Compliance: config.ComplianceConfig{
			BlockedDomains: []string{"*.evil.com"},
			KYCRequired:    true,
		},
	}

	gw.ApplyConfig(newCfg)

	if !gw.compliance.KYCRequired {
		t.Error("expected KYCRequired to be true after ApplyConfig")
	}
	if !gw.compliance.IsBlocked("test.evil.com") {
		t.Error("expected evil.com to be blocked after ApplyConfig")
	}
}

func TestApplyConfig_UpdatesRequestID(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)
	ri := NewRequestIDGenerator("req", "unix", nil)

	gw := &Gateway{
		rateLimiter:   NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:    nil,
		config:        &config.GatewayConfig{RequestIDPrefix: "req", RequestIDFormat: "unix"},
		redisConfig:   &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		requestID:     ri,
		compliance:    NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			RequestIDPrefix:          "custom",
			RequestIDFormat:          "timestamp",
			RateLimitRequests:        100,
			RateLimitWindowSeconds:   60,
			CircuitBreakerThreshold:  5,
		},
		Redis:      config.RedisConfig{},
		Matchmaker: config.MatchmakerConfig{CooldownTTLMinutes: 15},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(newCfg)

	snapshot := ri.Config()
	if snapshot["prefix"] != "custom" {
		t.Errorf("expected prefix 'custom', got '%s'", snapshot["prefix"])
	}
	if snapshot["format"] != "timestamp" {
		t.Errorf("expected format 'timestamp', got '%s'", snapshot["format"])
	}
}

func TestApplyConfig_UpdatesRequestLogging(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:   NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:    nil,
		config:        &config.GatewayConfig{RequestLoggingEnabled: false},
		redisConfig:   &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		requestLogger: NewRequestLogger(nil, false),
		compliance:    NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			RequestLoggingEnabled:    true,
			RateLimitRequests:        100,
			RateLimitWindowSeconds:   60,
			CircuitBreakerThreshold:  5,
		},
		Redis:      config.RedisConfig{},
		Matchmaker: config.MatchmakerConfig{CooldownTTLMinutes: 15},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(newCfg)

	if !gw.requestLogger.IsEnabled() {
		t.Error("expected request logger to be enabled after ApplyConfig")
	}
}

func TestUpdateConfigHandler_PersistsAndApplies(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:   NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:    nil,
		config:        &config.GatewayConfig{RateLimitRequests: 100, RateLimitWindowSeconds: 60, CircuitBreakerThreshold: 5, CooldownSeconds: 3600},
		redisConfig:   &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:    NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"gateway": {
			"rate_limit_requests": 200,
			"rate_limit_window_seconds": 30,
			"circuit_breaker_threshold": 10
		},
		"matchmaker": {
			"cooldown_ttl_minutes": 30
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	applied, _ := resp["applied"].([]interface{})
	hasRateLimit := false
	for _, a := range applied {
		if a.(string) == "rate_limit_requests" {
			hasRateLimit = true
		}
	}
	if !hasRateLimit {
		t.Error("expected 'rate_limit_requests' in applied list")
	}

	if rl.GetLimit() != 200 {
		t.Errorf("expected rate limiter limit 200 after update, got %d", rl.GetLimit())
	}

	if gw.circuitBreakerThreshold != 10 {
		t.Errorf("expected circuit breaker threshold 10 after update, got %d", gw.circuitBreakerThreshold)
	}
}

func TestRateLimiter_SetWindow_Distributed(t *testing.T) {
	t.Skip("requires Redis")
	rl := &DistributedRateLimiter{
		client: redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15}),
		limit:  100,
		window: 60 * time.Second,
	}

	rl.SetWindow(30 * time.Second)
	if rl.window != 30*time.Second {
		t.Errorf("expected window 30s, got %v", rl.window)
	}
}

func TestUpdateConfigHandler_PersistsToFile(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:      nil,
		config:          &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:     &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:      NewComplianceService(&config.ComplianceConfig{}),
	}

	cfgPath := config.ConfigPath()
	if cfgPath == "" || !fileExists(cfgPath) {
		t.Skip("config file path not set or file does not exist")
	}

	originalContent, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skip("cannot read config file")
	}

	defer func() {
		_ = os.WriteFile(cfgPath, originalContent, 0644)
	}()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"gateway": {
			"rate_limit_requests": 500
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	updatedContent, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read updated config file: %v", err)
	}

	if !strings.Contains(string(updatedContent), "rate_limit_requests: 500") {
		t.Error("config file was not updated with new rate_limit_requests value")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestConfigReload_ApplyConfigFromStruct(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:      nil,
		config:          &config.GatewayConfig{RateLimitRequests: 100, RateLimitWindowSeconds: 60, CircuitBreakerThreshold: 3},
		redisConfig:     &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:      NewComplianceService(&config.ComplianceConfig{BlockedDomains: []string{"*.gov"}}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			RateLimitRequests:        300,
			RateLimitWindowSeconds:   120,
			CircuitBreakerThreshold:  10,
			CooldownSeconds:          1800,
			RequestIDPrefix:          "custom",
			RequestIDFormat:          "timestamp",
			RequestLoggingEnabled:    true,
		},
		Redis:      config.RedisConfig{MaxRetries: 3},
		Matchmaker: config.MatchmakerConfig{CooldownTTLMinutes: 45},
		Compliance: config.ComplianceConfig{
			BlockedDomains: []string{"*.gov", "*.mil", "*.evil.com"},
			KYCRequired:    false,
		},
	}

	gw.ApplyConfig(newCfg)

	if rl.GetLimit() != 300 {
		t.Errorf("expected rate limiter limit 300, got %d", rl.GetLimit())
	}

	if gw.circuitBreakerThreshold != 10 {
		t.Errorf("expected circuit breaker threshold 10, got %d", gw.circuitBreakerThreshold)
	}

	if !gw.compliance.IsBlocked("test.evil.com") {
		t.Error("expected evil.com to be blocked after config reload")
	}

	if !gw.compliance.IsBlocked("test.mil") {
		t.Error("expected mil to be blocked after config reload")
	}

	if !strings.EqualFold(gw.config.RequestIDPrefix, "custom") {
		t.Errorf("expected request ID prefix 'custom', got '%s'", gw.config.RequestIDPrefix)
	}

	if gw.config.RateLimitRequests != 300 {
		t.Errorf("expected config rate_limit_requests 300, got %d", gw.config.RateLimitRequests)
	}

	if gw.config.RateLimitWindowSeconds != 120 {
		t.Errorf("expected config rate_limit_window_seconds 120, got %d", gw.config.RateLimitWindowSeconds)
	}
}

func TestConfigReload_EndToEnd_ApiToComponents(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)
	ri := NewRequestIDGenerator("req", "unix", nil)
	logger := NewRequestLogger(nil, false)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:      nil,
		config:          &config.GatewayConfig{
			RateLimitRequests:        100,
			RateLimitWindowSeconds:   60,
			CircuitBreakerThreshold:  5,
			RequestIDPrefix:          "req",
			RequestIDFormat:          "unix",
			RequestLoggingEnabled:    false,
		},
		redisConfig:     &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		requestID:       ri,
		requestLogger:   logger,
		compliance:     NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)

	originalContent := ""
	cfgPath := config.ConfigPath()
	if cfgPath != "" && fileExists(cfgPath) {
		origBytes, _ := os.ReadFile(cfgPath)
		originalContent = string(origBytes)
		defer func() {
			if originalContent != "" {
				_ = os.WriteFile(cfgPath, []byte(originalContent), 0644)
			}
		}()
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"gateway": {
			"rate_limit_requests": 42,
			"rate_limit_window_seconds": 15,
			"circuit_breaker_threshold": 8,
			"request_id_prefix": "x",
			"request_id_format": "uuid",
			"request_logging_enabled": true
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if rl.GetLimit() != 42 {
		t.Errorf("expected rate limiter limit 42, got %d", rl.GetLimit())
	}

	if gw.circuitBreakerThreshold != 8 {
		t.Errorf("expected circuit breaker threshold 8, got %d", gw.circuitBreakerThreshold)
	}

	if !logger.IsEnabled() {
		t.Error("expected request logger to be enabled after config update")
	}

	snapshot := ri.Config()
	if snapshot["prefix"] != "x" {
		t.Errorf("expected request ID prefix 'x', got '%s'", snapshot["prefix"])
	}
	if snapshot["format"] != "uuid" {
		t.Errorf("expected request ID format 'uuid', got '%s'", snapshot["format"])
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	applied, _ := resp["applied"].([]interface{})
	if len(applied) < 4 {
		t.Errorf("expected at least 4 applied changes, got %d", len(applied))
	}

	needsRestart, _ := resp["needsRestart"].([]interface{})
	if len(needsRestart) > 0 {
		t.Logf("changes requiring restart: %v (expected for this test)", needsRestart)
	}
}

func TestApplyConfig_UpdatesRateLimiterWindow(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:      nil,
		config:          &config.GatewayConfig{RateLimitRequests: 100, RateLimitWindowSeconds: 60},
		redisConfig:     &config.RedisConfig{},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:     NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway: config.GatewayConfig{
			RateLimitRequests:        200,
			RateLimitWindowSeconds:   120,
			CircuitBreakerThreshold:  10,
		},
		Matchmaker:  config.MatchmakerConfig{CooldownTTLMinutes: 45},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(newCfg)

	if rl.GetLimit() != 200 {
		t.Errorf("expected rate limiter limit 200, got %d", rl.GetLimit())
	}

	if gw.config.RateLimitWindowSeconds != 120 {
		t.Errorf("expected config rate_limit_window_seconds 120, got %d", gw.config.RateLimitWindowSeconds)
	}

	if gw.matchmakerConfig.CooldownTTLMinutes != 45 {
		t.Errorf("expected matchmaker cooldown_ttl_minutes 45, got %d", gw.matchmakerConfig.CooldownTTLMinutes)
	}
}

func TestUpdateConfigHandler_InvalidJSON(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		config:          &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:     &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:      NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{invalid json`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestUpdateConfigHandler_EmptyBody(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:     NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		config:          &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:     &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig: &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:      NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for empty body, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	applied, _ := resp["applied"].([]interface{})
	if len(applied) != 0 {
		t.Errorf("expected 0 applied changes for empty body, got %d", len(applied))
	}
}

func TestConfigHandler_GetsConfig(t *testing.T) {
	gw := &Gateway{
		config: &config.GatewayConfig{
			RateLimitRequests: 100,
			Port:              8000,
		},
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/config", nil)

	gw.getConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "success" {
		t.Errorf("expected status 'success', got '%v'", resp["status"])
	}

	cfg, ok := resp["config"].(map[string]interface{})
	if !ok {
		t.Fatal("expected config in response")
	}
	if cfg["RateLimitRequests"] != float64(100) {
		t.Errorf("expected RateLimitRequests 100, got %v", cfg["RateLimitRequests"])
	}
	if cfg["Port"] != float64(8000) {
		t.Errorf("expected Port 8000, got %v", cfg["Port"])
	}
}

func TestApplyConfig_NilCompliance(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:       NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:        nil,
		config:            &config.GatewayConfig{},
		redisConfig:       &config.RedisConfig{},
		matchmakerConfig:  &config.MatchmakerConfig{},
		compliance:        nil,
	}

	emptyCfg := &config.Config{
		Gateway:    config.GatewayConfig{},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(emptyCfg)

	if gw.compliance == nil {
		t.Error("expected compliance to be initialized after ApplyConfig")
	}
}

func TestUpdateConfigHandler_PersistsRedisMaxRetries(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:       NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		config:            &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:       &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig:  &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:        NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"redis": {
			"max_retries": 5
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if gw.redisConfig.MaxRetries != 5 {
		t.Errorf("expected redis max_retries 5 after update, got %d", gw.redisConfig.MaxRetries)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	applied, _ := resp["applied"].([]interface{})
	hasMaxRetries := false
	for _, a := range applied {
		if a.(string) == "redis.max_retries" {
			hasMaxRetries = true
		}
	}
	if !hasMaxRetries {
		t.Error("expected 'redis.max_retries' in applied list")
	}
}

func TestUpdateConfigHandler_PersistsComplianceBlockedDomains(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:       NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		config:            &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:       &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig:  &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:        NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"compliance": {
			"blocked_domains": ["*.evil.com", "badsite.net"],
			"kyc_required": true
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if !gw.compliance.KYCRequired {
		t.Error("expected KYCRequired to be true after update")
	}
	if len(gw.compliance.BlockedDomains) != 2 {
		t.Errorf("expected 2 blocked domains, got %d", len(gw.compliance.BlockedDomains))
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	applied, _ := resp["applied"].([]interface{})
	hasKYC := false
	hasBlocked := false
	for _, a := range applied {
		switch a.(string) {
		case "compliance.kyc_required":
			hasKYC = true
		case "compliance.blocked_domains":
			hasBlocked = true
		}
	}
	if !hasKYC {
		t.Error("expected 'compliance.kyc_required' in applied list")
	}
	if !hasBlocked {
			t.Error("expected 'compliance.blocked_domains' in applied list")
	}
}

func TestUpdateConfigHandler_PersistsMatchmakerCooldown(t *testing.T) {
	rl := NewLocalRateLimiter(100, 60)

	gw := &Gateway{
		rateLimiter:       NewTieredRateLimiter(rl, NewRateLimitTiers(nil)),
		matchmaker:        nil,
		config:            &config.GatewayConfig{RateLimitRequests: 100},
		redisConfig:       &config.RedisConfig{MaxRetries: 3},
		matchmakerConfig:  &config.MatchmakerConfig{CooldownTTLMinutes: 15},
		compliance:        NewComplianceService(&config.ComplianceConfig{}),
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/config", strings.NewReader(`{
		"matchmaker": {
			"cooldown_ttl_minutes": 45
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if gw.matchmakerConfig.CooldownTTLMinutes != 45 {
		t.Errorf("expected cooldown_ttl_minutes 45, got %d", gw.matchmakerConfig.CooldownTTLMinutes)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	applied, _ := resp["applied"].([]interface{})
	hasCooldown := false
	for _, a := range applied {
		if a.(string) == "matchmaker.cooldown_ttl_minutes" {
			hasCooldown = true
		}
	}
	if !hasCooldown {
		t.Error("expected 'matchmaker.cooldown_ttl_minutes' in applied list")
	}
}
