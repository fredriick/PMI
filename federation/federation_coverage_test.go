package federation

import (
	"context"
	"testing"

	"proxymesh/internal/config"
	"proxymesh/matchmaker"
)

func TestFederationService_Start_Enabled(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)
	err := fs.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	fs.Stop()
}

func TestFederationService_Stop_AfterStart(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)
	fs.Start()
	fs.Stop()
}

func TestFederationService_CheckPeerHealth(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           false,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)

	fs.RegisterPeer("us-east", "10.0.0.1:8000")

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Status != "healthy" {
		t.Errorf("expected healthy, got %s", stats[0].Status)
	}

	// Access private field to manipulate LastSeen - we can't directly test checkPeerHealth
	// but we can test the behavior through HandleMessage and time
}

func TestFederationService_SyncNodes(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           false,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	// Create a mock matchmaker
	mm := &matchmaker.Matchmaker{}
	fs := NewFederationService(cfg, mm)

	// syncNodes is private, but we can test the result through GetNodesForRegion
	// after HandleMessage (which simulates receiving a sync)
	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1", "node-2", "node-3"},
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	nodes, err := fs.GetNodesForRegion("us-east")
	if err != nil {
		t.Fatalf("GetNodesForRegion error: %v", err)
	}
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestFederationService_SelectNodeByRegion_WithMatchmaker(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           false,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	mm := &matchmaker.Matchmaker{}
	fs := NewFederationService(cfg, mm)

	// Pre-populate node cache
	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1"},
	}
	fs.HandleMessage(context.Background(), msg)

	// SelectNodeByRegion calls mm.GetNodeStatus which requires Redis
	_, err := fs.SelectNodeByRegion("us-east")
	if err == nil {
		t.Error("expected error when matchmaker has no Redis")
	}
}

func TestFederationService_SelectNodeByRegion_NoMatchmaker(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           false,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)

	_, err := fs.SelectNodeByRegion("us-east")
	if err == nil {
		t.Error("expected error when no matchmaker")
	}
}

func TestFederationService_SelectNodeByRegion_Fallback(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           false,
			Regions:           []string{"us-east", "eu-west"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	mm := &matchmaker.Matchmaker{}
	fs := NewFederationService(cfg, mm)

	// Pre-populate node cache for eu-west
	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "eu-west",
		Nodes:  []string{"eu-node-1"},
	}
	fs.HandleMessage(context.Background(), msg)

	// Should fall back to eu-west since us-east has no nodes
	_, err := fs.SelectNodeByRegion("us-east")
	if err == nil {
		t.Error("expected error when matchmaker has no Redis")
	}
}

func TestFederationService_HandleMessage_NodeSync_EmptyNodes(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{},
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	nodes, err := fs.GetNodesForRegion("us-east")
	if err != nil {
		t.Fatalf("GetNodesForRegion error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestFederationService_HandleMessage_Heartbeat_NewPeer(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	msg := &FederationMessage{
		Type:   "heartbeat",
		Region: "new-region",
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Region != "new-region" {
		t.Errorf("expected new-region, got %s", stats[0].Region)
	}
	if stats[0].Status != "healthy" {
		t.Errorf("expected healthy, got %s", stats[0].Status)
	}
}

func TestFederationService_HandleMessage_Heartbeat_UpdatesExisting(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	// Register peer first
	fs.RegisterPeer("us-east", "10.0.0.1:8000")

	// Send heartbeat - should update existing peer
	msg := &FederationMessage{
		Type:   "heartbeat",
		Region: "us-east",
	}
	err := fs.HandleMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleMessage error: %v", err)
	}

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].Status != "healthy" {
		t.Errorf("expected healthy, got %s", stats[0].Status)
	}
}

func TestFederationService_GetStats_MultiplePeers(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	fs.RegisterPeer("us-east", "10.0.0.1:8000")
	fs.RegisterPeer("eu-west", "10.0.0.2:8000")
	fs.RegisterPeer("ap-south", "10.0.0.3:8000")

	stats := fs.GetStats()
	if len(stats) != 3 {
		t.Errorf("expected 3 stats, got %d", len(stats))
	}

	regions := make(map[string]bool)
	for _, s := range stats {
		regions[s.Region] = true
		if s.Status != "healthy" {
			t.Errorf("peer %s expected healthy, got %s", s.Region, s.Status)
		}
	}
	if !regions["us-east"] || !regions["eu-west"] || !regions["ap-south"] {
		t.Errorf("missing expected regions: %v", regions)
	}
}

func TestFederationService_GetStats_WithNodeCounts(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	fs.RegisterPeer("us-east", "10.0.0.1:8000")

	// Add nodes via sync
	msg := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1", "node-2", "node-3", "node-4", "node-5"},
	}
	fs.HandleMessage(context.Background(), msg)

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].NodeCount != 5 {
		t.Errorf("expected NodeCount 5, got %d", stats[0].NodeCount)
	}
}

func TestFederationService_GetNodesForRegion_AfterHeartbeat(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	// Heartbeat creates peer entry
	msg := &FederationMessage{
		Type:   "heartbeat",
		Region: "us-east",
	}
	fs.HandleMessage(context.Background(), msg)

	// Node sync adds nodes
	msg2 := &FederationMessage{
		Type:   "node_sync",
		Region: "us-east",
		Nodes:  []string{"node-1", "node-2"},
	}
	fs.HandleMessage(context.Background(), msg2)

	nodes, err := fs.GetNodesForRegion("us-east")
	if err != nil {
		t.Fatalf("GetNodesForRegion error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestFederationService_ConcurrentRegisterPeer(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			fs.RegisterPeer("us-east", "10.0.0.1:8000")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := fs.GetStats()
	if len(stats) != 1 {
		t.Errorf("expected 1 stat, got %d", len(stats))
	}
}

func TestFederationService_ConcurrentHandleMessage(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{Enabled: false},
	}
	fs := NewFederationService(cfg, nil)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			msg := &FederationMessage{
				Type:   "node_sync",
				Region: "us-east",
				Nodes:  []string{"node-1"},
			}
			fs.HandleMessage(context.Background(), msg)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	nodes, err := fs.GetNodesForRegion("us-east")
	if err != nil {
		t.Fatalf("GetNodesForRegion error: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestFederationConfig_Defaults(t *testing.T) {
	fcfg := config.FederationConfig{}
	if fcfg.Enabled {
		t.Error("expected Enabled=false by default")
	}
	if fcfg.Regions != nil {
		t.Error("expected Regions=nil by default")
	}
	if fcfg.HeartbeatInterval != 0 {
		t.Errorf("expected HeartbeatInterval=0, got %d", fcfg.HeartbeatInterval)
	}
	if fcfg.SyncInterval != 0 {
		t.Errorf("expected SyncInterval=0, got %d", fcfg.SyncInterval)
	}
}

func TestFederationService_StartStop_Multiple(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)

	fs.Start()
	fs.Stop()
	fs.Stop() // Should not panic
}

func TestFederationService_HeartbeatInterval_Zero(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 0,
			SyncInterval:      1,
		},
	}
	fs := NewFederationService(cfg, nil)

	err := fs.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	fs.Stop()
}

func TestFederationService_SyncInterval_Zero(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"us-east"},
			HeartbeatInterval: 1,
			SyncInterval:      0,
		},
	}
	fs := NewFederationService(cfg, nil)

	err := fs.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	fs.Stop()
}