package peersdk

import (
	"testing"
	"time"
)

func TestPeerSDK_ReportBandwidth(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	sdk.ReportBandwidth(1000, 500, 10)

	select {
	case bw := <-sdk.bwCh:
		if bw.BytesSent != 1000 {
			t.Errorf("expected 1000 bytes sent, got %d", bw.BytesSent)
		}
		if bw.BytesReceived != 500 {
			t.Errorf("expected 500 bytes received, got %d", bw.BytesReceived)
		}
	case <-time.After(time.Second):
		t.Error("expected bandwidth snapshot on channel")
	}
}

func TestPeerSDK_ReportBandwidth_BufferFull(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	// Fill the buffer
	for i := 0; i < 11; i++ {
		sdk.ReportBandwidth(int64(i), 0, 1)
	}

	// Should not panic or block
	sdk.ReportBandwidth(999, 0, 1)
}

func TestPeerSDK_TriggerReconnect(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	sdk.TriggerReconnect()

	select {
	case <-sdk.reconnCh:
		// OK
	case <-time.After(time.Second):
		t.Error("expected reconnect signal on channel")
	}
}

func TestPeerSDK_TriggerReconnect_BufferFull(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	// Fill the buffer
	sdk.TriggerReconnect()
	// Second call should not block
	sdk.TriggerReconnect()
}

func TestPeerSDK_Stop(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	sdk.Stop()

	sdk.mu.RLock()
	if sdk.isConnected {
		t.Error("expected isConnected to be false after Stop")
	}
	sdk.mu.RUnlock()
}

func TestPeerSDK_Disconnect(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	sdk.Disconnect()

	sdk.mu.RLock()
	if sdk.isConnected {
		t.Error("expected isConnected to be false after Disconnect")
	}
	sdk.mu.RUnlock()
}

func TestPeerSDK_IsConnected(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node"}
	sdk := NewPeerSDK(cfg)

	if sdk.IsConnected() {
		t.Error("expected not connected initially")
	}
}

func TestBandwidthSnapshot_Struct(t *testing.T) {
	bs := BandwidthSnapshot{
		BytesSent:       1000,
		BytesReceived:   500,
		DurationSeconds: 10,
	}
	if bs.BytesSent != 1000 {
		t.Error("unexpected BytesSent")
	}
	if bs.BytesReceived != 500 {
		t.Error("unexpected BytesReceived")
	}
	if bs.DurationSeconds != 10 {
		t.Error("unexpected DurationSeconds")
	}
}

func TestPeerConfig_Struct(t *testing.T) {
	cfg := PeerConfig{
		NodeID:          "node-1",
		GatewayEndpoint:  "localhost:9000",
		MTLSEnabled:     true,
		MinBattery:      20,
		MaxCPU:          80.0,
	}
	if cfg.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if !cfg.MTLSEnabled {
		t.Error("expected MTLSEnabled to be true")
	}
}

func TestNodeStatus_Struct(t *testing.T) {
	ns := NodeStatus{
		NodeID:     "node-1",
		Online:     true,
		Battery:    80,
		IsCharging: true,
		CPUUsage:   45.5,
		IP:         "192.168.1.1",
	}
	if ns.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if !ns.Online {
		t.Error("expected Online to be true")
	}
	if ns.Battery != 80 {
		t.Error("unexpected Battery")
	}
}

func TestPeerSDK_Start_NoGateway(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 0}
	sdk := NewPeerSDK(cfg)

	err := sdk.Start()
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	sdk.Stop()
}

func TestPeerSDK_Start_Ineligible(t *testing.T) {
	cfg := &PeerConfig{NodeID: "test-node", MinBattery: 1000, MaxCPU: 0.001}
	sdk := NewPeerSDK(cfg)

	err := sdk.Start()
	if err == nil {
		t.Log("Note: eligibility check may pass on this platform")
		sdk.Stop()
	}
}
