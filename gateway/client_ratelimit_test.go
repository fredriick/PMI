package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClientRateLimiter_New(t *testing.T) {
	crl := NewClientRateLimiter(nil, 100, 60)
	if crl == nil {
		t.Fatal("NewClientRateLimiter returned nil")
	}
	if crl.defaultLimit != 100 {
		t.Errorf("expected defaultLimit 100, got %d", crl.defaultLimit)
	}
	if crl.windowSeconds != 60 {
		t.Errorf("expected windowSeconds 60, got %d", crl.windowSeconds)
	}
}

func TestClientRateLimiter_Middleware_Whitelisted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	crl := NewClientRateLimiter(nil, 100, 60)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	crl.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected whitelisted request to pass through")
	}
}

func TestClientRateLimiter_LocalCheck(t *testing.T) {
	crl := NewClientRateLimiter(nil, 2, 60)
	ip := "192.168.1.1"

	allowed, _ := crl.localCheck(ip, 2)
	if !allowed {
		t.Error("expected first request allowed")
	}
	allowed, _ = crl.localCheck(ip, 2)
	if !allowed {
		t.Error("expected second request allowed")
	}
	allowed, _ = crl.localCheck(ip, 2)
	if allowed {
		t.Error("expected third request denied (limit reached)")
	}
}

func TestClientRateLimiter_SetAndGetClientLimit(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	crl := NewClientRateLimiter(client, 100, 60)
	crl.SetClientLimit("10.0.0.1", 50)

	limit, err := crl.GetClientLimit("10.0.0.1")
	if err != nil {
		t.Fatalf("GetClientLimit error: %v", err)
	}
	if limit != 50 {
		t.Errorf("expected limit 50, got %d", limit)
	}
}

func TestClientRateLimiter_GetClientLimit_NoRedis(t *testing.T) {
	crl := NewClientRateLimiter(nil, 42, 60)
	limit, err := crl.GetClientLimit("any-ip")
	if err != nil {
		t.Fatalf("GetClientLimit error: %v", err)
	}
	if limit != 42 {
		t.Errorf("expected default limit 42, got %d", limit)
	}
}

func TestClientRateLimiter_IsWhitelisted(t *testing.T) {
	crl := NewClientRateLimiter(nil, 100, 60)
	if !crl.isWhitelisted("127.0.0.1") {
		t.Error("expected loopback to be whitelisted")
	}
	if !crl.isWhitelisted("10.0.0.1") {
		t.Error("expected private IP to be whitelisted")
	}
	if crl.isWhitelisted("8.8.8.8") {
		t.Error("expected public IP to NOT be whitelisted")
	}
}

func TestClientRateLimiter_RemoveClientLimit(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	crl := NewClientRateLimiter(client, 100, 60)
	crl.SetClientLimit("10.0.0.2", 75)
	crl.RemoveClientLimit("10.0.0.2")
}

func TestClientRateLimiter_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	crl := NewClientRateLimiter(nil, 1, 60)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	crl.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected first request to pass")
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/test", nil)
	crl.Middleware()(c2)
	if !c2.IsAborted() {
		t.Error("expected second request to be rate limited")
	}
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w2.Code)
	}
}
