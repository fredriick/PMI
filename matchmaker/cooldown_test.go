package matchmaker

import (
	"context"
	"testing"
	"time"

	"proxymesh/internal/config"
)

func newTestCooldownRedis(t *testing.T) (*RedisClient, func()) {
	rdb := redisClient(t)
	rc := &RedisClient{
		client: rdb,
		ctx:    context.Background(),
	}
	return rc, func() { rdb.Close() }
}

func TestCooldownService_SetGetTTL(t *testing.T) {
	rc, cleanup := newTestCooldownRedis(t)
	defer cleanup()

	initialTTL := 15 * time.Minute
	cs := NewCooldownService(rc, initialTTL)

	if cs.GetTTL() != initialTTL {
		t.Errorf("expected TTL %v, got %v", initialTTL, cs.GetTTL())
	}

	newTTL := 30 * time.Minute
	cs.SetTTL(newTTL)

	if cs.GetTTL() != newTTL {
		t.Errorf("expected TTL %v, got %v", newTTL, cs.GetTTL())
	}
}

func TestCooldownService_AddAndGet(t *testing.T) {
	rc, cleanup := newTestCooldownRedis(t)
	defer cleanup()

	ttl := 10 * time.Minute
	cs := NewCooldownService(rc, ttl)

	target := "example.com"
	nodeID := "node1"

	err := cs.AddCooldown(target, nodeID)
	if err != nil {
		t.Fatalf("AddCooldown failed: %v", err)
	}

	inCooldown, err := cs.IsInCooldown(target, nodeID)
	if err != nil {
		t.Fatalf("IsInCooldown failed: %v", err)
	}
	if !inCooldown {
		t.Error("expected node to be in cooldown after AddCooldown")
	}

	inCooldown, err = cs.IsInCooldown(target, "other-node")
	if err != nil {
		t.Fatalf("IsInCooldown failed: %v", err)
	}
	if inCooldown {
		t.Error("expected other node to NOT be in cooldown")
	}
}

func TestCooldownService_NoRedis(t *testing.T) {
	cs := NewCooldownService(nil, 15*time.Minute)

	if cs.GetTTL() != 15*time.Minute {
		t.Errorf("expected TTL 15m, got %v", cs.GetTTL())
	}
}

func TestCooldownService_Stop(t *testing.T) {
	ttl := 10 * time.Minute
	cs := NewCooldownService(nil, ttl)

	cs.SetTTL(20 * time.Minute)

	cs.Stop()

	if cs.GetTTL() != 20*time.Minute {
		t.Errorf("expected TTL preserved after Stop, got %v", cs.GetTTL())
	}
}

func TestCooldownService_SetTTL_Concurrent(t *testing.T) {
	cs := NewCooldownService(nil, 10*time.Minute)

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			cs.SetTTL(5 * time.Minute)
			cs.GetTTL()
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	if cs.GetTTL() != 5*time.Minute {
		t.Errorf("expected TTL 5m, got %v", cs.GetTTL())
	}
}

func TestNewRedisClient_InvalidConfig(t *testing.T) {
	_, err := NewRedisClient(&config.RedisConfig{
		Host: "invalid-host-that-does-not-exist",
		Port: 6379,
	})
	if err == nil {
		t.Error("expected error for invalid Redis config")
	}
}
