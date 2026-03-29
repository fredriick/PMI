package gateway

import (
	"sync"
	"testing"
	"time"
)

func TestLocalRateLimiter_AllowWithinLimit(t *testing.T) {
	rl := NewLocalRateLimiter(5, 60)

	for i := 0; i < 5; i++ {
		allowed, err := rl.Allow("client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestLocalRateLimiter_BlockOverLimit(t *testing.T) {
	rl := NewLocalRateLimiter(3, 60)

	for i := 0; i < 3; i++ {
		rl.Allow("client1")
	}

	allowed, _ := rl.Allow("client1")
	if allowed {
		t.Error("4th request should be blocked")
	}
}

func TestLocalRateLimiter_DifferentClients(t *testing.T) {
	rl := NewLocalRateLimiter(2, 60)

	rl.Allow("client1")
	rl.Allow("client1")

	allowed, _ := rl.Allow("client2")
	if !allowed {
		t.Error("different client should have independent limit")
	}
}

func TestLocalRateLimiter_WindowReset(t *testing.T) {
	rl := NewLocalRateLimiter(2, 1) // 1 second window

	rl.Allow("client1")
	rl.Allow("client1")

	allowed, _ := rl.Allow("client1")
	if allowed {
		t.Error("should be blocked within window")
	}

	time.Sleep(1100 * time.Millisecond)

	allowed, _ = rl.Allow("client1")
	if !allowed {
		t.Error("should be allowed after window reset")
	}
}

func TestLocalRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewLocalRateLimiter(1000, 60)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				rl.Allow("shared-client")
			}
		}()
	}
	wg.Wait()

	allowed, _ := rl.Allow("shared-client")
	if allowed {
		t.Error("should be blocked after exceeding limit under concurrent load")
	}
}

func TestLocalRateLimiter_AllowReturnsError(t *testing.T) {
	rl := NewLocalRateLimiter(10, 60)
	allowed, err := rl.Allow("test")
	if err != nil {
		t.Errorf("local limiter should not return error, got: %v", err)
	}
	if !allowed {
		t.Error("should be allowed")
	}
}
