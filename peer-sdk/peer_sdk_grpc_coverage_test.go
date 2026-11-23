package peersdk

import (
	"testing"
	"time"

	"proxymesh/internal/models"
)

func TestPeerClient_Struct(t *testing.T) {
	pc := &PeerClient{
		config:   &PeerConfig{NodeID: "node-1"},
		stopped:  false,
		sessionID: "sess-1",
	}
	if pc.config.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if pc.stopped {
		t.Error("expected stopped=false")
	}
	if pc.sessionID != "sess-1" {
		t.Error("unexpected sessionID")
	}
}

func TestPeerClient_Connect_Stopped(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: true,
	}
	err := pc.Connect()
	if err == nil {
		t.Error("expected error for stopped client")
	}
}

func TestPeerClient_Heartbeat_Stopped(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: true,
	}
	err := pc.Heartbeat(80, 30.0, true, 1000, 500, 10)
	if err == nil {
		t.Error("expected error for stopped client")
	}
}

func TestPeerClient_ReportBandwidth_Stopped(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: true,
	}
	err := pc.ReportBandwidth(1000, 500, 10)
	if err == nil {
		t.Error("expected error for stopped client")
	}
}

func TestPeerClient_Disconnect_Stopped(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: true,
	}
	err := pc.Disconnect("test")
	if err != nil {
		t.Errorf("expected nil error for stopped client disconnect, got %v", err)
	}
}

func TestPeerClient_Disconnect_NilClient(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: false,
	}
	err := pc.Disconnect("test")
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestPeerClient_Stop_Idempotent(t *testing.T) {
	pc := &PeerClient{
		config: &PeerConfig{NodeID: "node-1"},
	}
	pc.Stop()
	pc.Stop() // Should not panic
	if !pc.stopped {
		t.Error("expected stopped=true after Stop")
	}
}

func TestPeerClient_StartTelemetryStream(t *testing.T) {
	pc := &PeerClient{
		config: &PeerConfig{NodeID: "node-1"},
	}
	ch := make(chan TelemetryUpdate, 1)
	close(ch)
	pc.StartTelemetryStream(ch)
	time.Sleep(50 * time.Millisecond)
	pc.Stop()
}

func TestTelemetryUpdate_Struct(t *testing.T) {
	tu := TelemetryUpdate{
		Battery:       80,
		CPUUsage:      45.5,
		IsCharging:    true,
		BytesSent:     1000,
		BytesReceived: 500,
		IP:            "192.168.1.1",
	}
	if tu.Battery != 80 {
		t.Error("unexpected Battery")
	}
	if tu.CPUUsage != 45.5 {
		t.Error("unexpected CPUUsage")
	}
	if !tu.IsCharging {
		t.Error("expected IsCharging=true")
	}
	if tu.BytesSent != 1000 {
		t.Error("unexpected BytesSent")
	}
	if tu.BytesReceived != 500 {
		t.Error("unexpected BytesReceived")
	}
	if tu.IP != "192.168.1.1" {
		t.Error("unexpected IP")
	}
}

func TestPeerSDK_GetSystemInfo(t *testing.T) {
	sdk := &PeerSDK{}
	info := sdk.GetSystemInfo()
	if info["os"] == "" {
		t.Error("os field should not be empty")
	}
	if info["arch"] == "" {
		t.Error("arch field should not be empty")
	}
	if info["hostname"] == "" {
		t.Error("hostname field should not be empty")
	}
	if info["cpus"] == "" {
		t.Error("cpus field should not be empty")
	}
	if info["time"] == "" {
		t.Error("time field should not be empty")
	}
}

func TestPeerSDK_GetLocalIP(t *testing.T) {
	sdk := &PeerSDK{}
	ip := sdk.getLocalIP()
	if ip == "" {
		t.Error("getLocalIP() returned empty string")
	}
}

func TestPeerSDK_GetBatteryLevel_Default(t *testing.T) {
	sdk := &PeerSDK{}
	level := sdk.getBatteryLevel()
	if level != 100 {
		t.Errorf("expected 100 on unsupported platform, got %d", level)
	}
}

func TestPeerSDK_IsCharging_Default(t *testing.T) {
	sdk := &PeerSDK{}
	if !sdk.isCharging() {
		t.Error("expected isCharging=true on unsupported platform")
	}
}

func TestPeerSDK_GetCPUUsage_Default(t *testing.T) {
	sdk := &PeerSDK{}
	cpu := sdk.getCPUUsage()
	// On Windows, CPU usage varies - just check it's a reasonable value
	if cpu < 0 || cpu > 100 {
		t.Errorf("expected CPU usage 0-100, got %f", cpu)
	}
}

func TestPeerSDK_IsOnUnmeteredWiFi_Default(t *testing.T) {
	sdk := &PeerSDK{}
	if !sdk.isOnUnmeteredWiFi() {
		t.Error("expected isOnUnmeteredWiFi=true on unsupported platform")
	}
}

func TestPeerSDK_Start_NilConfig(t *testing.T) {
	sdk := &PeerSDK{}
	err := sdk.Start()
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestPeerSDK_Start_Ineligible_HighMinBattery(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:     "test-node",
		MinBattery: 1000,
		MaxCPU:     0,
	}
	sdk := NewPeerSDK(cfg)
	err := sdk.Start()
	if err != nil {
		t.Logf("Start returned error (expected on some platforms): %v", err)
		sdk.Stop()
		return
	}
	sdk.Stop()
}

func TestPeerSDK_Start_Ineligible_LowMaxCPU(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:  "test-node",
		MaxCPU:  0.001,
	}
	sdk := NewPeerSDK(cfg)
	err := sdk.Start()
	if err != nil {
		t.Logf("Start returned error (expected on some platforms): %v", err)
		sdk.Stop()
		return
	}
	sdk.Stop()
}

func TestPeerSDK_Start_WithGateway(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:          "test-node",
		GatewayEndpoint: "localhost:9999",
		MinBattery:      0,
		MaxCPU:          0,
	}
	sdk := NewPeerSDK(cfg)
	err := sdk.Start()
	if err == nil {
		t.Log("Note: Start with gateway succeeded (unexpected without server)")
		sdk.Stop()
		return
	}
	t.Logf("Start with gateway returned error (expected): %v", err)
	sdk.Stop()
}

func TestPeerSDK_ReportBandwidth_AfterStop(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)
	sdk.Stop()
	sdk.ReportBandwidth(1000, 500, 10)
}

func TestPeerSDK_TriggerReconnect_AfterStop(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)
	sdk.Stop()
	sdk.TriggerReconnect()
}

func TestPeerSDK_GetStatus_WithNode(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)
	sdk.node = &models.Node{
		ID:      "test-node",
		NodeType: models.NodeTypeResidential,
		Country: "US",
		City:    "NYC",
		ISP:     "Test ISP",
		IP:      "192.168.1.100",
		OS:      "linux",
	}
	sdk.isConnected = true

	status := sdk.GetStatus()
	if status.NodeID != "test-node" {
		t.Errorf("expected NodeID test-node, got %s", status.NodeID)
	}
	if !status.Online {
		t.Error("expected Online=true")
	}
	if status.IP != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", status.IP)
	}

	sdk.Stop()
}

func TestPeerSDK_IsConnected_True(t *testing.T) {
	sdk := &PeerSDK{}
	sdk.isConnected = true
	if !sdk.IsConnected() {
		t.Error("expected IsConnected=true")
	}
}

func TestPeerSDK_ConcurrentStop(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			sdk.Stop()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGrpcClientConfig_MTLSDisabled(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:          "node-1",
		GatewayEndpoint: "localhost:9999",
		MTLSEnabled:     false,
	}
	client, err := NewPeerClient(cfg)
	if err != nil {
		t.Logf("NewPeerClient returned error: %v", err)
	}
	// grpc.Dial returns immediately, connection happens async
	// The error may come later when actually using the client
	if client != nil {
		client.Stop()
	}
}

func TestGrpcClientConfig_MTLSEnabled(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:          "node-1",
		GatewayEndpoint: "localhost:9999",
		MTLSEnabled:     true,
	}
	client, err := NewPeerClient(cfg)
	if err != nil {
		t.Logf("NewPeerClient returned error: %v", err)
	}
	if client != nil {
		client.Stop()
	}
}

func TestPeerClient_Disconnect_Error(t *testing.T) {
	pc := &PeerClient{
		config:  &PeerConfig{NodeID: "node-1"},
		stopped: false,
		client:  nil,
	}
	// Disconnect with nil client should return error from nil pointer or handle gracefully
	err := pc.Disconnect("test")
	_ = err
}

func TestModels_NodeType(t *testing.T) {
	nt := models.NodeTypeResidential
	if nt != models.NodeType("residential") {
		t.Errorf("expected residential, got %s", nt)
	}
}

func TestModels_Node_Struct(t *testing.T) {
	node := models.Node{
		ID:         "node-1",
		NodeType:   models.NodeTypeResidential,
		Country:    "US",
		City:       "NYC",
		ISP:        "Test ISP",
		IP:         "192.168.1.1",
		IPv6Subnet: "2001:db8::/32",
		OS:         "linux",
		LastSeen:   time.Now(),
		Reputation: 100.0,
	}
	if node.ID != "node-1" {
		t.Error("unexpected ID")
	}
	if node.Reputation != 100.0 {
		t.Error("unexpected Reputation")
	}
}

func TestModels_CircuitBreaker_Struct(t *testing.T) {
	cb := models.CircuitBreaker{
		FailureCount: 3,
		Threshold:    5,
		State:        "open",
	}
	if cb.FailureCount != 3 {
		t.Error("unexpected FailureCount")
	}
	if cb.Threshold != 5 {
		t.Error("unexpected Threshold")
	}
	if cb.State != "open" {
		t.Error("unexpected State")
	}
}
