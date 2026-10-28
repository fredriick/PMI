package matchmaker

import (
	"testing"
	"time"

	"proxymesh/internal/models"
)

func TestMatchmaker_New(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       5,
		cooldownTTL:     10 * time.Minute,
	}

	if mm.threshold != 5 {
		t.Errorf("expected threshold 5, got %d", mm.threshold)
	}
	if mm.cooldownTTL != 10*time.Minute {
		t.Errorf("expected cooldownTTL 10m, got %v", mm.cooldownTTL)
	}
	if mm.circuitBreakers == nil {
		t.Error("circuitBreakers should be initialized")
	}
}

func TestMatchmaker_GetCircuitBreakers(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	breakers := mm.GetCircuitBreakers()

	if len(breakers) != 1 {
		t.Errorf("expected 1 circuit breaker, got %d", len(breakers))
	}
	if breakers["node1"].FailureCount != 2 {
		t.Errorf("expected failure count 2, got %d", breakers["node1"].FailureCount)
	}
	if breakers["node1"].Threshold != 3 {
		t.Errorf("expected threshold 3, got %d", breakers["node1"].Threshold)
	}
	if breakers["node1"].State != "closed" {
		t.Errorf("expected state closed, got %s", breakers["node1"].State)
	}
}

func TestMatchmaker_GetCircuitBreakers_Empty(t *testing.T) {
	mm := newTestMatchmaker()

	breakers := mm.GetCircuitBreakers()

	if len(breakers) != 0 {
		t.Errorf("expected 0 circuit breakers, got %d", len(breakers))
	}
}

func TestMatchmaker_ResetCircuitBreaker(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	if mm.isNodeHealthy("node1") {
		t.Error("node should be unhealthy before reset")
	}

	mm.ResetCircuitBreaker("node1")

	if !mm.isNodeHealthy("node1") {
		t.Error("node should be healthy after reset")
	}
}

func TestMatchmaker_ResetCircuitBreaker_NonExistent(t *testing.T) {
	mm := newTestMatchmaker()

	mm.ResetCircuitBreaker("nonexistent")

	if !mm.isNodeHealthy("nonexistent") {
		t.Error("non-existent node should be healthy")
	}
}

func TestMatchmaker_SetCooldownTTL_Zero(t *testing.T) {
	mm := newTestMatchmaker()

	mm.SetCooldownTTL(0)

	if mm.GetCooldownTTL() != 0 {
		t.Errorf("expected TTL 0, got %v", mm.GetCooldownTTL())
	}
}

func TestMatchmaker_SelectNode_Validation(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.SelectNode("", "NYC", "")
	if err == nil {
		t.Error("expected error for empty country")
	}

	_, err = mm.SelectNode("US", "", "")
	if err == nil {
		t.Error("expected no error for empty city (optional)")
	}
}

func TestMatchmaker_GetCapacityReport_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetCapacityReport()
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_GetSessionNode_Empty(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetSessionNode("")
	if err == nil {
		t.Error("expected error for empty session_id")
	}
}

func TestMatchmaker_SetSessionNode_Validation(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.SetSessionNode("", "node1", 3600)
	if err == nil {
		t.Error("expected error for empty session_id")
	}

	err = mm.SetSessionNode("sess1", "", 3600)
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestMatchmaker_DeregisterNode_Validation(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.DeregisterNode("")
	if err == nil {
		t.Error("expected error for empty node_id")
	}
}

func TestMatchmaker_GetNodeStatus_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetNodeStatus("node1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_GetAllNodes_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetAllNodes()
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_DecrementNodeLoad_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.DecrementNodeLoad("node1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_RecordBandwidth_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.RecordBandwidth("node1", 1000, 500, 10)
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_ListSessions_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.ListSessions()
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_DeleteSession_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.DeleteSession("sess1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_GetHealthScore_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	score := mm.GetHealthScore("node1")
	if score != nil {
		t.Errorf("expected nil health score without Redis, got %v", score)
	}
}

func TestMatchmaker_GetTopHealthPeers_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	peers := mm.GetTopHealthPeers(10)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers without Redis, got %d", len(peers))
	}
}

func TestMatchmaker_GetHealthScoreHistory_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	history := mm.GetHealthScoreHistory("node1")
	if len(history) != 0 {
		t.Errorf("expected 0 history entries without Redis, got %d", len(history))
	}
}

func TestMatchmaker_UpdateHealthScore_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	mm.UpdateHealthScore("node1", 100, 50.0, 0.95)

	score := mm.GetHealthScore("node1")
	if score != nil {
		t.Errorf("expected nil health score without Redis, got %v", score)
	}
}

func TestMatchmaker_GetLatencyTracker(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	tracker := mm.GetLatencyTracker()
	if tracker != nil {
		t.Errorf("expected nil latency tracker, got %v", tracker)
	}
}

func TestMatchmaker_RegisterNode_WithValidRequest(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	req := &models.NodeRegistrationRequest{
		NodeID:   "test-node-1",
		Country:  "US",
		City:     "NYC",
		IP:       "1.2.3.4",
		ISP:      "Test ISP",
		OS:       "Linux",
		NodeType: "peer",
	}

	err := mm.RegisterNode(req)
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_Heartbeat_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.Heartbeat(&models.NodeHeartbeatRequest{
		NodeID:    "node1",
		Battery:   80,
		CPUUsage:  30.0,
		IsCharging: true,
	})
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_GetCooldownEntries_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetCooldownEntries()
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_AddToCooldown_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.AddToCooldown("target1", "node1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_SelectNode_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.SelectNode("US", "", "example.com")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_FilterEligibleNodes(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	candidates := []string{"node1", "node2", "node3"}

	// With nil redis, filterEligibleNodes returns all candidates (no cooldown check possible)
	eligible := mm.filterEligibleNodes(candidates, "target1")

	// When redis is nil, all nodes are returned (no cooldown check)
	if len(eligible) != 3 {
		t.Errorf("expected 3 eligible nodes when redis is nil, got %d", len(eligible))
	}
}

func TestMatchmaker_FilterEligibleNodes_WithCooldown(t *testing.T) {
	// This test would require a mock redis to properly test cooldown filtering
	// Skipped for now as it requires integration test setup
}

func TestMatchmaker_SelectByLoad(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	nodes := []string{"node1", "node2"}

	selected, err := mm.selectByLoad(nodes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == "" {
		t.Error("expected a selected node")
	}
	if selected != "node1" && selected != "node2" {
		t.Errorf("selected node should be one of the candidates, got %s", selected)
	}
}

func TestMatchmaker_SelectByLoad_Empty(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.selectByLoad([]string{})
	if err == nil {
		t.Error("expected error for empty node list")
	}
}

func TestMatchmaker_SelectByLoad_Single(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	selected, err := mm.selectByLoad([]string{"node1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "node1" {
		t.Errorf("expected node1, got %s", selected)
	}
}

func TestMatchmaker_PingNode_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	healthy := mm.pingNode("node1")
	if healthy {
		t.Error("expected false when no Redis client")
	}
}

func TestMatchmaker_CheckNodeHealth_NoRedis(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	mm.checkNodeHealth()
}

func TestMatchmaker_RecordFailure_SetsLastFailure(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")

	breakers := mm.GetCircuitBreakers()
	if breakers["node1"].LastFailure.IsZero() {
		t.Error("expected LastFailure to be set")
	}
}

func TestMatchmaker_RecordFailure_MultipleNodes(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node2")
	mm.RecordFailure("node2")

	breakers := mm.GetCircuitBreakers()

	if len(breakers) != 2 {
		t.Errorf("expected 2 circuit breakers, got %d", len(breakers))
	}
	if breakers["node1"].FailureCount != 1 {
		t.Errorf("node1: expected 1 failure, got %d", breakers["node1"].FailureCount)
	}
	if breakers["node2"].FailureCount != 2 {
		t.Errorf("node2: expected 2 failures, got %d", breakers["node2"].FailureCount)
	}
}

func TestMatchmaker_ConcurrentRecordFailure(t *testing.T) {
	mm := newTestMatchmaker()

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			mm.RecordFailure("node1")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	breakers := mm.GetCircuitBreakers()
	if breakers["node1"].FailureCount != 100 {
		t.Errorf("expected 100 failures, got %d", breakers["node1"].FailureCount)
	}
	if breakers["node1"].State != "open" {
		t.Errorf("expected state open, got %s", breakers["node1"].State)
	}
}

func TestMatchmaker_ConcurrentRecordSuccess(t *testing.T) {
	mm := newTestMatchmaker()

	for i := 0; i < 5; i++ {
		mm.RecordFailure("node1")
	}

	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			mm.RecordSuccess("node1")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	if !mm.isNodeHealthy("node1") {
		t.Error("node should be healthy after many successes")
	}
}

func TestMatchmaker_CircuitBreakerThreshold(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       5,
	}

	for i := 0; i < 4; i++ {
		mm.RecordFailure("node1")
		if !mm.isNodeHealthy("node1") {
			t.Errorf("node should be healthy after %d failures (threshold=5)", i+1)
		}
	}

	mm.RecordFailure("node1")
	if mm.isNodeHealthy("node1") {
		t.Error("node should be unhealthy after 5 failures (threshold=5)")
	}
}

func TestMatchmaker_GetCircuitBreakers_Copy(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")

	breakers1 := mm.GetCircuitBreakers()
	breakers2 := mm.GetCircuitBreakers()

	if breakers1["node1"] == breakers2["node1"] {
		t.Error("GetCircuitBreakers should return copies, not same pointer")
	}
}

func TestMatchmaker_RegisterNode_ReferralCode(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	req := &models.NodeRegistrationRequest{
		NodeID:       "node1",
		Country:      "US",
		IP:           "1.2.3.4",
		ReferralCode: "ref123",
	}

	err := mm.RegisterNode(req)
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_Heartbeat_WithAllFields(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.Heartbeat(&models.NodeHeartbeatRequest{
		NodeID:     "node1",
		Battery:    90,
		CPUUsage:   45.5,
		IsCharging: false,
	})
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_SetSessionNode_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.SetSessionNode("sess1", "node1", 3600)
	if err == nil {
		t.Error("expected error when no Redis client")
	}

	err = mm.SetSessionNode("sess1", "node1", 0)
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_GetSessionNode_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	_, err := mm.GetSessionNode("sess1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_DecrementNodeLoad_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.DecrementNodeLoad("node1")
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_RecordBandwidth_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	err := mm.RecordBandwidth("node1", 10000, 5000, 100)
	if err == nil {
		t.Error("expected error when no Redis client")
	}

	err = mm.RecordBandwidth("node1", 0, 0, 0)
	if err == nil {
		t.Error("expected error when no Redis client")
	}
}

func TestMatchmaker_UpdateHealthScore_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	mm.UpdateHealthScore("node1", 50, 100.0, 0.99)
	mm.UpdateHealthScore("node1", 200, 10.0, 0.5)

	score := mm.GetHealthScore("node1")
	if score != nil {
		t.Errorf("expected nil without Redis, got %v", score)
	}
}

func TestMatchmaker_GetTopHealthPeers_WithCount(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	peers := mm.GetTopHealthPeers(5)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers without Redis, got %d", len(peers))
	}

	peers = mm.GetTopHealthPeers(0)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers for count 0, got %d", len(peers))
	}

	peers = mm.GetTopHealthPeers(-1)
	if len(peers) != 0 {
		t.Errorf("expected 0 peers for negative count, got %d", len(peers))
	}
}

func TestMatchmaker_GetHealthScoreHistory_Valid(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
	}

	history := mm.GetHealthScoreHistory("node1")
	if len(history) != 0 {
		t.Errorf("expected 0 history without Redis, got %d", len(history))
	}
}

func TestMatchmaker_StopHealthCheck_NoPanic(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
		stopHealthCheck: make(chan bool),
	}

	mm.StopHealthCheck()
}

func TestMatchmaker_StopCooldownCleanup_NoPanic(t *testing.T) {
	mm := &Matchmaker{
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       3,
		stopCooldown: make(chan bool),
	}

	mm.StopCooldownCleanup()
}

func TestMatchmaker_IsNodeHealthy_AfterReset(t *testing.T) {
	mm := newTestMatchmaker()

	mm.RecordFailure("node1")
	mm.RecordFailure("node1")
	mm.RecordFailure("node1")

	if mm.isNodeHealthy("node1") {
		t.Error("should be unhealthy")
	}

	mm.ResetCircuitBreaker("node1")

	if !mm.isNodeHealthy("node1") {
		t.Error("should be healthy after reset")
	}
}

func TestMatchmaker_CooldownTTL_GetSet(t *testing.T) {
	mm := newTestMatchmaker()

	original := mm.GetCooldownTTL()

	mm.SetCooldownTTL(20 * time.Minute)
	if mm.GetCooldownTTL() != 20*time.Minute {
		t.Errorf("expected 20m, got %v", mm.GetCooldownTTL())
	}

	mm.SetCooldownTTL(original)
	if mm.GetCooldownTTL() != original {
		t.Errorf("expected %v, got %v", original, mm.GetCooldownTTL())
	}
}