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

func TestHasMinScope_WithRedis(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)

	readKey, _ := svc.CreateKey("read-key", 0, string(ScopeRead))
	writeKey, _ := svc.CreateKey("write-key", 0, string(ScopeWrite))
	adminKey, _ := svc.CreateKey("admin-key", 0, string(ScopeAdmin))
	allKey, _ := svc.CreateKey("all-key", 0, string(ScopeAll))

	tests := []struct {
		key       string
		minScope  string
		expected  bool
	}{
		{readKey.Key, "read", true},
		{readKey.Key, "write", false},
		{readKey.Key, "admin", false},
		{writeKey.Key, "read", true},
		{writeKey.Key, "write", true},
		{writeKey.Key, "admin", false},
		{adminKey.Key, "read", true},
		{adminKey.Key, "write", true},
		{adminKey.Key, "admin", true},
		{allKey.Key, "read", true},
		{allKey.Key, "write", true},
		{allKey.Key, "admin", true},
		{"invalid-key", "read", false},
	}

	for _, tt := range tests {
		got := svc.HasMinScope(tt.key, tt.minScope)
		if got != tt.expected {
			t.Errorf("HasMinScope(%q, %q) = %v, want %v", tt.key, tt.minScope, got, tt.expected)
		}
	}
}

func TestRequireMinScope_MiddlewareIntegration(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	readKey, _ := svc.CreateKey("read-only", 0, string(ScopeRead))
	writeKey, _ := svc.CreateKey("write-only", 0, string(ScopeWrite))

	gw := &Gateway{apiKeyService: svc}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing auth"})
			return
		}
		key := extractBearerToken(authHeader)
		if key == "" {
			key = authHeader
		}
		scope, err := svc.ValidateKey(key)
		if err != nil || scope == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}
		c.Set("api_key", key)
		c.Set("api_key_scope", scope)
		c.Next()
	})
	router.Use(gw.requireMinScope("write"))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+readKey.Key)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for read key on write endpoint, got %d", w.Code)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+writeKey.Key)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for write key on write endpoint, got %d: %s", w2.Code, w2.Body.String())
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
