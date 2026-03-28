package gateway

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowWithinLimit(t *testing.T) {
	rl := NewRateLimiter(5, 60)

	for i := 0; i < 5; i++ {
		if !rl.Allow("client1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiter_BlockOverLimit(t *testing.T) {
	rl := NewRateLimiter(3, 60)

	for i := 0; i < 3; i++ {
		rl.Allow("client1")
	}

	if rl.Allow("client1") {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiter_DifferentClients(t *testing.T) {
	rl := NewRateLimiter(2, 60)

	rl.Allow("client1")
	rl.Allow("client1")

	if !rl.Allow("client2") {
		t.Error("different client should have independent limit")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(2, 1) // 1 second window

	rl.Allow("client1")
	rl.Allow("client1")

	if rl.Allow("client1") {
		t.Error("should be blocked within window")
	}

	time.Sleep(1100 * time.Millisecond)

	if !rl.Allow("client1") {
		t.Error("should be allowed after window reset")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimiter(1000, 60)
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

	// After 2000 attempts with limit 1000, next should be blocked
	if rl.Allow("shared-client") {
		t.Error("should be blocked after exceeding limit under concurrent load")
	}
}
