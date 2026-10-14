package gateway

import (
	"testing"
	"time"
)

func TestAPIKeyService_New(t *testing.T) {
	svc := NewAPIKeyService(nil)
	if svc == nil {
		t.Fatal("NewAPIKeyService returned nil")
	}
	if svc.ctx == nil {
		t.Error("expected ctx to be initialized")
	}
}

func TestAPIKeyService_CreateKey(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, err := svc.CreateKey("test-key", 0, "read")
	if err != nil {
		t.Fatalf("CreateKey error: %v", err)
	}
	if key.Key == "" {
		t.Error("expected non-empty key")
	}
	if key.Scope != "read" {
		t.Errorf("expected scope 'read', got %q", key.Scope)
	}
}

func TestAPIKeyService_CreateKey_InvalidScopeDefaultsToRead(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, err := svc.CreateKey("test-key", 0, "invalid")
	if err != nil {
		t.Fatalf("CreateKey error: %v", err)
	}
	if key.Scope != "read" {
		t.Errorf("expected default scope 'read', got %q", key.Scope)
	}
}

func TestAPIKeyScope_Constants(t *testing.T) {
	if ScopeRead != "read" {
		t.Errorf("expected ScopeRead='read', got %q", ScopeRead)
	}
	if ScopeWrite != "write" {
		t.Errorf("expected ScopeWrite='write', got %q", ScopeWrite)
	}
	if ScopeAdmin != "admin" {
		t.Errorf("expected ScopeAdmin='admin', got %q", ScopeAdmin)
	}
	if ScopeAll != "*" {
		t.Errorf("expected ScopeAll='*', got %q", ScopeAll)
	}
}

func TestAPIKeyService_HasMinScope_NoRedis(t *testing.T) {
	svc := NewAPIKeyService(nil)
	if svc.HasMinScope("some-key", "read") {
		t.Error("expected HasMinScope to return false without Redis")
	}
}

func TestAPIKeyService_HasScope_NoRedis(t *testing.T) {
	svc := NewAPIKeyService(nil)
	if svc.HasScope("some-key", "read") {
		t.Error("expected HasScope to return false without Redis")
	}
}

func TestAPIKeyService_ValidateKey_NilService(t *testing.T) {
	var svc *APIKeyService
	_, err := svc.ValidateKey("key")
	if err == nil {
		t.Error("expected error for nil service")
	}
}

func TestAPIKeyService_SetAndGetKeyRateLimit(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, _ := svc.CreateKey("rate-key", 0, "read")

	err := svc.SetKeyRateLimit(key.Key, 100, 60)
	if err != nil {
		t.Fatalf("SetKeyRateLimit error: %v", err)
	}

	reqs, win, err := svc.GetKeyRateLimit(key.Key)
	if err != nil {
		t.Fatalf("GetKeyRateLimit error: %v", err)
	}
	if reqs != 100 {
		t.Errorf("expected 100 requests, got %d", reqs)
	}
	if win != 60 {
		t.Errorf("expected window 60, got %d", win)
	}
}

func TestAPIKeyService_GetKeyHash(t *testing.T) {
	svc := NewAPIKeyService(nil)
	h1 := svc.GetKeyHash("my-key")
	h2 := svc.GetKeyHash("my-key")
	if h1 != h2 {
		t.Error("expected same hash for same key")
	}
	if h1 == "" {
		t.Error("expected non-empty hash")
	}
}

func TestAPIKeyService_ValidateKey_NilClient(t *testing.T) {
	svc := &APIKeyService{}
	_, err := svc.ValidateKey("key")
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestAPIKeyService_UpdateScope(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, _ := svc.CreateKey("scope-key", 0, "read")

	err := svc.UpdateScope(svc.GetKeyHash(key.Key), "admin")
	if err != nil {
		t.Fatalf("UpdateScope error: %v", err)
	}

	scope, err := svc.ValidateKey(key.Key)
	if err != nil {
		t.Fatalf("ValidateKey error: %v", err)
	}
	if scope != "admin" {
		t.Errorf("expected scope 'admin', got %q", scope)
	}
}

func TestAPIKeyService_UpdateScope_Invalid(t *testing.T) {
	svc := NewAPIKeyService(nil)
	err := svc.UpdateScope("hash", "invalid")
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestAPIKeyService_CreateKey_WithTTL(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	svc := NewAPIKeyService(client)
	key, err := svc.CreateKey("ttl-key", 1, "write")
	if err != nil {
		t.Fatalf("CreateKey error: %v", err)
	}
	if key.ExpiresAt.IsZero() {
		t.Error("expected non-zero ExpiresAt")
	}
	if key.Scope != "write" {
		t.Errorf("expected scope 'write', got %q", key.Scope)
	}
	_ = time.Now()
}
