package matchmaker

import (
	"testing"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
)

func TestRedisClient_NewRedisClient_InvalidConfig(t *testing.T) {
	cfg := &config.RedisConfig{
		Host: "invalid-host-that-does-not-exist",
		Port: 6379,
	}
	_, err := NewRedisClient(cfg)
	if err == nil {
		t.Error("expected error for invalid Redis config")
	}
}

func TestRedisClient_AddNode(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	node := &models.Node{
		ID:      "node-1",
		Country: "US",
		City:    "NYC",
		ISP:     "Test ISP",
		Battery: 80,
		OS:      "Linux",
		LastSeen: time.Now(),
		Reputation: 95,
	}

	err := client.AddNode(node)
	if err != nil {
		t.Fatalf("AddNode error: %v", err)
	}
}

func TestRedisClient_GetTopNodes(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		Reputation: 95,
	})
	client.AddNode(&models.Node{
		ID:       "node-2",
		Country:  "US",
		Reputation: 85,
	})

	nodes, err := client.GetTopNodes("US", 10)
	if err != nil {
		t.Fatalf("GetTopNodes error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestRedisClient_GetNodesByCity(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:     "node-1",
		Country: "US",
		City:   "NYC",
	})

	nodes, err := client.GetNodesByCity("US", "NYC", 10)
	if err != nil {
		t.Fatalf("GetNodesByCity error: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestRedisClient_GetNodeMeta(t *testing.T) {
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

func TestRedisClient_GetNodeMeta_NotFound(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	_, err := client.GetNodeMeta("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent node meta")
	}
}

func TestRedisClient_AddToCooldown(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.AddToCooldown("target-1", "node-1")
	if err != nil {
		t.Fatalf("AddToCooldown error: %v", err)
	}

	inCooldown, err := client.IsInCooldown("target-1", "node-1")
	if err != nil {
		t.Fatalf("IsInCooldown error: %v", err)
	}
	if !inCooldown {
		t.Error("expected node to be in cooldown")
	}
}

func TestRedisClient_AddToCooldownWithTTL(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.AddToCooldownWithTTL("target-2", "node-2", time.Second)
	if err != nil {
		t.Fatalf("AddToCooldownWithTTL error: %v", err)
	}

	inCooldown, _ := client.IsInCooldown("target-2", "node-2")
	if !inCooldown {
		t.Error("expected node to be in cooldown")
	}

	err = client.RemoveFromCooldown("target-2", "node-2")
	if err != nil {
		t.Fatalf("RemoveFromCooldown error: %v", err)
	}

	inCooldown, _ = client.IsInCooldown("target-2", "node-2")
	if inCooldown {
		t.Error("expected node to NOT be in cooldown after removal")
	}
}

func TestRedisClient_UpdateNodeStatus(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		LastSeen: time.Now(),
	})

	err := client.UpdateNodeStatus("node-1", 75, 45.5, true)
	if err != nil {
		t.Fatalf("UpdateNodeStatus error: %v", err)
	}
}

func TestRedisClient_UpdateNodeStatus_NotFound(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.UpdateNodeStatus("nonexistent", 50, 30.0, false)
	if err == nil {
		t.Error("expected error for nonexistent node")
	}
}

func TestRedisClient_RemoveNode(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:       "node-1",
		Country:  "US",
		LastSeen: time.Now(),
	})

	err := client.RemoveNode("node-1")
	if err != nil {
		t.Fatalf("RemoveNode error: %v", err)
	}
}

func TestRedisClient_GetSessionNode(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.SetSessionNode("session-1", "node-1", 3600)
	if err != nil {
		t.Fatalf("SetSessionNode error: %v", err)
	}

	nodeID, err := client.GetSessionNode("session-1")
	if err != nil {
		t.Fatalf("GetSessionNode error: %v", err)
	}
	if nodeID != "node-1" {
		t.Errorf("expected node-1, got %s", nodeID)
	}
}

func TestRedisClient_GetSessionNode_NotFound(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	nodeID, err := client.GetSessionNode("nonexistent")
	if err != nil {
		t.Fatalf("GetSessionNode error: %v", err)
	}
	if nodeID != "" {
		t.Errorf("expected empty string, got %s", nodeID)
	}
}

func TestRedisClient_IncrementDecrementLoad(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.IncrementNodeLoad("node-1")
	if err != nil {
		t.Fatalf("IncrementNodeLoad error: %v", err)
	}

	load, err := client.GetNodeLoad("node-1")
	if err != nil {
		t.Fatalf("GetNodeLoad error: %v", err)
	}
	if load != 1 {
		t.Errorf("expected load 1, got %d", load)
	}

	err = client.DecrementNodeLoad("node-1")
	if err != nil {
		t.Fatalf("DecrementNodeLoad error: %v", err)
	}

	load, _ = client.GetNodeLoad("node-1")
	if load != 0 {
		t.Errorf("expected load 0, got %d", load)
	}
}

func TestRedisClient_RecordBandwidth(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.RecordBandwidth("node-1", 1000, 500, 10)
	if err != nil {
		t.Fatalf("RecordBandwidth error: %v", err)
	}

	bw, err := client.GetBandwidth("node-1", time.Now())
	if err != nil {
		t.Fatalf("GetBandwidth error: %v", err)
	}
	if bw.BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", bw.BytesSent)
	}
	if bw.BytesReceived != 500 {
		t.Errorf("expected 500 bytes received, got %d", bw.BytesReceived)
	}
}

func TestRedisClient_GetBandwidth_Empty(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	bw, err := client.GetBandwidth("nonexistent", time.Now())
	if err != nil {
		t.Fatalf("GetBandwidth error: %v", err)
	}
	if bw.BytesSent != 0 {
		t.Errorf("expected 0 bytes sent, got %d", bw.BytesSent)
	}
}

func TestRedisClient_GetNodeReputation(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	client.AddNode(&models.Node{
		ID:        "node-1",
		Country:   "US",
		Reputation: 88,
	})

	score, err := client.GetNodeReputation("node-1", "US")
	if err != nil {
		t.Fatalf("GetNodeReputation error: %v", err)
	}
	if score != 88 {
		t.Errorf("expected reputation 88, got %f", score)
	}
}

func TestRedisClient_GetNodeReputation_Default(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	score, err := client.GetNodeReputation("nonexistent", "US")
	if err != nil {
		t.Fatalf("GetNodeReputation error: %v", err)
	}
	if score != 100 {
		t.Errorf("expected default reputation 100, got %f", score)
	}
}

func TestRedisClient_UpdateNodeReputation(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	err := client.UpdateNodeReputation("node-1", "US", 92)
	if err != nil {
		t.Fatalf("UpdateNodeReputation error: %v", err)
	}

	score, _ := client.GetNodeReputation("node-1", "US")
	if score != 92 {
		t.Errorf("expected reputation 92, got %f", score)
	}
}

func TestRedisClient_Close(t *testing.T) {
	client := redisClientMatchMaker(t)
	err := client.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}
}

func TestRedisClient_Client(t *testing.T) {
	client := redisClientMatchMaker(t)
	defer client.Close()

	rc := client.Client()
	if rc == nil {
		t.Error("expected non-nil Redis client")
	}
}

func TestBandwidthData_Struct(t *testing.T) {
	bd := models.BandwidthData{
		BytesSent:      1000,
		BytesReceived:  500,
		DurationSeconds: 10,
	}
	if bd.BytesSent != 1000 {
		t.Error("unexpected BytesSent")
	}
	if bd.BytesReceived != 500 {
		t.Error("unexpected BytesReceived")
	}
	if bd.DurationSeconds != 10 {
		t.Error("unexpected DurationSeconds")
	}
}

func TestNodeMeta_Struct(t *testing.T) {
	nm := models.NodeMeta{
		NodeID:   "node-1",
		ISP:      "Test ISP",
		Battery:  80,
		OS:       "Linux",
		LastSeen: time.Now(),
	}
	if nm.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if nm.ISP != "Test ISP" {
		t.Error("unexpected ISP")
	}
}

func redisClientMatchMaker(t *testing.T) *RedisClient {
	t.Helper()
	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		DB:       14,
		PoolSize: 10,
	}
	client, err := NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	client.client.FlushDB(client.ctx)
	return client
}
