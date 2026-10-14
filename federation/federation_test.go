package federation

import (
	"context"
	"testing"
	"time"

	"proxymesh/internal/config"
)

func TestFederationService_New(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:            false,
			Regions:            []string{"us-east", "eu-west"},
			HeartbeatInterval:  30,
			SyncInterval:       60,
		},
	}
	fs := NewFederationService(cfg, nil)
	if fs == nil {
		t.Fatal("NewFederationService returned nil")
	}
	if len(fs.peers) != 0 {
		t.Error("expected empty peers map")
	}
}

func TestFederationService_Start_Disabled(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)
	err := fs.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
}

func TestFederationService_RegisterPeer(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	err := fs.RegisterPeer("us-east", "10.0.0.1:8000")
	if err != nil {
		t.Fatalf("RegisterPeer error: %v", err)
	}

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Region != "us-east" {
		t.Errorf("expected region us-east, got %s", stats[0].Region)
	}
	if stats[0].Status != "healthy" {
		t.Errorf("expected healthy status, got %s", stats[0].Status)
	}
}

func TestFederationService_GetNodesForRegion_NotFound(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	_, err := fs.GetNodesForRegion("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent region")
	}
}

func TestFederationService_GetNodesForRegion_Found(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1", "node-2"},
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	nodes, err := fs.GetNodesForRegion("us-east")
	if err != nil {
		t.Fatalf("GetNodesForRegion error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestFederationService_HandleMessage_Heartbeat(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	msg := &FederationMessage{
		Type:   "heartbeat",
		Region: "eu-west",
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat after heartbeat, got %d", len(stats))
	}
}

func TestFederationService_HandleMessage_UnknownType(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	msg := &FederationMessage{
		Type:   "unknown",
		Region: "us-east",
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err == nil {
		t.Error("expected error for unknown message type")
	}
}

func TestFederationService_GetStats_Empty(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	stats := fs.GetStats()
	if len(stats) != 0 {
		t.Errorf("expected 0 stats, got %d", len(stats))
	}
}

func TestFederationService_Stop(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)
	fs.Stop()
}

func TestFederationStats_Struct(t *testing.T) {
	stats := FederationStats{
		Region:    "us-east",
		PeerCount: 1,
		NodeCount: 5,
		LastSync:  time.Now(),
		Status:    "healthy",
	}
	if stats.Region != "us-east" {
		t.Error("unexpected Region")
	}
	if stats.PeerCount != 1 {
		t.Error("unexpected PeerCount")
	}
	if stats.NodeCount != 5 {
		t.Error("unexpected NodeCount")
	}
}

func TestPeerGateway_Struct(t *testing.T) {
	pg := PeerGateway{
		Region:   "us-east",
		Address:  "10.0.0.1:8000",
		LastSeen: time.Now(),
		Healthy:  true,
	}
	if pg.Region != "us-east" {
		t.Error("unexpected Region")
	}
	if !pg.Healthy {
		t.Error("expected Healthy to be true")
	}
}

func TestFederationMessage_Struct(t *testing.T) {
	msg := FederationMessage{
		Type:      "heartbeat",
		Region:    "us-east",
		Nodes:     []string{"node-1", "node-2"},
		Timestamp: time.Now(),
	}
	if msg.Type != "heartbeat" {
		t.Error("unexpected Type")
	}
	if len(msg.Nodes) != 2 {
		t.Error("unexpected Nodes count")
	}
}

func TestFederationService_RegisterPeer_Duplicate(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	fs.RegisterPeer("us-east", "10.0.0.1:8000")
	fs.RegisterPeer("us-east", "10.0.0.2:8000")

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat (overwritten), got %d", len(stats))
	}
	if stats[0].Region != "us-east" {
		t.Errorf("expected region us-east, got %s", stats[0].Region)
	}
}

func TestFederationService_GetStats_WithNodeCache(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	fs.RegisterPeer("us-east", "10.0.0.1:8000")
	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1", "node-2", "node-3"},
	}
	fs.HandleMessage(context.Background(), msg)

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].NodeCount != 3 {
		t.Errorf("expected NodeCount 3, got %d", stats[0].NodeCount)
	}
}
