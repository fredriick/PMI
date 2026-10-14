package gateway

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewRequestDedup_DefaultTTL(t *testing.T) {
	rd := NewRequestDedup(0)
	if rd == nil {
		t.Fatal("NewRequestDedup(0) returned nil")
	}
	if rd.ttl != 30*time.Second {
		t.Errorf("expected default TTL 30s, got %v", rd.ttl)
	}
}

func TestNewRequestDedup_CustomTTL(t *testing.T) {
	rd := NewRequestDedup(5 * time.Second)
	if rd.ttl != 5*time.Second {
		t.Errorf("expected TTL 5s, got %v", rd.ttl)
	}
}

func TestRequestDedup_ComputeHash(t *testing.T) {
	rd := NewRequestDedup(30 * time.Second)
	hash := rd.ComputeHash("GET", "/api/test", "body", "user1")

	if len(hash) != 64 {
		t.Errorf("expected SHA256 hash length 64, got %d", len(hash))
	}
}

func TestRequestDedup_ComputeHash_Deterministic(t *testing.T) {
	rd := NewRequestDedup(30 * time.Second)
	hash1 := rd.ComputeHash("GET", "/api/test", "body", "user1")
	hash2 := rd.ComputeHash("GET", "/api/test", "body", "user1")

	if hash1 != hash2 {
		t.Error("expected deterministic hash for same inputs")
	}
}

func TestRequestDedup_CheckAndMark_New(t *testing.T) {
	rd := NewRequestDedup(30 * time.Second)
	isNew, nodeID := rd.CheckAndMark("hash1", "node-1")

	if !isNew {
		t.Error("expected new request")
	}
	if nodeID != "" {
		t.Errorf("expected empty node ID for new request, got %q", nodeID)
	}
}

func TestRequestDedup_CheckAndMark_DuplicateWithinTTL(t *testing.T) {
	rd := NewRequestDedup(30 * time.Second)

	rd.CheckAndMark("hash1", "node-1")
	isNew, existingNode := rd.CheckAndMark("hash1", "node-2")

	if isNew {
		t.Error("expected duplicate request within TTL")
	}
	if existingNode != "node-1" {
		t.Errorf("expected existing node ID node-1, got %q", existingNode)
	}
}

func TestRequestDedup_CheckAndMark_Expired(t *testing.T) {
	rd := NewRequestDedup(1 * time.Nanosecond)

	rd.CheckAndMark("hash1", "node-1")
	time.Sleep(10 * time.Nanosecond)
	isNew, existingNode := rd.CheckAndMark("hash1", "node-2")

	if !isNew {
		t.Error("expected new request after TTL expiration")
	}
	if existingNode != "" {
		t.Errorf("expected empty node ID for new request after expiry, got %q", existingNode)
	}
}

func TestRequestDedup_Middleware_DedupHit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rd := NewRequestDedup(30 * time.Second)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Node-ID", "node-1")
	c.Request.Header.Set("X-User", "user1")

	rd.Middleware()(c)
	c.Next()

	if c.Writer.Header().Get("X-Dedup") != "MISS" {
		t.Error("expected X-Dedup MISS for first request")
	}

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/api/test", nil)
	c2.Request.Header.Set("X-Node-ID", "node-2")
	c2.Request.Header.Set("X-User", "user1")

	rd.Middleware()(c2)
	c2.Next()

	if c2.Writer.Header().Get("X-Dedup") != "HIT" {
		t.Error("expected X-Dedup HIT for duplicate request")
	}
	if c2.Writer.Header().Get("X-Dedup-Node") != "node-1" {
		t.Errorf("expected X-Dedup-Node node-1, got %q", c2.Writer.Header().Get("X-Dedup-Node"))
	}
}

func TestRequestDedup_Middleware_DedupMiss(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rd := NewRequestDedup(30 * time.Second)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	rd.Middleware()(c)
	c.Next()

	if c.Writer.Header().Get("X-Dedup") != "MISS" {
		t.Error("expected X-Dedup MISS for first request")
	}
}
