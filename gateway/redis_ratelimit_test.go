package gateway

import (
	"testing"
	"time"
)

func TestDistributedRateLimiter_New(t *testing.T) {
	rl := NewDistributedRateLimiter(nil, 100, 60)
	if rl == nil {
		t.Fatal("NewDistributedRateLimiter returned nil")
	}
	if rl.limit != 100 {
		t.Errorf("expected limit 100, got %d", rl.limit)
	}
	if rl.window != time.Minute {
		t.Errorf("expected window 1m, got %v", rl.window)
	}
}

func TestDistributedRateLimiter_SetGetLimit(t *testing.T) {
	rl := NewDistributedRateLimiter(nil, 10, 30)
	rl.SetLimit(50)
	if rl.GetLimit() != 50 {
		t.Errorf("expected limit 50, got %d", rl.GetLimit())
	}
}

func TestDistributedRateLimiter_SetWindow(t *testing.T) {
	rl := NewDistributedRateLimiter(nil, 10, 30)
	rl.SetWindow(2 * time.Minute)
	if rl.window != 2*time.Minute {
		t.Errorf("expected window 2m, got %v", rl.window)
	}
}

func TestDistributedRateLimiter_Allow(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	rl := NewDistributedRateLimiter(client, 100, 60)
	allowed, err := rl.Allow("client-1")
	if err != nil {
		t.Fatalf("Allow error: %v", err)
	}
	if !allowed {
		t.Error("expected first request allowed")
	}
}

func TestDistributedRateLimiter_Allow_WithRedis(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	rl := NewDistributedRateLimiter(client, 1000, 60)
	for i := 0; i < 5; i++ {
		allowed, err := rl.Allow("client-2")
		if err != nil {
			t.Fatalf("Allow error: %v", err)
		}
		if !allowed {
			t.Errorf("expected request %d allowed", i+1)
		}
	}
}
