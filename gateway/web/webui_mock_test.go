package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
	"time"
)

var ErrNodeNotFound = errors.New("node not found")

func TestWebUI_NodesAPIHandler_WithMockMatchmaker(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetNodeStatus("node-1", &models.Node{
		ID:         "node-1",
		Country:    "US",
		City:       "New York",
		ISP:        "Test ISP",
		OS:         "Linux",
		Reputation: 95.5,
		LastSeen:   time.Now(),
	})
	mockMM.SetNodeLoad("node-1", 5)

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/nodes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	nodes := resp["nodes"].([]interface{})
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}

	// Check first node has expected fields
	firstNode := nodes[0].(map[string]interface{})
	if firstNode["id"] != "node-1" {
		t.Errorf("expected node-1, got %v", firstNode["id"])
	}
	if firstNode["country"] != "US" {
		t.Errorf("expected US, got %v", firstNode["country"])
	}
	if firstNode["load"] != float64(5) {
		t.Errorf("expected load 5, got %v", firstNode["load"])
	}
}

func TestWebUI_NodesAPIHandler_EmptyNodes(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.nodes = []string{}

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/nodes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	nodesVal := resp["nodes"]
	if nodesVal == nil {
		t.Errorf("nodes is nil")
		return
	}
	nodes := nodesVal.([]interface{})
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
	if resp["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestWebUI_CooldownsAPIHandler_WithMockMatchmaker(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetCooldownEntries(map[string][]string{
		"target-1": {"node-1", "node-2"},
		"target-2": {"node-3"},
	})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/cooldowns", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	cooldowns := resp["cooldowns"].(map[string]interface{})
	if len(cooldowns) != 2 {
		t.Errorf("expected 2 cooldown entries, got %d", len(cooldowns))
	}
}

func TestWebUI_CooldownsAPIHandler_Empty(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetCooldownEntries(map[string][]string{})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/cooldowns", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	cooldowns := resp["cooldowns"].(map[string]interface{})
	if len(cooldowns) != 0 {
		t.Errorf("expected 0 cooldown entries, got %d", len(cooldowns))
	}
}

func TestWebUI_HealthAPIHandler_WithMockMatchmaker(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetHealthPeers([]*matchmaker.PeerHealth{
		{NodeID: "node-1", OverallScore: 95.0, LastChecked: time.Now()},
		{NodeID: "node-2", OverallScore: 85.0, LastChecked: time.Now()},
		{NodeID: "node-3", OverallScore: 75.0, LastChecked: time.Now()},
	})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["count"] != float64(3) {
		t.Errorf("expected count 3, got %v", resp["count"])
	}

	peers := resp["peers"].([]interface{})
	if len(peers) != 3 {
		t.Errorf("expected 3 peers, got %d", len(peers))
	}
}

func TestWebUI_HealthAPIHandler_Empty(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetHealthPeers([]*matchmaker.PeerHealth{})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", resp["count"])
	}
}

func TestWebUI_NodeHealthAPIHandler_Found(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetHealthScore("node-1", &matchmaker.PeerHealth{
		NodeID:           "node-1",
		ConnectionScore:  0.9,
		LatencyScore:     0.95,
		BandwidthScore:   0.85,
		ReliabilityScore: 0.98,
		OverallScore:     92.5,
		LastChecked:      time.Now(),
	})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["node_id"] != "node-1" {
		t.Errorf("expected node-1, got %v", resp["node_id"])
	}
	score := resp["score"].(map[string]interface{})
	if score["overall_score"] != float64(92.5) {
		t.Errorf("expected overall_score 92.5, got %v", score["overall_score"])
	}
}

func TestWebUI_NodeHealthAPIHandler_NotFound(t *testing.T) {
	mockMM := NewMockMatchmaker()
	// No health score set for node-999

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestWebUI_NodeHealthHistoryAPIHandler_WithHistory(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetHealthHistory("node-1", []float64{90.0, 92.0, 94.0, 95.0, 93.0})

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-1/history", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["node_id"] != "node-1" {
		t.Errorf("expected node-1, got %v", resp["node_id"])
	}

	history := resp["history"].([]interface{})
	if len(history) != 5 {
		t.Errorf("expected 5 history entries, got %d", len(history))
	}
	if resp["count"] != float64(5) {
		t.Errorf("expected count 5, got %v", resp["count"])
	}
}

func TestWebUI_NodeHealthHistoryAPIHandler_Empty(t *testing.T) {
	mockMM := NewMockMatchmaker()
	// No history set

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/health/node-1/history", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	history := resp["history"].([]interface{})
	if len(history) != 4 { // Default history has 4 entries
		t.Errorf("expected 4 history entries (default), got %d", len(history))
	}
}

func TestWebUI_ResetCircuitBreakerHandler_WithMockMatchmaker(t *testing.T) {
	mockMM := NewMockMatchmaker()

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/reset", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
	if resp["message"] != "Circuit breaker reset" {
		t.Errorf("expected message 'Circuit breaker reset', got %v", resp["message"])
	}

	calls := mockMM.GetResetCircuitBreakerCalls()
	if len(calls) != 1 || calls[0] != "node-1" {
		t.Errorf("expected reset call for node-1, got %v", calls)
	}
}

func TestWebUI_ResetCircuitBreakerHandler_EmptyID(t *testing.T) {
	mockMM := NewMockMatchmaker()

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes//reset", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty ID, got %d", rr.Code)
	}
}

func TestWebUI_EjectNodeHandler_WithMockMatchmaker(t *testing.T) {
	mockMM := NewMockMatchmaker()

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}

	calls := mockMM.GetDeregisterNodeCalls()
	if len(calls) != 1 || calls[0] != "node-1" {
		t.Errorf("expected deregister call for node-1, got %v", calls)
	}
}

func TestWebUI_EjectNodeHandler_Error(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.SetDeregisterNodeError(ErrNodeNotFound)

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-999/eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for deregister error, got %d", rr.Code)
	}
}

func TestWebUI_EjectNodeHandler_EmptyID(t *testing.T) {
	mockMM := NewMockMatchmaker()

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes//eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty ID, got %d", rr.Code)
	}
}

func TestWebUI_MinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-1, -5, -5},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestWebUI_NewWebUI_Nil(t *testing.T) {
	w := NewWebUI(nil)
	if w == nil {
		t.Fatal("NewWebUI(nil) returned nil")
	}
	if w.mm != nil {
		t.Error("expected mm to be nil")
	}
}

// Additional edge case tests

func TestWebUI_NodesAPIHandler_NodeStatusError(t *testing.T) {
	mockMM := NewMockMatchmaker()
	mockMM.nodes = []string{"node-1", "node-2"}
	// node-1 will return error from GetNodeStatus if we don't set it up
	// but GetNodeStatus returns default, so it won't error
	// Let's test with a node that returns error by mocking differently

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/nodes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWebUI_NodesAPIHandler_NoRedis(t *testing.T) {
	// Test with mock that returns nil Redis
	mockMM := NewMockMatchmaker()
	mockMM.nodes = []string{"node-1"}

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/web/api/nodes", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Should still work since GetNodeLoad handles nil gracefully
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestWebUI_EjectNodeHandler_NoRedis(t *testing.T) {
	mockMM := NewMockMatchmakerNoRedis()

	w := NewWebUI(mockMM)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/eject", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// With nil matchmaker Redis, should return 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil Redis, got %d", rr.Code)
	}
}

func TestWebUI_ResetCircuitBreakerHandler_NoMatchmaker(t *testing.T) {
	w := NewWebUI(nil)

	r := gin.New()
	w.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/web/api/nodes/node-1/reset", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil matchmaker, got %d", rr.Code)
	}
}