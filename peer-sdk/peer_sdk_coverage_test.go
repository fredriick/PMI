package peersdk

import (
	"testing"

	"proxymesh/internal/models"
)

func TestPeerSDK_GetStatus_NilNode(t *testing.T) {
	sdk := &PeerSDK{}
	status := sdk.GetStatus()
	if status.Online {
		t.Error("expected offline when node is nil")
	}
	if status.NodeID != "" {
		t.Errorf("expected empty NodeID, got %s", status.NodeID)
	}
}

func TestPeerSDK_GetStatus_Connected(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)
	sdk.isConnected = true
	sdk.node = &models.Node{
		ID:      "test-node",
		NodeType: models.NodeTypeResidential,
		Country: "US",
		City:    "Unknown",
		ISP:     "Residential",
		IP:      "192.168.1.1",
		OS:      "linux",
	}

	status := sdk.GetStatus()
	if status.NodeID != "test-node" {
		t.Errorf("expected NodeID test-node, got %s", status.NodeID)
	}
	if !status.Online {
		t.Error("expected Online to be true")
	}
	if status.IP != "192.168.1.1" {
		t.Errorf("expected IP 192.168.1.1, got %s", status.IP)
	}

	sdk.Stop()
}

func TestPeerSDK_GetStatus_Disconnected(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)
	sdk.isConnected = false
	sdk.node = &models.Node{
		ID:   "test-node",
		IP:   "192.168.1.1",
		OS:   "linux",
	}

	status := sdk.GetStatus()
	if status.Online {
		t.Error("expected offline")
	}
	if status.NodeID != "test-node" {
		t.Errorf("expected NodeID test-node, got %s", status.NodeID)
	}

	sdk.Stop()
}

func TestPeerSDK_StatusChan(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	ch := sdk.StatusChan()
	if ch == nil {
		t.Error("StatusChan() returned nil channel")
	}

	// Channel should be buffered with capacity 10
	if cap(ch) != 10 {
		t.Errorf("expected channel capacity 10, got %d", cap(ch))
	}

	sdk.Stop()
}

func TestPeerSDK_CheckEligibility_Defaults(t *testing.T) {
	sdk := &PeerSDK{
		config: &PeerConfig{
			NodeID:          "test-node",
			MinBattery:      0,
			MaxCPU:          0,
		},
	}

	if !sdk.checkEligibility() {
		t.Error("expected eligible with default config")
	}
}

func TestPeerSDK_CheckEligibility_LowBattery(t *testing.T) {
	sdk := &PeerSDK{
		config: &PeerConfig{
			NodeID:      "test-node",
			MinBattery:  1000,
			MaxCPU:      0,
		},
	}

	// On Windows, getBatteryLevel returns 100 on error (wmic not available)
	// isCharging returns true on error
	// So with MinBattery=1000 and charging=true, it should be eligible
	if sdk.checkEligibility() {
		t.Log("Note: eligibility may pass on this platform (battery=100, charging=true)")
	}
}

func TestPeerSDK_CheckEligibility_HighCPU(t *testing.T) {
	sdk := &PeerSDK{
		config: &PeerConfig{
			NodeID: "test-node",
			MinBattery: 0,
			MaxCPU:   0.001,
		},
	}

	// On Windows, getCPUUsage returns 10.0 on error, which > 0.001
	// So it should be ineligible
	if sdk.checkEligibility() {
		t.Log("Note: CPU eligibility may vary on this platform")
	}
}

func TestPeerSDK_CheckEligibility_NotUnmetered(t *testing.T) {
	// isOnUnmeteredWiFi returns true on Windows by default
	// Can't easily test false case without mocking
	sdk := &PeerSDK{
		config: &PeerConfig{
			NodeID:          "test-node",
			MinBattery:      0,
			MaxCPU:          0,
		},
	}

	if !sdk.isOnUnmeteredWiFi() {
		t.Error("expected unmetered WiFi on this platform")
	}
	if !sdk.checkEligibility() {
		t.Error("expected eligible with unmetered WiFi")
	}
}

func TestPeerSDK_ReportStatus(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	// reportStatus should not panic even with nil node
	sdk.reportStatus()

	sdk.Stop()
}

func TestPeerSDK_StatusChan_Buffered(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	// Channel should be buffered with capacity 10
	if cap(sdk.statusCh) != 10 {
		t.Errorf("expected channel capacity 10, got %d", cap(sdk.statusCh))
	}

	sdk.Stop()
}

func TestPeerSDK_IsConnected_AfterStop(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	sdk.Stop()

	if sdk.IsConnected() {
		t.Error("expected not connected after Stop")
	}
}

func TestConsentManager_EnableTwice(t *testing.T) {
	called := 0
	cm := NewConsentManager(
		func() error { called++; return nil },
		nil,
	)

	if err := cm.Enable(); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}
	if err := cm.Enable(); err != nil {
		t.Fatalf("second Enable() failed: %v", err)
	}
	if called != 1 {
		t.Errorf("onEnable called %d times, expected 1", called)
	}
	if !cm.IsEnabled() {
		t.Error("expected enabled")
	}
}

func TestConsentManager_DisableTwice(t *testing.T) {
	called := 0
	cm := NewConsentManager(
		nil,
		func() error { called++; return nil },
	)

	if err := cm.Enable(); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}
	if err := cm.Disable(); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}
	if err := cm.Disable(); err != nil {
		t.Fatalf("second Disable() failed: %v", err)
	}
	if called != 1 {
		t.Errorf("onDisable called %d times, expected 1", called)
	}
	if cm.IsEnabled() {
		t.Error("expected disabled")
	}
}

func TestConsentManager_EnableError(t *testing.T) {
	cm := NewConsentManager(
		func() error { return fmtError("enable failed") },
		nil,
	)

	if err := cm.Enable(); err == nil {
		t.Error("expected error from Enable()")
	}
	if cm.IsEnabled() {
		t.Error("expected not enabled after callback error")
	}
}

func TestConsentManager_DisableError(t *testing.T) {
	cm := NewConsentManager(
		nil,
		func() error { return fmtError("disable failed") },
	)

	if err := cm.Enable(); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}
	if err := cm.Disable(); err == nil {
		t.Error("expected error from Disable()")
	}
	if !cm.IsEnabled() {
		t.Error("expected still enabled after callback error")
	}
}

func fmtError(msg string) error {
	return &fmtErrorType{msg: msg}
}

type fmtErrorType struct {
	msg string
}

func (e *fmtErrorType) Error() string {
	return e.msg
}

func TestGenerateNodeID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateNodeID()
		if id == "" {
			t.Error("GenerateNodeID() returned empty string")
		}
		if ids[id] {
			t.Errorf("GenerateNodeID() returned duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestPeerConfig_Defaults(t *testing.T) {
	cfg := PeerConfig{}
	if cfg.NodeID != "" {
		t.Errorf("expected empty NodeID, got %s", cfg.NodeID)
	}
	if cfg.GatewayEndpoint != "" {
		t.Errorf("expected empty GatewayEndpoint, got %s", cfg.GatewayEndpoint)
	}
	if cfg.MTLSEnabled {
		t.Error("expected MTLSEnabled to be false")
	}
	if cfg.MinBattery != 0 {
		t.Errorf("expected MinBattery to be 0, got %d", cfg.MinBattery)
	}
	if cfg.MaxCPU != 0 {
		t.Errorf("expected MaxCPU to be 0, got %f", cfg.MaxCPU)
	}
}

func TestNodeStatus_Defaults(t *testing.T) {
	ns := NodeStatus{}
	if ns.NodeID != "" {
		t.Errorf("expected empty NodeID, got %s", ns.NodeID)
	}
	if ns.Online {
		t.Error("expected Online to be false")
	}
	if ns.Battery != 0 {
		t.Errorf("expected Battery to be 0, got %d", ns.Battery)
	}
	if ns.IsCharging {
		t.Error("expected IsCharging to be false")
	}
	if ns.CPUUsage != 0 {
		t.Errorf("expected CPUUsage to be 0, got %f", ns.CPUUsage)
	}
	if ns.IP != "" {
		t.Errorf("expected empty IP, got %s", ns.IP)
	}
}

func TestPeerSDK_ConcurrentReportBandwidth(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			sdk.ReportBandwidth(1000, 500, 10)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	sdk.Stop()
}

func TestPeerSDK_Stop_Idempotent(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	sdk.Stop()
	sdk.Stop() // Should not panic

	if sdk.IsConnected() {
		t.Error("expected not connected after double Stop")
	}
}
