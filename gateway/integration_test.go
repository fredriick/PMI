package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestGateway() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	comp := NewComplianceService(&config.ComplianceConfig{
		BlockedDomains: []string{"*.gov", "*.mil", "*.bankofamerica.com"},
		KYCRequired:    false,
	})

	rl := NewLocalRateLimiter(100, 60)

	router.Use(rl.Middleware())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "1.0.0"})
	})

	router.GET("/test-blocked", func(c *gin.Context) {
		target := c.Query("url")
		if comp.IsBlocked(target) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Target domain is blocked"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.Any("/proxy", func(c *gin.Context) {
		user, pass, hasAuth := c.Request.BasicAuth()
		if !hasAuth {
			user = c.Query("user")
			pass = c.Query("pass")
		}
		if user == "" || pass == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "connected", "user": user})
	})

	return router
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["status"] != "healthy" {
		t.Errorf("expected healthy, got %s", body["status"])
	}
}

func TestBlockedDomain_Gov(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/test-blocked?url=https://irs.gov/taxes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestBlockedDomain_Bank(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/test-blocked?url=https://www.bankofamerica.com/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAllowedDomain(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/test-blocked?url=https://example.com", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProxyAuth_MissingCredentials(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/proxy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProxyAuth_BasicAuth(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/proxy", nil)
	req.SetBasicAuth("testuser", "testpass")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["user"] != "testuser" {
		t.Errorf("expected testuser, got %s", body["user"])
	}
}

func TestProxyAuth_QueryParams(t *testing.T) {
	router := setupTestGateway()

	req := httptest.NewRequest("GET", "/proxy?user=queryuser&pass=querypass", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)

	if body["user"] != "queryuser" {
		t.Errorf("expected queryuser, got %s", body["user"])
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router := gin.New()
	SetupMetricsRoutes(router)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("metrics body should not be empty")
	}
}

func TestRateLimiter_Exceeds(t *testing.T) {
	router := gin.New()
	rl := NewLocalRateLimiter(2, 60)
	router.Use(rl.Middleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func TestExtractSessionFromPath(t *testing.T) {
	gw := &Gateway{}

	tests := []struct {
		path    string
		session string
	}{
		{"session-abc123", "abc123"},
		{"session-abc123/path", "abc123"},
		{"session-abc123-country-us", "abc123"},
		{"no-matching-prefix", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := gw.extractSessionFromPath(tt.path)
		if got != tt.session {
			t.Errorf("extractSessionFromPath(%q) = %q, want %q", tt.path, got, tt.session)
		}
	}
}

func TestExtractHostFromTarget(t *testing.T) {
	gw := &Gateway{}

	tests := []struct {
		target string
		host   string
	}{
		{"https://example.com/path", "example.com"},
		{"http://example.com:8080/path", "example.com:8080"},
		{"example.com/path?query=1", "example.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := gw.extractHostFromTarget(tt.target)
		if got != tt.host {
			t.Errorf("extractHostFromTarget(%q) = %q, want %q", tt.target, got, tt.host)
		}
	}
}

func TestExtractTarget(t *testing.T) {
	gw := &Gateway{}

	router := gin.New()
	router.GET("/*path", func(c *gin.Context) {
		result := gw.extractTarget(c, "country")
		c.JSON(http.StatusOK, gin.H{"country": result})
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/country-us-path", "us"},
		{"/country-de-path", "de"},
		{"/no-modifier", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var body map[string]string
		json.Unmarshal(w.Body.Bytes(), &body)

		if body["country"] != tt.expected {
			t.Errorf("path %q: expected %q, got %q", tt.path, tt.expected, body["country"])
		}
	}
}
