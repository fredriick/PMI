package peersdk

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"testing"
)

func TestGenerateNodeID(t *testing.T) {
	id1 := GenerateNodeID()
	id2 := GenerateNodeID()

	if id1 == "" {
		t.Error("GenerateNodeID() returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateNodeID() returned duplicate IDs")
	}
}

func TestGetSystemInfo(t *testing.T) {
	sdk := &PeerSDK{}
	info := sdk.GetSystemInfo()

	if info["os"] != runtime.GOOS {
		t.Errorf("expected os=%q, got %q", runtime.GOOS, info["os"])
	}
	if info["arch"] != runtime.GOARCH {
		t.Errorf("expected arch=%q, got %q", runtime.GOARCH, info["arch"])
	}
	if info["cpus"] == "" {
		t.Error("cpus field should not be empty")
	}
	cpus, err := strconv.Atoi(info["cpus"])
	if err != nil {
		t.Errorf("cpus should be an integer, got %q: %v", info["cpus"], err)
	}
	if cpus <= 0 {
		t.Errorf("cpus should be > 0, got %d", cpus)
	}
	if info["hostname"] == "" {
		hostname, _ := os.Hostname()
		if hostname != "" {
			t.Errorf("expected hostname %q, got empty", hostname)
		}
	}
	if info["time"] == "" {
		t.Error("time field should not be empty")
	}
}

func TestGetLocalIP(t *testing.T) {
	sdk := &PeerSDK{}
	ip := sdk.getLocalIP()

	if ip == "" {
		t.Error("getLocalIP() returned empty string")
	}
	if ip != "127.0.0.1" {
		if net.ParseIP(ip) == nil {
			t.Errorf("getLocalIP() returned invalid IP: %q", ip)
		}
	}
}

func TestConsentManager(t *testing.T) {
	var enabledCalled, disabledCalled bool

	cm := NewConsentManager(
		func() error { enabledCalled = true; return nil },
		func() error { disabledCalled = true; return nil },
	)

	if cm.IsEnabled() {
		t.Error("ConsentManager should start disabled")
	}

	if err := cm.Enable(); err != nil {
		t.Fatalf("Enable() failed: %v", err)
	}
	if !cm.IsEnabled() {
		t.Error("ConsentManager should be enabled after Enable()")
	}
	if !enabledCalled {
		t.Error("onEnable callback was not called")
	}

	if err := cm.Enable(); err != nil {
		t.Fatalf("double Enable() should not error, got: %v", err)
	}

	if err := cm.Disable(); err != nil {
		t.Fatalf("Disable() failed: %v", err)
	}
	if cm.IsEnabled() {
		t.Error("ConsentManager should be disabled after Disable()")
	}
	if !disabledCalled {
		t.Error("onDisable callback was not called")
	}
}

func TestConsentManager_NilCallbacks(t *testing.T) {
	cm := NewConsentManager(nil, nil)

	if err := cm.Enable(); err != nil {
		t.Fatalf("Enable() with nil callback should not error: %v", err)
	}
	if err := cm.Disable(); err != nil {
		t.Fatalf("Disable() with nil callback should not error: %v", err)
	}
}

func TestNewPeerSDK(t *testing.T) {
	cfg := &PeerConfig{
		NodeID:          "test-node-1",
		GatewayEndpoint: "localhost:8443",
		MTLSEnabled:     true,
		MinBattery:      20,
		MaxCPU:          80.0,
	}
	sdk := NewPeerSDK(cfg)

	if sdk == nil {
		t.Fatal("NewPeerSDK() returned nil")
	}
	if sdk.config != cfg {
		t.Error("config not set correctly")
	}
	if sdk.ctx == nil {
		t.Error("context should not be nil")
	}
	if sdk.cancel == nil {
		t.Error("cancel func should not be nil")
	}
	if sdk.statusCh == nil {
		t.Error("statusCh should not be nil")
	}
	if sdk.reconnCh == nil {
		t.Error("reconnCh should not be nil")
	}

	sdk.Stop()
}
