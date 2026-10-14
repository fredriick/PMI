package gateway

import (
	"testing"
	"time"
)

func TestRequestDeduplicator_New(t *testing.T) {
	rd := NewRequestDeduplicator(time.Second)
	if rd == nil {
		t.Fatal("NewRequestDeduplicator returned nil")
	}
	if rd.ttl != time.Second {
		t.Errorf("expected ttl 1s, got %v", rd.ttl)
	}
}

func TestRequestDeduplicator_BeginAndComplete(t *testing.T) {
	rd := NewRequestDeduplicator(time.Minute)

	key, done, nodeID, isDup := rd.BeginRequest("GET", "http://a.com", "US", "NY")
	if key == "" {
		t.Error("expected non-empty key")
	}
	if isDup {
		t.Error("expected not duplicate for first request")
	}
	if nodeID != "" {
		t.Error("expected empty nodeID for new request")
	}
	if done == nil {
		t.Error("expected non-nil done channel")
	}

	key2, done2, _, isDup2 := rd.BeginRequest("GET", "http://a.com", "US", "NY")
	if key != key2 {
		t.Error("expected same key for same request")
	}
	if !isDup2 {
		t.Error("expected duplicate for same request")
	}
	if done != done2 {
		t.Error("expected same done channel for duplicate")
	}

	rd.CompleteRequest(key, "node-1", nil)
	if rd.GetNodeForRequest(key) != "" {
		t.Error("expected empty node after completion")
	}
}

func TestRequestDeduplicator_CompleteRequest_UnknownKey(t *testing.T) {
	rd := NewRequestDeduplicator(time.Minute)
	rd.CompleteRequest("unknown-key", "node", nil)
}

func TestRequestDeduplicator_GetNodeForRequest(t *testing.T) {
	rd := NewRequestDeduplicator(time.Minute)
	node := rd.GetNodeForRequest("nonexistent")
	if node != "" {
		t.Error("expected empty string for nonexistent key")
	}
}

func TestRequestDeduplicator_Stats(t *testing.T) {
	rd := NewRequestDeduplicator(time.Minute)
	total, pending := rd.Stats()
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if pending != 0 {
		t.Errorf("expected 0 pending, got %d", pending)
	}

	rd.BeginRequest("GET", "http://a.com", "US", "NY")
	total, pending = rd.Stats()
	if total != 1 {
		t.Errorf("expected 1 total, got %d", total)
	}
}
