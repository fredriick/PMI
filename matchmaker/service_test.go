package matchmaker

import (
	"sync"
	"testing"

	"proxymesh/internal/models"
)

func newTestMatchmaker() *Matchmaker {
	return &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}
}

func TestRecordFailure_OpensCircuit(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	if !mm.isNodeHealthy("node1") {
		t.Error("node should still be healthy after 2 failures (threshold=3)")
	}

	mm.RecordFailure("node1")
	if mm.isNodeHealthy("node1") {
		t.Error("node should be unhealthy after 3 failures (threshold=3)")
	}
}

func TestRecordSuccess_ClosesCircuit(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	if mm.isNodeHealthy("node1") {
		t.Error("node should be unhealthy")
	}

	mm.RecordSuccess("node1")

	if !mm.isNodeHealthy("node1") {
		t.Error("node should be healthy after success")
	}
}

func TestIsNodeHealthy_UnknownNode(t *testing.T) {
	mm := newTestMatchmaker()

	if !mm.isNodeHealthy("unknown") {
		t.Error("unknown node should be considered healthy")
	}
}

func TestRecordFailure_IndependentNodes(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	if !mm.isNodeHealthy("node2") {
		t.Error("node2 should not be affected by node1 failures")
	}
}

func TestRecordFailure_Concurrent(t *testing.T) {
	mm := newTestMatchmaker()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mm.RecordFailure("node1")
		}()
	}
	wg.Wait()

	if mm.isNodeHealthy("node1") {
		t.Error("node should be unhealthy after many concurrent failures")
	}
}

func TestRandomSelect_SingleNode(t *testing.T) {
	mm := newTestMatchmaker()

	selected, err := mm.randomSelect([]string{"node1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "node1" {
		t.Errorf("expected node1, got %s", selected)
	}
}

func TestRandomSelect_EmptyList(t *testing.T) {
	mm := newTestMatchmaker()

	_, err := mm.randomSelect([]string{})
	if err == nil {
		t.Error("expected error for empty node list")
	}
}

func TestRandomSelect_FromMultiple(t *testing.T) {
	mm := newTestMatchmaker()
	nodes := []string{"a", "b", "c", "d", "e"}
	counts := make(map[string]int)

	for i := 0; i < 1000; i++ {
		selected, err := mm.randomSelect(nodes)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[selected]++
	}

	for _, node := range nodes {
		if counts[node] == 0 {
			t.Errorf("node %s was never selected in 1000 iterations", node)
		}
	}
}

func TestRegisterNode_Validation(t *testing.T) {
	mm := &Matchmaker{}

	tests := []struct {
		name    string
		req     *models.NodeRegistrationRequest
		wantErr bool
	}{
		{"empty node_id", &models.NodeRegistrationRequest{Country: "US", IP: "1.2.3.4"}, true},
		{"empty country", &models.NodeRegistrationRequest{NodeID: "n1", IP: "1.2.3.4"}, true},
		{"empty ip", &models.NodeRegistrationRequest{NodeID: "n1", Country: "US"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mm.RegisterNode(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeartbeat_Validation(t *testing.T) {
	mm := &Matchmaker{}

	err := mm.Heartbeat(&models.NodeHeartbeatRequest{NodeID: ""})
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestDeregisterNode_Validation(t *testing.T) {
	mm := &Matchmaker{}

	err := mm.DeregisterNode("")
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestGetSessionNode_Validation(t *testing.T) {
	mm := &Matchmaker{}

	_, err := mm.GetSessionNode("")
	if err == nil {
		t.Error("expected error for empty session_id")
	}
}

func TestSetSessionNode_Validation(t *testing.T) {
	mm := &Matchmaker{}

	err := mm.SetSessionNode("", "node1", 3600)
	if err == nil {
		t.Error("expected error for empty session_id")
	}

	err = mm.SetSessionNode("sess1", "", 3600)
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}
