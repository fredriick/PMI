package gateway

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestTrafficShaper_New(t *testing.T) {
	ts := NewTrafficShaper(nil, 1000, 500)
	if ts == nil {
		t.Fatal("NewTrafficShaper returned nil")
	}
	if ts.defaultRate != 1000 {
		t.Errorf("expected defaultRate 1000, got %d", ts.defaultRate)
	}
	if ts.defaultBurst != 500 {
		t.Errorf("expected defaultBurst 500, got %d", ts.defaultBurst)
	}
}

func TestTrafficShaper_CheckBandwidth(t *testing.T) {
	ts := NewTrafficShaper(nil, 1000, 100)
	allowed, err := ts.checkBandwidth("10.0.0.1", 10)
	if err != nil {
		t.Fatalf("checkBandwidth error: %v", err)
	}
	if !allowed {
		t.Error("expected bandwidth check allowed")
	}
}

func TestTrafficShaper_SetClientRate(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ts := NewTrafficShaper(client, 1000, 500)
	ts.SetClientRate("10.0.0.1", 2000, 1000)
}

func TestTrafficShaper_IsWhitelisted(t *testing.T) {
	ts := NewTrafficShaper(nil, 1000, 500)
	if !ts.isWhitelisted("127.0.0.1") {
		t.Error("expected loopback to be whitelisted")
	}
}

func TestDDoSProtection_New(t *testing.T) {
	ddos := NewDDoSProtection(nil, 5, 60, time.Minute)
	if ddos == nil {
		t.Fatal("NewDDoSProtection returned nil")
	}
	if ddos.threshold != 5 {
		t.Errorf("expected threshold 5, got %d", ddos.threshold)
	}
	if ddos.windowSeconds != 60 {
		t.Errorf("expected windowSeconds 60, got %d", ddos.windowSeconds)
	}
}

func TestDDoSProtection_RecordFailure(t *testing.T) {
	ddos := NewDDoSProtection(nil, 3, 60, time.Minute)
	ddos.RecordFailure("10.0.0.1")
	ddos.RecordFailure("10.0.0.1")
	if ddos.isBlocked("10.0.0.1") {
		t.Error("expected not blocked after 2 failures (threshold 3)")
	}
	ddos.RecordFailure("10.0.0.1")
	if !ddos.isBlocked("10.0.0.1") {
		t.Error("expected blocked after threshold failures")
	}
}

func TestDDoSProtection_BlockUnblockIP(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ddos := NewDDoSProtection(client, 5, 60, time.Minute)
	ddos.BlockIP("10.0.0.2")
	if !ddos.isBlocked("10.0.0.2") {
		t.Error("expected IP to be blocked")
	}
	ddos.UnblockIP("10.0.0.2")
}

func TestDDoSProtection_Whitelist(t *testing.T) {
	ddos := NewDDoSProtection(nil, 5, 60, time.Minute)
	ddos.AddToWhitelist("10.0.0.3")
	if !ddos.isWhitelisted("10.0.0.3") {
		t.Error("expected IP to be in whitelist")
	}
	ddos.RemoveFromWhitelist("10.0.0.3")
	if ddos.isWhitelisted("10.0.0.3") {
		t.Error("expected IP to be removed from whitelist")
	}
}

func TestDDoSProtection_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ddos := NewDDoSProtection(nil, 5, 60, time.Minute)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	ddos.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request to pass through")
	}
}
