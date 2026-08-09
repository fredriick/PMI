package gateway

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestWebSocketHub_RegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		conn:   nil,
		send:   make(chan []byte, 256),
		prefix: "admin",
	}

	hub.Register(client)
	if hub.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.Unregister(client)
	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var mu sync.Mutex
	received := make([]WSMessage, 0)

	client := &Client{
		hub:    hub,
		conn:   nil,
		send:   make(chan []byte, 256),
		prefix: "admin",
	}

	go func() {
		for msg := range client.send {
			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err == nil {
				mu.Lock()
				received = append(received, wsMsg)
				mu.Unlock()
			}
		}
	}()

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	update := NodeUpdate{
		NodeID:    "node-1",
		Status:    "healthy",
		Country:   "US",
		Load:      50,
		Timestamp: time.Now().Unix(),
	}
	hub.BroadcastNodeUpdate(update)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received))
	}
	if received[0].Type != "node_update" {
		t.Fatalf("expected type node_update, got %s", received[0].Type)
	}

	var payload NodeUpdate
	json.Unmarshal(received[0].Payload, &payload)
	if payload.NodeID != "node-1" {
		t.Fatalf("expected node_id node-1, got %s", payload.NodeID)
	}

	hub.Unregister(client)
}

func TestWebSocketHub_BroadcastMetrics(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var mu sync.Mutex
	received := make([]WSMessage, 0)

	client := &Client{
		hub:    hub,
		conn:   nil,
		send:   make(chan []byte, 256),
		prefix: "admin",
	}

	go func() {
		for msg := range client.send {
			var wsMsg WSMessage
			if err := json.Unmarshal(msg, &wsMsg); err == nil {
				mu.Lock()
				received = append(received, wsMsg)
				mu.Unlock()
			}
		}
	}()

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	metrics := MetricsUpdate{
		Requests:   1000,
		Success:    950,
		Failed:     50,
		LatencyAvg: 25.5,
		Timestamp:  time.Now().Unix(),
	}
	hub.BroadcastMetricsUpdate(metrics)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 message, got %d", len(received))
	}
	if received[0].Type != "metrics_update" {
		t.Fatalf("expected type metrics_update, got %s", received[0].Type)
	}

	hub.Unregister(client)
}

func TestWebSocketHub_MustMarshal(t *testing.T) {
	data := mustMarshal(struct {
		Hello string `json:"hello"`
	}{Hello: "world"})

	if string(data) != `{"hello":"world"}` {
		t.Fatalf("expected marshaled JSON, got %s", string(data))
	}
}

func TestWebSocketHub_MultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var mu sync.Mutex
	totalReceived := 0

	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = &Client{
			hub:    hub,
			conn:   nil,
			send:   make(chan []byte, 256),
			prefix: "admin",
		}

		go func(c *Client) {
			for range c.send {
				mu.Lock()
				totalReceived++
				mu.Unlock()
			}
		}(clients[i])

		hub.Register(clients[i])
	}

	time.Sleep(100 * time.Millisecond)
	if hub.ClientCount() != 5 {
		t.Fatalf("expected 5 clients, got %d", hub.ClientCount())
	}

	hub.Broadcast([]byte("test message"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if totalReceived != 5 {
		t.Fatalf("expected 5 clients to receive message, got %d", totalReceived)
	}

	for _, client := range clients {
		hub.Unregister(client)
	}

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after unregister, got %d", hub.ClientCount())
	}
}

func TestWebSocketHub_SlowClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		conn:   nil,
		send:   make(chan []byte, 2),
		prefix: "admin",
	}

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast([]byte("message 1"))
	hub.Broadcast([]byte("message 2"))
	hub.Broadcast([]byte("message 3"))

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected slow client to be removed, got %d", hub.ClientCount())
	}
}

func TestWSMessageJSON(t *testing.T) {
	msg := WSMessage{
		Type:    "test_type",
		Payload: json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded WSMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != "test_type" {
		t.Fatalf("expected type test_type, got %s", decoded.Type)
	}
	if string(decoded.Payload) != `{"key":"value"}` {
		t.Fatalf("expected payload {\"key\":\"value\"}, got %s", decoded.Payload)
	}
}

func TestNodeUpdateJSON(t *testing.T) {
	update := NodeUpdate{
		NodeID:    "node-123",
		Status:    "healthy",
		Country:   "US",
		Load:      75,
		Timestamp: 1234567890,
	}

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded NodeUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.NodeID != "node-123" {
		t.Fatalf("expected node_id node-123, got %s", decoded.NodeID)
	}
	if decoded.Status != "healthy" {
		t.Fatalf("expected status healthy, got %s", decoded.Status)
	}
	if decoded.Load != 75 {
		t.Fatalf("expected load 75, got %d", decoded.Load)
	}
}

func TestMetricsUpdateJSON(t *testing.T) {
	metrics := MetricsUpdate{
		Requests:   5000,
		Success:    4800,
		Failed:     200,
		LatencyAvg: 35.5,
		Timestamp:  1234567890,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MetricsUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Requests != 5000 {
		t.Fatalf("expected requests 5000, got %d", decoded.Requests)
	}
	if decoded.LatencyAvg != 35.5 {
		t.Fatalf("expected latency 35.5, got %f", decoded.LatencyAvg)
	}
}

func TestStartNodeBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	gw := &Gateway{
		wsHub: hub,
	}

	go gw.StartNodeBroadcast()

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestWebSocketHubConcurrentAccess(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var wg sync.WaitGroup
	numClients := 50

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &Client{
				hub:    hub,
				conn:   nil,
				send:   make(chan []byte, 256),
				prefix: "admin",
			}
			hub.Register(client)
			hub.Broadcast([]byte("concurrent test"))
			hub.Unregister(client)
		}(i)
	}

	wg.Wait()

	if hub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after concurrent test, got %d", hub.ClientCount())
	}
}
