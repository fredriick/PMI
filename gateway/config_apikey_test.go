package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"proxymesh/internal/config"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	_ = rdb.FlushDB(context.Background()).Err()
	return rdb
}

func TestGetConfigHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	gw := &Gateway{config: &config.GatewayConfig{Host: "test"}}
	gw.getConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestUpdateConfigHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"host":"test"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	gw := &Gateway{}
	gw.updateConfigHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestAPIKeyService_CreateAndValidate(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)

	key, err := svc.CreateKey("test-key", 1, string(ScopeRead))
	if err != nil {
		t.Fatalf("CreateKey failed: %v", err)
	}
	if key.Name != "test-key" {
		t.Fatalf("expected name test-key, got %s", key.Name)
	}
	if key.Scope != string(ScopeRead) {
		t.Fatalf("expected scope read, got %s", key.Scope)
	}

	scope, err := svc.ValidateKey(key.Key)
	if err != nil {
		t.Fatalf("ValidateKey failed: %v", err)
	}
	if scope != string(ScopeRead) {
		t.Fatalf("expected scope read, got %s", scope)
	}
}

func TestAPIKeyService_HasScope(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)

	tests := []struct {
		scope    string
		required string
		expected bool
	}{
		{string(ScopeRead), string(ScopeRead), true},
		{string(ScopeWrite), string(ScopeRead), false},
		{string(ScopeAll), string(ScopeRead), true},
		{string(ScopeAll), string(ScopeAdmin), true},
	}

	for _, tt := range tests {
		key, _ := svc.CreateKey("scope-test", 1, tt.scope)
		if got := svc.HasScope(key.Key, tt.required); got != tt.expected {
			t.Errorf("HasScope(%s, %s) = %v, want %v", tt.scope, tt.required, got, tt.expected)
		}
	}
}

func TestAPIKeyService_RevokeKey(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, _ := svc.CreateKey("revoke-test", 1, string(ScopeRead))

	if err := svc.RevokeKey(key.Key); err != nil {
		t.Fatalf("RevokeKey failed: %v", err)
	}

	scope, err := svc.ValidateKey(key.Key)
	if err != nil {
		t.Fatalf("ValidateKey after revoke failed: %v", err)
	}
	if scope != "" {
		t.Fatalf("expected empty scope after revoke, got %s", scope)
	}
}

func TestAPIKeyService_ListKeys(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	svc.CreateKey("list-1", 1, string(ScopeRead))
	svc.CreateKey("list-2", 1, string(ScopeWrite))

	keys, err := svc.ListKeys()
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestAPIKeyService_RateLimit(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, _ := svc.CreateKey("rate-test", 1, string(ScopeRead))

	if err := svc.SetKeyRateLimit(key.Key, 100, 60); err != nil {
		t.Fatalf("SetKeyRateLimit failed: %v", err)
	}

	requests, window, err := svc.GetKeyRateLimit(key.Key)
	if err != nil {
		t.Fatalf("GetKeyRateLimit failed: %v", err)
	}
	if requests != 100 || window != 60 {
		t.Fatalf("expected 100/60, got %d/%d", requests, window)
	}
}

func TestAPIKeyService_ExpiredKey(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, _ := svc.CreateKey("expired-test", -1, string(ScopeRead))

	scope, err := svc.ValidateKey(key.Key)
	if err != nil {
		t.Fatalf("ValidateKey failed: %v", err)
	}
	if scope != "" {
		t.Fatalf("expected empty scope for expired key, got %s", scope)
	}
}
