package matchmaker

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSessionPersistence_CreateAndRetrieve(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	if err := sp.CreateSession("sess1", "node1", "user1", "node2"); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	nodeID, ok := sp.GetNodeForSession("sess1")
	if !ok {
		t.Fatal("expected session to be found")
	}
	if nodeID != "node1" {
		t.Errorf("expected node1, got %s", nodeID)
	}
}

func TestSessionPersistence_RetrieveNonexistent(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	nodeID, ok := sp.GetNodeForSession("nonexistent")
	if ok {
		t.Errorf("expected false for nonexistent session, got %s", nodeID)
	}
}

func TestSessionPersistence_GetFailoverNode(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.CreateSession("sess1", "node1", "user1", "node2")

	if sp.GetFailoverNode("nonexistent") != "" {
		t.Error("expected empty failover for nonexistent session")
	}

	failover := sp.GetFailoverNode("sess1")
	if failover != "node2" {
		t.Errorf("expected failover 'node2', got '%s'", failover)
	}
}

func TestSessionPersistence_Failover(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.CreateSession("sess1", "node1", "user1", "node2")

	if sp.Failover("nonexistent") {
		t.Error("expected false for failover on nonexistent session")
	}

	if !sp.Failover("sess1") {
		t.Error("expected true for successful failover")
	}

	nodeID, ok := sp.GetNodeForSession("sess1")
	if !ok {
		t.Fatal("expected session to still exist after failover")
	}
	if nodeID != "node2" {
		t.Errorf("expected node2 after failover, got %s", nodeID)
	}

	if sp.Failover("sess1") {
		t.Error("expected false for second failover (no more failover nodes)")
	}
}

func TestSessionPersistence_Failover_NoSFailoverNode(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.CreateSession("sess1", "node1", "user1", "")

	if sp.Failover("sess1") {
		t.Error("expected false for failover with no failover node")
	}
}

func TestSessionPersistence_DeleteSession(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.CreateSession("sess1", "node1", "user1", "node2")

	sp.DeleteSession("sess1")

	_, ok := sp.GetNodeForSession("sess1")
	if ok {
		t.Error("expected session to be deleted")
	}
}

func TestSessionPersistence_DeleteNonexistent(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.DeleteSession("nonexistent")
}

func TestSerializeSession(t *testing.T) {
	s := &SessionInfo{
		SessionID:    "sess1",
		NodeID:       "node1",
		UserID:       "user1",
		CreatedAt:    time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:    time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC),
		FailoverNode: "node2",
	}

	result, err := serializeSession(s)
	if err != nil {
		t.Fatalf("serializeSession failed: %v", err)
	}

	expected := "sess1|node1|user1|1577836800|1577923200|node2"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestSessionPersistence_ExpiredSession(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	sp.CreateSession("sess1", "node1", "user1", "node2")

	sp.mu.Lock()
	sp.sessions["sess1"].ExpiresAt = time.Now().Add(-1 * time.Hour)
	sp.mu.Unlock()

	_, ok := sp.GetNodeForSession("sess1")
	if ok {
		t.Error("expected expired session to not be found")
	}
}

func TestSessionPersistence_ConcurrentAccess(t *testing.T) {
	rc, cleanup := newTestCooldownRedis(t)
	defer cleanup()

	rdb := rc.client
	ctx := context.Background()
	sp := &SessionPersistence{
		redis:      rdb,
		ctx:        ctx,
		sessions:   make(map[string]*SessionInfo),
		defaultTTL: 1 * time.Hour,
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sp.CreateSession(string(rune(n)), "node1", "user1", "node2")
			sp.GetNodeForSession(string(rune(n)))
			sp.DeleteSession(string(rune(n)))
		}(i)
	}
	wg.Wait()
}

func TestSessionPersistence_MaxEntries(t *testing.T) {
	sp := NewSessionPersistence(nil, 1*time.Hour)

	for i := 0; i < 200; i++ {
		sp.CreateSession(string(rune(i)), "node1", "user1", "")
	}

	sp.mu.RLock()
	count := len(sp.sessions)
	sp.mu.RUnlock()

	if count != 200 {
		t.Errorf("expected 200 sessions stored, got %d", count)
	}
}
