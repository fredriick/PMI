package matchmaker

import (
	"testing"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
)

func TestRedisClient_GetCooldownEntries(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddToCooldown("target-1", "node-1")
	client.AddToCooldown("target-1", "node-2")
	client.AddToCooldown("target-2", "node-3")

	entries, err := client.GetCooldownEntries()
	if err != nil {
		t.Fatalf("GetCooldownEntries error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 targets, got %d", len(entries))
	}
	if len(entries["target-1"]) != 2 {
		t.Errorf("expected 2 nodes for target-1, got %d", len(entries["target-1"]))
	}
	if len(entries["target-2"]) != 1 {
		t.Errorf("expected 1 node for target-2, got %d", len(entries["target-2"]))
	}
}

func TestRedisClient_GetNode(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		ISP:      "Test ISP",
		Battery:  80,
		OS:       "Linux",
		LastSeen: time.Now(),
	})

	node, err := client.GetNode("node-1")
	if err != nil {
		t.Fatalf("GetNode error: %v", err)
	}
	if node.ID != "node-1" {
		t.Errorf("expected node-1, got %s", node.ID)
	}
	if node.ISP != "Test ISP" {
		t.Errorf("expected ISP 'Test ISP', got %s", node.ISP)
	}
}

func TestRedisClient_GetNode_NotFound(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	_, err := client.GetNode("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent node")
	}
}

func TestRedisClient_GetBandwidthHistory(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.RecordBandwidth("node-1", 1000, 500, 10)
	client.RecordBandwidth("node-1", 2000, 1000, 20)

	history, err := client.GetBandwidthHistory("node-1")
	if err != nil {
		t.Fatalf("GetBandwidthHistory error: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected at least one bandwidth history entry")
	}
}

func TestRedisClient_GetAllNodes(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		LastSeen: time.Now(),
	})
	client.AddNode(&models.Node{
		ID:       "node-2",
		Country:  "EU",
		LastSeen: time.Now(),
	})

	nodes, err := client.GetAllNodes()
	if err != nil {
		t.Fatalf("GetAllNodes error: %v", err)
	}
	if len(nodes) < 2 {
		t.Errorf("expected at least 2 nodes, got %d", len(nodes))
	}
}

func TestRedisClient_ListSessions(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.SetSessionNode("session-1", "node-1", 3600)
	client.SetSessionNode("session-2", "node-2", 3600)

	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s["session_id"] == "" || s["node_id"] == "" {
			t.Errorf("session missing fields: %v", s)
		}
	}
}

func TestRedisClient_DeleteSession(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.SetSessionNode("session-1", "node-1", 3600)

	err := client.DeleteSession("session-1")
	if err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	nodeID, err := client.GetSessionNode("session-1")
	if err != nil {
		t.Fatalf("GetSessionNode error: %v", err)
	}
	if nodeID != "" {
		t.Errorf("expected empty nodeID after delete, got %s", nodeID)
	}
}

func TestRedisClient_StoreAndGetBandwidthSnapshots(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	snapshot := BandwidthSnapshot{
		Timestamp:     time.Now().Unix(),
		BytesSent:       1000,
		BytesReceived:   500,
		TotalGB:         0.001,
	}

	err := client.StoreBandwidthSnapshot("node-1", snapshot)
	if err != nil {
		t.Fatalf("StoreBandwidthSnapshot error: %v", err)
	}

	snapshots, err := client.GetBandwidthSnapshots("node-1", 10)
	if err != nil {
		t.Fatalf("GetBandwidthSnapshots error: %v", err)
	}
	if len(snapshots) != 1 {
		t.Errorf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", snapshots[0].BytesSent)
	}
}

func TestRedisClient_RecordAndGetPayoutHistory(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	payout := map[string]interface{}{
		"amount":      100.50,
		"currency":    "USD",
		"period":      "2024-01",
		"node_id":     "node-1",
		"timestamp":   time.Now().Unix(),
	}

	err := client.RecordPayout("node-1", payout)
	if err != nil {
		t.Fatalf("RecordPayout error: %v", err)
	}

	history, err := client.GetPayoutHistory("node-1", 10)
	if err != nil {
		t.Fatalf("GetPayoutHistory error: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 payout record, got %d", len(history))
	}
	if history[0]["amount"] != 100.50 {
		t.Errorf("expected amount 100.50, got %v", history[0]["amount"])
	}
}

func TestRedisClient_RecordReferral(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.RecordReferral("ref-code-1", "new-node-1")
	if err != nil {
		t.Fatalf("RecordReferral error: %v", err)
	}

	err = client.RecordReferral("ref-code-1", "new-node-2")
	if err != nil {
		t.Fatalf("RecordReferral error: %v", err)
	}

	referrals, err := client.GetReferrals("ref-code-1")
	if err != nil {
		t.Fatalf("GetReferrals error: %v", err)
	}
	if len(referrals) != 2 {
		t.Errorf("expected 2 referrals, got %d", len(referrals))
	}

	count, err := client.GetReferralCount("ref-code-1")
	if err != nil {
		t.Fatalf("GetReferralCount error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestRedisClient_GetReferrals_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	referrals, err := client.GetReferrals("nonexistent-code")
	if err != nil {
		t.Fatalf("GetReferrals error: %v", err)
	}
	if len(referrals) != 0 {
		t.Errorf("expected 0 referrals, got %d", len(referrals))
	}
}

func TestRedisClient_GetReferralCount_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	count, err := client.GetReferralCount("nonexistent-code")
	if err != nil {
		t.Fatalf("GetReferralCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
}

func TestRedisClient_RemoveFromCooldown(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddToCooldown("target-1", "node-1")

	inCooldown, _ := client.IsInCooldown("target-1", "node-1")
	if !inCooldown {
		t.Error("expected node to be in cooldown before removal")
	}

	err := client.RemoveFromCooldown("target-1", "node-1")
	if err != nil {
		t.Fatalf("RemoveFromCooldown error: %v", err)
	}

	inCooldown, _ = client.IsInCooldown("target-1", "node-1")
	if inCooldown {
		t.Error("expected node to NOT be in cooldown after removal")
	}
}

func TestRedisClient_NewRedisClient_SentinelConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		SentinelEnabled: true,
		SentinelAddrs:   []string{"invalid-host:26379"},
		SentinelMaster:  "mymaster",
		Password:        "",
		DB:              0,
	}
	_, err := NewRedisClient(cfg)
	if err == nil {
		t.Error("expected error for invalid Sentinel config")
	}
}

func TestRedisClient_NewRedisClient_ClusterConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		ClusterEnabled: true,
		ClusterAddrs:   []string{"invalid-host:7000"},
		Password:       "",
	}
	_, err := NewRedisClient(cfg)
	if err == nil {
		t.Error("expected error for invalid Cluster config")
	}
}

func TestRedisClient_GetNodeLoad_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	load, err := client.GetNodeLoad("nonexistent")
	if err != nil {
		t.Fatalf("GetNodeLoad error: %v", err)
	}
	if load != 0 {
		t.Errorf("expected load 0 for nonexistent node, got %d", load)
	}
}

func TestRedisClient_UpdateNodeReputation_NewNode(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.UpdateNodeReputation("new-node", "US", 85)
	if err != nil {
		t.Fatalf("UpdateNodeReputation error: %v", err)
	}

	score, _ := client.GetNodeReputation("new-node", "US")
	if score != 85 {
		t.Errorf("expected reputation 85, got %f", score)
	}
}

func TestRedisClient_RecordBandwidth_WithPipeline(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.RecordBandwidth("node-1", 10000, 5000, 100)
	if err != nil {
		t.Fatalf("RecordBandwidth error: %v", err)
	}

	bw, err := client.GetBandwidth("node-1", time.Now())
	if err != nil {
		t.Fatalf("GetBandwidth error: %v", err)
	}
	if bw.BytesSent != 10000 {
		t.Errorf("expected 10000 bytes sent, got %d", bw.BytesSent)
	}
	if bw.BytesReceived != 5000 {
		t.Errorf("expected 5000 bytes received, got %d", bw.BytesReceived)
	}
	if bw.DurationSeconds != 100 {
		t.Errorf("expected duration 100, got %d", bw.DurationSeconds)
	}
}

func TestBandwidthSnapshot_Struct(t *testing.T) {
	ts := time.Now().Unix()
	bs := BandwidthSnapshot{
		Timestamp:     ts,
		BytesSent:       1000,
		BytesReceived:   500,
		TotalGB:         0.001,
	}
	if bs.Timestamp != ts {
		t.Error("unexpected Timestamp")
	}
	if bs.BytesSent != 1000 {
		t.Error("unexpected BytesSent")
	}
	if bs.BytesReceived != 500 {
		t.Error("unexpected BytesReceived")
	}
	if bs.TotalGB != 0.001 {
		t.Error("unexpected TotalGB")
	}
}

func TestRedisClient_GetNodeMeta_WithCity(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		City:     "NYC",
		ISP:      "Test ISP",
		Battery:  80,
		OS:       "Linux",
		LastSeen: time.Now(),
	})

	meta, err := client.GetNodeMeta("node-1")
	if err != nil {
		t.Fatalf("GetNodeMeta error: %v", err)
	}
	if meta.NodeID != "node-1" {
		t.Errorf("expected node-1, got %s", meta.NodeID)
	}
	if meta.ISP != "Test ISP" {
		t.Errorf("expected ISP 'Test ISP', got %s", meta.ISP)
	}
}

func TestRedisClient_AddNode_WithCity(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	node := &models.Node{
		ID:       "node-2",
		Country:  "US",
		City:     "LA",
		ISP:      "Test ISP 2",
		Battery:  90,
		OS:       "Windows",
		LastSeen: time.Now(),
		Reputation: 88,
	}

	err := client.AddNode(node)
	if err != nil {
		t.Fatalf("AddNode error: %v", err)
	}

	nodes, err := client.GetNodesByCity("US", "LA", 10)
	if err != nil {
		t.Fatalf("GetNodesByCity error: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node for LA, got %d", len(nodes))
	}
	if nodes[0] != "node-2" {
		t.Errorf("expected node-2, got %s", nodes[0])
	}
}

func TestRedisClient_GetTopNodes_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	nodes, err := client.GetTopNodes("NONEXISTENT", 10)
	if err != nil {
		t.Fatalf("GetTopNodes error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for nonexistent country, got %d", len(nodes))
	}
}

func TestRedisClient_GetNodesByCity_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	nodes, err := client.GetNodesByCity("US", "NONEXISTENT", 10)
	if err != nil {
		t.Fatalf("GetNodesByCity error: %v", err)
	}
	if nodes != nil && len(nodes) != 0 {
		t.Errorf("expected nil or empty nodes, got %v", nodes)
	}
}