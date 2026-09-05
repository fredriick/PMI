package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestResponseCache_NoRedis(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)

	rc.Set("key1", "response1", "text/html")

	resp, ct, found := rc.Get("key1")
	if !found {
		t.Fatal("expected cache hit")
	}
	if resp != "response1" {
		t.Errorf("expected 'response1', got '%s'", resp)
	}
	if ct != "text/html" {
		t.Errorf("expected 'text/html', got '%s'", ct)
	}

	_, _, found = rc.Get("nonexistent")
	if found {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestResponseCache_Invalidate(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)

	rc.Set("key1", "response1", "text/html")
	rc.Set("key2", "response2", "application/json")

	rc.Invalidate("key1")

	_, _, found := rc.Get("key1")
	if found {
		t.Error("expected key1 to be invalidated")
	}

	_, _, found = rc.Get("key2")
	if !found {
		t.Error("expected key2 to still be cached")
	}
}

func TestResponseCache_InvalidatePattern(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)

	rc.Set("prefix:1", "response1", "text/html")
	rc.Set("prefix:2", "response2", "text/html")
	rc.Set("other:1", "response3", "text/html")

	rc.Invalidate("prefix:*")

	_, _, found := rc.Get("prefix:1")
	if found {
		t.Error("expected prefix:1 to be invalidated by pattern")
	}
	_, _, found = rc.Get("prefix:2")
	if found {
		t.Error("expected prefix:2 to be invalidated by pattern")
	}
	_, _, found = rc.Get("other:1")
	if !found {
		t.Error("expected other:1 to still be cached")
	}
}

func TestResponseCache_Middleware(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)
	rc.Set("cached-path", "cached response", "text/plain")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rc.Middleware(5*time.Minute, func(c *gin.Context) string {
		return "cached-path"
	}))
	router.GET("/cached", func(c *gin.Context) {
		c.String(http.StatusOK, "not cached")
	})

	req := httptest.NewRequest("GET", "/cached", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Cache") != "HIT" {
		t.Error("expected X-Cache: HIT")
	}
	if w.Body.String() != "cached response" {
		t.Errorf("expected 'cached response', got '%s'", w.Body.String())
	}
}

func TestResponseCache_Middleware_NonGet(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rc.Middleware(5*time.Minute, func(c *gin.Context) string {
		return "some-key"
	}))
	router.POST("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/api", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestResponseCache_Middleware_EmptyKey(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rc.Middleware(5*time.Minute, func(c *gin.Context) string {
		return ""
	}))
	router.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNodeRetryHandler_SetGetFallback(t *testing.T) {
	nrh := NewNodeRetryHandler(3, 100*time.Millisecond)

	fallbacks := []string{"node2", "node3", "node4"}
	nrh.SetFallback("node1", fallbacks)

	got := nrh.GetFallback("node1")
	if got != "node2" {
		t.Errorf("expected 'node2', got '%s'", got)
	}

	got = nrh.GetFallback("nonexistent")
	if got != "" {
		t.Errorf("expected empty string for nonexistent node, got '%s'", got)
	}
}

func TestNodeRetryHandler_ExecuteWithRetry_Success(t *testing.T) {
	nrh := NewNodeRetryHandler(2, 1*time.Millisecond)

	attempts := 0
	err := nrh.ExecuteWithRetry(func(nodeID string) error {
		attempts++
		if attempts < 2 {
			return http.ErrAbortHandler
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestNodeRetryHandler_ExecuteWithRetry_Failure(t *testing.T) {
	nrh := NewNodeRetryHandler(1, 1*time.Millisecond)

	attempts := 0
	err := nrh.ExecuteWithRetry(func(nodeID string) error {
		attempts++
		return http.ErrAbortHandler
	})

	if err == nil {
		t.Error("expected error after max retries")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRateLimitHeaders_SetHeaders(t *testing.T) {
	rlh := NewRateLimitHeaders(true)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	now := time.Now().Add(60 * time.Second)
	rlh.SetHeaders(c, 100, 50, now)

	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("expected X-RateLimit-Limit 100, got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "50" {
		t.Errorf("expected X-RateLimit-Remaining 50, got '%s'", w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestRateLimitHeaders_SetHeaders_Disabled(t *testing.T) {
	rlh := NewRateLimitHeaders(false)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	rlh.SetHeaders(c, 100, 0, time.Now())

	if w.Header().Get("X-RateLimit-Limit") != "" {
		t.Error("expected no headers when disabled")
	}
}

func TestRateLimitHeaders_Middleware(t *testing.T) {
	rlh := NewRateLimitHeaders(true)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rlh.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("expected X-RateLimit-Limit 100, got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}
}
