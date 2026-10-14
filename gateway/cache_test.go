package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestResponseCache_New(t *testing.T) {
	rc := NewResponseCache(nil, 5*time.Minute)
	if rc == nil {
		t.Fatal("NewResponseCache(nil) returned nil")
	}
	if rc.localCache == nil {
		t.Error("expected localCache map to be initialized")
	}
	if rc.ttl != 5*time.Minute {
		t.Errorf("expected TTL 5m, got %v", rc.ttl)
	}
	if rc.redis != nil {
		t.Error("expected nil redis when nil passed")
	}
}

func TestResponseCache_Get_Set_NoRedis(t *testing.T) {
	rc := NewResponseCache(nil, 1*time.Hour)

	// Get on empty cache should return false
	response, contentType, found := rc.Get("key1")
	if found {
		t.Error("expected not found for empty cache")
	}
	if response != "" || contentType != "" {
		t.Error("expected empty values for not found")
	}

	// Set and Get
	rc.Set("key1", "response1", "application/json")
	response, contentType, found = rc.Get("key1")
	if !found {
		t.Error("expected found for cached key")
	}
	if response != "response1" {
		t.Errorf("expected response1, got %q", response)
	}
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}
}

func TestResponseCache_Get_Expired(t *testing.T) {
	rc := NewResponseCache(nil, 1*time.Hour)

	// Directly insert an expired entry into localCache
	rc.mu.Lock()
	rc.localCache["expired-key"] = &cacheEntry{
		Response:    "old response",
		ContentType: "text/plain",
		Expiry:      time.Now().Add(-1 * time.Second),
	}
	rc.mu.Unlock()

	response, _, found := rc.Get("expired-key")
	if found {
		t.Error("expected not found for expired entry")
	}
	if response != "" {
		t.Errorf("expected empty response for expired entry, got %q", response)
	}
}

func TestResponseCache_Invalidate_NoPattern(t *testing.T) {
	rc := NewResponseCache(nil, 1*time.Hour)

	rc.Set("key1", "response1", "text/plain")
	rc.Set("key2", "response2", "text/plain")

	rc.Invalidate("key1")

	if _, _, found := rc.Get("key1"); found {
		t.Error("expected key1 to be invalidated")
	}
	if _, _, found := rc.Get("key2"); !found {
		t.Error("expected key2 to still exist")
	}
}

func TestResponseCache_Invalidate_WithWildcard(t *testing.T) {
	rc := NewResponseCache(nil, 1*time.Hour)

	rc.Set("prefix:key1", "response1", "text/plain")
	rc.Set("prefix:key2", "response2", "text/plain")
	rc.Set("other:key3", "response3", "text/plain")

	rc.Invalidate("prefix:*")

	if _, _, found := rc.Get("prefix:key1"); found {
		t.Error("expected prefix:key1 to be invalidated")
	}
	if _, _, found := rc.Get("prefix:key2"); found {
		t.Error("expected prefix:key2 to be invalidated")
	}
	if _, _, found := rc.Get("other:key3"); !found {
		t.Error("expected other:key3 to still exist")
	}
}

func TestNewNodeRetryHandler(t *testing.T) {
	nrh := NewNodeRetryHandler(3, 10*time.Millisecond)
	if nrh == nil {
		t.Fatal("NewNodeRetryHandler returned nil")
	}
	if nrh.maxRetries != 3 {
		t.Errorf("expected maxRetries 3, got %d", nrh.maxRetries)
	}
	if nrh.backoffBase != 10*time.Millisecond {
		t.Errorf("expected backoffBase 10ms, got %v", nrh.backoffBase)
	}
	if nrh.fallbackNodes == nil {
		t.Error("expected fallbackNodes map to be initialized")
	}
}

func TestNodeRetryHandler_SetGetFallback_Cache(t *testing.T) {
	nrh := NewNodeRetryHandler(3, 10*time.Millisecond)

	nrh.SetFallback("node-1", []string{"node-2", "node-3"})
	fallback := nrh.GetFallback("node-1")
	if fallback != "node-2" {
		t.Errorf("expected fallback node-2, got %q", fallback)
	}

	empty := nrh.GetFallback("nonexistent")
	if empty != "" {
		t.Errorf("expected empty string for nonexistent node, got %q", empty)
	}
}

func TestNodeRetryHandler_ExecuteWithRetry_Success_Cache(t *testing.T) {
	nrh := NewNodeRetryHandler(3, 1*time.Millisecond)

	attempts := 0
	err := nrh.ExecuteWithRetry(func(nodeID string) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestNodeRetryHandler_ExecuteWithRetry_Failure_Cache(t *testing.T) {
	nrh := NewNodeRetryHandler(2, 1*time.Millisecond)

	err := nrh.ExecuteWithRetry(func(nodeID string) error {
		return &testError{msg: "always fails"}
	})

	if err == nil {
		t.Error("expected error after max retries")
	}
	if !strings.Contains(err.Error(), "max retries exceeded") {
		t.Errorf("expected 'max retries exceeded' in error, got %v", err)
	}
}

func TestNodeRetryHandler_ExecuteWithRetry_EventualSuccess(t *testing.T) {
	nrh := NewNodeRetryHandler(3, 1*time.Millisecond)

	attempts := 0
	err := nrh.ExecuteWithRetry(func(nodeID string) error {
		attempts++
		if attempts < 3 {
			return &testError{msg: "temporary failure"}
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error on eventual success, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestRateLimitHeaders_New(t *testing.T) {
	rlh := NewRateLimitHeaders(true)
	if rlh == nil {
		t.Fatal("NewRateLimitHeaders(true) returned nil")
	}
	if !rlh.Enabled {
		t.Error("expected enabled")
	}
	if rlh.HeaderLimit != "X-RateLimit-Limit" {
		t.Errorf("expected default HeaderLimit, got %s", rlh.HeaderLimit)
	}
}

func TestRateLimitHeaders_New_Disabled(t *testing.T) {
	rlh := NewRateLimitHeaders(false)
	if rlh.Enabled {
		t.Error("expected disabled")
	}
}

func TestRateLimitHeaders_SetHeaders_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rlh := NewRateLimitHeaders(true)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	rlh.SetHeaders(c, 100, 50, time.Now().Add(time.Minute))

	if c.Writer.Header().Get("X-RateLimit-Limit") != "100" {
		t.Error("expected X-RateLimit-Limit 100")
	}
	if c.Writer.Header().Get("X-RateLimit-Remaining") != "50" {
		t.Error("expected X-RateLimit-Remaining 50")
	}
	if c.Writer.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset header")
	}
}

func TestRateLimitHeaders_SetHeaders_Disabled_Cache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rlh := NewRateLimitHeaders(false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	rlh.SetHeaders(c, 100, 0, time.Now().Add(time.Minute))

	if c.Writer.Header().Get("X-RateLimit-Limit") != "" {
		t.Error("expected no headers when disabled")
	}
}

func TestRateLimitHeaders_SetHeaders_RetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rlh := NewRateLimitHeaders(true)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	rlh.SetHeaders(c, 100, 0, time.Now().Add(30*time.Second))

	if c.Writer.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header when remaining is 0")
	}
}

func TestRateLimitHeaders_Middleware_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rlh := NewRateLimitHeaders(true)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	rlh.Middleware()(c)
	c.Next()

	if c.Writer.Header().Get("X-RateLimit-Limit") != "100" {
		t.Error("expected X-RateLimit-Limit 100 from middleware")
	}
}

func TestRateLimitHeaders_Middleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rlh := NewRateLimitHeaders(false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	rlh.Middleware()(c)
	c.Next()

	if c.Writer.Header().Get("X-RateLimit-Limit") != "" {
		t.Error("expected no headers when middleware disabled")
	}
}
