package federation

import (
	"encoding/json"
	"testing"
	"time"

	"proxymesh/internal/config"
)

func TestNewFederationService(t *testing.T) {
	cfg := &config.Config{}
	svc := NewFederationService(cfg, nil)
	if svc == nil {
		t.Fatal("NewFederationService() returned nil")
	}
	if svc.peers == nil {
		t.Error("peers map should be initialised")
	}
	if svc.nodeCache == nil {
		t.Error("nodeCache map should be initialised")
	}
	if svc.stop == nil {
		t.Error("stop channel should be initialised")
	}
}

func TestFederationService_RegisterPeer(t *testing.T) {
	cfg := &config.Config{
		Federation: config.FederationConfig{
			Enabled:           true,
			Regions:           []string{"eu-west-1"},
			HeartbeatInterval: 30,
			SyncInterval:      60,
		},
	}
	svc := NewFederationService(cfg, nil)

	err := svc.RegisterPeer("eu-west-1", "10.0.0.1:9000")
	if err != nil {
		t.Fatalf("RegisterPeer() unexpected error: %v", err)
	}
	err = svc.RegisterPeer("eu-west-1", "10.0.0.1:9001")
	if err != nil {
		t.Fatalf("RegisterPeer() duplicate should not error: %v", err)
	}

	stats := svc.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat entry, got %d", len(stats))
	}
	if stats[0].Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", stats[0].Region, "eu-west-1")
	}
	if stats[0].PeerCount != 1 {
		t.Errorf("PeerCount = %d, want 1", stats[0].PeerCount)
	}
	if stats[0].Status != "healthy" {
		t.Errorf("Status = %q, want %q", stats[0].Status, "healthy")
	}
}

func TestFederationService_GetStats_MultiplePeers(t *testing.T) {
	svc := &FederationService{
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
	}

	svc.RegisterPeer("us-east-1", "10.0.0.1:9000")
	svc.RegisterPeer("eu-west-1", "10.0.0.2:9000")

	stats := svc.GetStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stat entries, got %d", len(stats))
	}

	regions := make(map[string]bool)
	for _, s := range stats {
		regions[s.Region] = true
	}
	if !regions["us-east-1"] || !regions["eu-west-1"] {
		t.Errorf("missing expected regions: %v", regions)
	}
}

func TestFederationService_HandleMessage_Heartbeat(t *testing.T) {
	svc := &FederationService{
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
		stop:      make(chan bool),
	}

	msg := &FederationMessage{
		Type:      "heartbeat",
		Region:    "ap-southeast-1",
		Timestamp: time.Now(),
	}
	err := svc.HandleMessage(nil, msg)
	if err != nil {
		t.Fatalf("HandleMessage(heartbeat) unexpected error: %v", err)
	}

	stats := svc.GetStats()
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat after heartbeat, got %d", len(stats))
	}
	if stats[0].Region != "ap-southeast-1" {
		t.Errorf("Region = %q, want %q", stats[0].Region, "ap-southeast-1")
	}
}

func TestFederationService_HandleMessage_NodeSync(t *testing.T) {
	svc := &FederationService{
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
		stop:      make(chan bool),
	}

	nodeIDs := []string{"node-a", "node-b", "node-c"}
	msg := &FederationMessage{
		Type:      "node_sync",
		Region:    "eu-central-1",
		Nodes:     nodeIDs,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`["node-a","node-b","node-c"]`),
	}
	err := svc.HandleMessage(nil, msg)
	if err != nil {
		t.Fatalf("HandleMessage(node_sync) unexpected error: %v", err)
	}

	nodes, err := svc.GetNodesForRegion("eu-central-1")
	if err != nil {
		t.Fatalf("GetNodesForRegion() unexpected error: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
}

func TestFederationService_HandleMessage_Unknown(t *testing.T) {
	svc := &FederationService{
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
		stop:      make(chan bool),
	}

	msg := &FederationMessage{Type: "unknown_type"}
	err := svc.HandleMessage(nil, msg)
	if err == nil {
		t.Error("HandleMessage(unknown) should return error")
	}
}

func TestFederationService_GetNodesForRegion_Missing(t *testing.T) {
	svc := &FederationService{
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
		stop:      make(chan bool),
	}

	_, err := svc.GetNodesForRegion("nonexistent")
	if err == nil {
		t.Error("expected error for missing region")
	}
}

func TestFederationService_Disabled(t *testing.T) {
	svc := &FederationService{
		cfg: config.FederationConfig{Enabled: false},
	}
	err := svc.Start()
	if err != nil {
		t.Fatalf("Start() on disabled federation should not error: %v", err)
	}
}

func TestFederationService_Stop(t *testing.T) {
	svc := &FederationService{
		cfg:  config.FederationConfig{Enabled: true},
		stop: make(chan bool),
	}
	svc.Stop()
	select {
	case <-svc.stop:
		// OK
	default:
		t.Error("stop channel should be closed")
	}
}
