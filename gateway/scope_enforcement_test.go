package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/config"
)

func TestScopeLevel(t *testing.T) {
	tests := []struct {
		scope    string
		expected int
	}{
		{"read", 1},
		{"write", 2},
		{"admin", 3},
		{"*", 99},
		{"unknown", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := scopeLevel(tt.scope)
		if got != tt.expected {
			t.Errorf("scopeLevel(%q) = %d, want %d", tt.scope, got, tt.expected)
		}
	}
}

func TestRequireMinScope_NilService_AllowsAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: nil}

	mw := gw.requireMinScope("read")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	mw(c)

	if c.IsAborted() {
		t.Error("expected request to pass through when apiKeyService is nil")
	}
}

func TestRequireMinScope_ReadScopeAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("read")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("api_key_scope", "read")

	mw(c)

	if c.IsAborted() {
		t.Error("expected read scope to be allowed with requireMinScope('read')")
	}
}

func TestRequireMinScope_WriteScopeAllowedForRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("read")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("api_key_scope", "write")

	mw(c)

	if c.IsAborted() {
		t.Error("expected write scope to be allowed with requireMinScope('read')")
	}
}

func TestRequireMinScope_AdminScopeAllowedForWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("write")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("api_key_scope", "admin")

	mw(c)

	if c.IsAborted() {
		t.Error("expected admin scope to be allowed with requireMinScope('write')")
	}
}

func TestRequireMinScope_ReadScopeDeniedForWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("write")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("api_key_scope", "read")

	mw(c)

	if !c.IsAborted() {
		t.Error("expected read scope to be denied with requireMinScope('write')")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestRequireMinScope_WildcardAllowedForAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("admin")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Set("api_key_scope", "*")

	mw(c)

	if c.IsAborted() {
		t.Error("expected wildcard scope to be allowed with requireMinScope('admin')")
	}
}

func TestRequireMinScope_MissingScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gw := &Gateway{apiKeyService: &APIKeyService{}}

	mw := gw.requireMinScope("read")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	mw(c)

	if !c.IsAborted() {
		t.Error("expected request to be denied when no scope is set")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHasMinScope_WithoutRedis(t *testing.T) {
	svc := &APIKeyService{}

	result := svc.HasMinScope("some-key", "read")
	if result {
		t.Error("expected HasMinScope to return false when Redis is not configured")
	}
}

func TestApplyConfig_NilRateLimiter(t *testing.T) {
	gw := &Gateway{
		rateLimiter:         nil,
		matchmaker:            nil,
		config:                &config.GatewayConfig{},
		redisConfig:           &config.RedisConfig{},
		matchmakerConfig:      &config.MatchmakerConfig{},
		compliance:            NewComplianceService(&config.ComplianceConfig{}),
	}

	newCfg := &config.Config{
		Gateway:    config.GatewayConfig{},
		Redis:      config.RedisConfig{},
		Matchmaker: config.MatchmakerConfig{},
		Compliance: config.ComplianceConfig{},
	}

	gw.ApplyConfig(newCfg)
}
