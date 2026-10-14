package gateway

import (
	"net/http"
	"testing"
	"time"
)

func TestNewPeerBroadcaster_DefaultMaxHistory(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 0)
	if pb.maxHistory != 100 {
		t.Errorf("expected maxHistory 100, got %d", pb.maxHistory)
	}
	if pb.subs == nil {
		t.Error("expected subs to be initialized")
	}
	if pb.history == nil {
		t.Error("expected history to be initialized")
	}
}

func TestNewPeerBroadcaster_CustomMaxHistory(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 50)
	if pb.maxHistory != 50 {
		t.Errorf("expected maxHistory 50, got %d", pb.maxHistory)
	}
}

func TestPeerBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	ch := pb.Subscribe("node-1")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	if pb.subs["node-1"] == nil {
		t.Fatal("expected subs[node-1] to exist")
	}
	if _, exists := pb.subs["node-1"]["default"]; !exists {
		t.Error("expected default subscription to exist")
	}

	pb.Unsubscribe("node-1")
	if pb.subs["node-1"] != nil {
		if _, exists := pb.subs["node-1"]["default"]; exists {
			t.Error("expected default sub to be removed after unsubscribe")
		}
	}
}

func TestPeerBroadcaster_SubscribeMultiple(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	ch1 := pb.Subscribe("node-1")
	ch2 := pb.Subscribe("node-2")

	if ch1 == ch2 {
		t.Error("expected different channels for different nodes")
	}

	pb.Unsubscribe("node-1")
	pb.Unsubscribe("node-2")
}

func TestPeerBroadcaster_Broadcast(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 5)

	ch := pb.Subscribe("node-1")

	pb.Broadcast("test-type", "test-payload")

	select {
	case msg := <-ch:
		if string(msg) != "test-payload" {
			t.Errorf("expected payload 'test-payload', got '%s'", string(msg))
		}
	default:
		t.Error("expected message on channel")
	}

	history := pb.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("expected 1 history message, got %d", len(history))
	}
	if history[0].Type != "test-type" {
		t.Errorf("expected type 'test-type', got '%s'", history[0].Type)
	}
	if history[0].Payload != "test-payload" {
		t.Errorf("expected payload 'test-payload', got '%s'", history[0].Payload)
	}
}

func TestPeerBroadcaster_Broadcast_HistoryCap(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 3)

	pb.Broadcast("msg1", "p1")
	pb.Broadcast("msg2", "p2")
	pb.Broadcast("msg3", "p3")
	pb.Broadcast("msg4", "p4")

	history := pb.GetHistory(10)
	if len(history) != 3 {
		t.Errorf("expected 3 history messages (capped), got %d", len(history))
	}
	if history[0].Type != "msg2" {
		t.Errorf("expected oldest message to be evicted, got '%s'", history[0].Type)
	}
	if history[2].Type != "msg4" {
		t.Errorf("expected newest message to be 'msg4', got '%s'", history[2].Type)
	}
}

func TestPeerBroadcaster_Broadcast_NoSubscribers(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	pb.Broadcast("type", "payload")

	history := pb.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("expected 1 history message, got %d", len(history))
	}
}

func TestPeerBroadcaster_Broadcast_NilRedis(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	pb.Broadcast("type", "payload")

	history := pb.GetHistory(1)
	if len(history) != 1 || history[0].Payload != "payload" {
		t.Error("expected broadcast to work with nil redis")
	}
}

func TestPeerBroadcaster_SendToNode_Success(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)
	pb.Subscribe("node-1")

	err := pb.SendToNode("node-1", "type", "payload")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPeerBroadcaster_SendToNode_NotConnected(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	err := pb.SendToNode("node-99", "type", "payload")
	if err == nil {
		t.Error("expected error for disconnected node")
	}
	if err != nil && err.Error() != "node node-99 not connected" {
		t.Errorf("expected 'node node-99 not connected', got '%v'", err)
	}
}

func TestPeerBroadcaster_GetHistory_Limit(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)

	pb.Broadcast("m1", "p1")
	pb.Broadcast("m2", "p2")
	pb.Broadcast("m3", "p3")

	history := pb.GetHistory(2)
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}
	if history[0].Type != "m1" {
		t.Errorf("expected 'm1' as first, got '%s'", history[0].Type)
	}
	if history[1].Type != "m2" {
		t.Errorf("expected 'm2' as second, got '%s'", history[1].Type)
	}
}

func TestPeerBroadcaster_GetHistory_ZeroLimit(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)
	pb.Broadcast("m1", "p1")

	history := pb.GetHistory(0)
	if len(history) != 1 {
		t.Errorf("expected 1 message for limit 0 (returns all), got %d", len(history))
	}
}

func TestPeerBroadcaster_GetHistory_LimitExceeds(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)
	pb.Broadcast("m1", "p1")
	pb.Broadcast("m2", "p2")

	history := pb.GetHistory(10)
	if len(history) != 2 {
		t.Errorf("expected 2 messages (all), got %d", len(history))
	}
}

func TestPeerBroadcaster_Unsubscribe_NonExistent(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 100)
	pb.Unsubscribe("nonexistent")
}

func TestConfigNotifier_SubscribeUnsubscribe(t *testing.T) {
	cn := NewConfigNotifier()

	ch := cn.Subscribe("client-1")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	cn.Unsubscribe("client-1")
	if _, exists := cn.channels["client-1"]; exists {
		t.Error("expected client-1 to be removed")
	}
}

func TestConfigNotifier_Notify(t *testing.T) {
	cn := NewConfigNotifier()
	ch := cn.Subscribe("client-1")
	defer cn.Unsubscribe("client-1")

	cn.Notify("hello")

	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Errorf("expected 'hello', got '%s'", msg)
		}
	default:
		t.Error("expected message on channel")
	}
}

func TestConfigNotifier_NotifyConfigReload(t *testing.T) {
	cn := NewConfigNotifier()
	ch := cn.Subscribe("client-1")
	defer cn.Unsubscribe("client-1")

	cn.NotifyConfigReload()

	select {
	case msg := <-ch:
		if msg != "config_reloaded" {
			t.Errorf("expected 'config_reloaded', got '%s'", msg)
		}
	default:
		t.Error("expected message on channel")
	}
}

func TestConfigNotifier_MultipleSubscribers(t *testing.T) {
	cn := NewConfigNotifier()
	ch1 := cn.Subscribe("c1")
	ch2 := cn.Subscribe("c2")
	defer cn.Unsubscribe("c1")
	defer cn.Unsubscribe("c2")

	cn.Notify("msg")

	select {
	case msg := <-ch1:
		if msg != "msg" {
			t.Errorf("expected 'msg', got '%s'", msg)
		}
	default:
		t.Error("expected message on ch1")
	}

	select {
	case msg := <-ch2:
		if msg != "msg" {
			t.Errorf("expected 'msg', got '%s'", msg)
		}
	default:
		t.Error("expected message on ch2")
	}
}

func TestConfigNotifier_NonExistentUnsubscribe(t *testing.T) {
	cn := NewConfigNotifier()
	cn.Unsubscribe("nonexistent")
}

func TestResponseCompressor_New(t *testing.T) {
	rc := NewResponseCompressor(true, 5)
	if !rc.enabled {
		t.Error("expected enabled")
	}
	if rc.level != 5 {
		t.Errorf("expected level 5, got %d", rc.level)
	}
}

func TestResponseCompressor_New_DefaultLevel(t *testing.T) {
	rc := NewResponseCompressor(true, 0)
	if rc.level != 5 {
		t.Errorf("expected default level 5, got %d", rc.level)
	}
}

func TestResponseCompressor_New_Disabled(t *testing.T) {
	rc := NewResponseCompressor(false, 5)
	if rc.enabled {
		t.Error("expected disabled")
	}
}

func TestResponseCompressor_shouldSkip(t *testing.T) {
	rc := NewResponseCompressor(true, 5)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/ws/connect", true},
		{"/api/peer/stream", true},
		{"/metrics", true},
		{"/v1/metrics", true},
		{"/api/admin/config", false},
		{"/health", false},
	}

	for _, tt := range tests {
		result := rc.shouldSkip(tt.path)
		if result != tt.expected {
			t.Errorf("shouldSkip(%q) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestConnectionDrainer_New(t *testing.T) {
	cd := NewConnectionDrainer(30 * time.Second)
	if cd.drainTimeout != 30*time.Second {
		t.Errorf("expected drainTimeout 30s, got %v", cd.drainTimeout)
	}
	if cd.ActiveConnections() != 0 {
		t.Errorf("expected 0 active connections, got %d", cd.ActiveConnections())
	}
	if cd.IsDraining() {
		t.Error("expected not draining")
	}
}

func TestConnectionDrainer_StartDrain(t *testing.T) {
	cd := NewConnectionDrainer(time.Second)

	started := cd.StartDrain()
	if !started {
		t.Error("expected StartDrain to return true")
	}
	if !cd.IsDraining() {
		t.Error("expected IsDraining to return true")
	}

	second := cd.StartDrain()
	if second {
		t.Error("expected StartDrain to return false when already draining")
	}
}

func TestConnectionDrainer_Connections(t *testing.T) {
	cd := NewConnectionDrainer(time.Second)

	cd.IncrementConnections()
	cd.IncrementConnections()
	if cd.ActiveConnections() != 2 {
		t.Errorf("expected 2 connections, got %d", cd.ActiveConnections())
	}

	cd.DecrementConnections()
	if cd.ActiveConnections() != 1 {
		t.Errorf("expected 1 connection, got %d", cd.ActiveConnections())
	}

	cd.DecrementConnections()
	if cd.ActiveConnections() != 0 {
		t.Errorf("expected 0 connections, got %d", cd.ActiveConnections())
	}
}

func TestConnectionDrainer_WaitForDrain_Timeout(t *testing.T) {
	cd := NewConnectionDrainer(100 * time.Millisecond)
	cd.IncrementConnections()
	cd.StartDrain()

	start := time.Now()
	cd.WaitForDrain()
	elapsed := time.Since(start)

	if elapsed < 80*time.Millisecond {
		t.Errorf("expected WaitForDrain to block for ~100ms, took %v", elapsed)
	}
	if cd.ActiveConnections() != 1 {
		t.Errorf("expected connections still 1 (not decremented by WaitForDrain), got %d", cd.ActiveConnections())
	}
	if !cd.IsDraining() {
		t.Error("expected still draining")
	}
}

func TestConnectionDrainer_WrapServer(t *testing.T) {
	cd := NewConnectionDrainer(time.Second)
	server := &http.Server{}

	wrapped := cd.WrapServer(server)
	if wrapped == nil {
		t.Fatal("expected non-nil server")
	}
	if wrapped.Handler == nil {
		t.Error("expected handler to be set")
	}
	if cd.ActiveConnections() != 0 {
		t.Errorf("expected 0 connections (handler not invoked yet), got %d", cd.ActiveConnections())
	}
}

func TestGracefulShutdown_New(t *testing.T) {
	gs := NewGracefulShutdown(5 * time.Second)
	if gs.drainTimeout != 5*time.Second {
		t.Errorf("expected drainTimeout 5s, got %v", gs.drainTimeout)
	}
	if gs.shutdownCh == nil {
		t.Error("expected shutdownCh to be initialized")
	}
}

func TestGracefulShutdown_AddServer(t *testing.T) {
	gs := NewGracefulShutdown(time.Second)
	server := &http.Server{}

	gs.AddServer(server)
	gs.AddServer(server)

	gs.mu.Lock()
	count := len(gs.servers)
	gs.mu.Unlock()

	if count != 2 {
		t.Errorf("expected 2 servers, got %d", count)
	}
}

func TestGracefulShutdown_Trigger(t *testing.T) {
	gs := NewGracefulShutdown(time.Second)

	done := make(chan struct{})
	go func() {
		gs.Start()
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	gs.Trigger()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("expected Start to return after Trigger")
	}
}
