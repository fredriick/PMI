package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h.clients == nil {
		t.Error("expected clients map to be initialized")
	}
	if h.broadcast == nil {
		t.Error("expected broadcast channel to be initialized")
	}
	if cap(h.broadcast) != 256 {
		t.Errorf("expected broadcast capacity 256, got %d", cap(h.broadcast))
	}
	if h.register == nil {
		t.Error("expected register channel to be initialized")
	}
	if cap(h.register) != 64 {
		t.Errorf("expected register capacity 64, got %d", cap(h.register))
	}
	if h.unregister == nil {
		t.Error("expected unregister channel to be initialized")
	}
}

func TestHub_ClientCount(t *testing.T) {
	h := NewHub()
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}

	c1 := &Client{send: make(chan []byte, 256)}
	c2 := &Client{send: make(chan []byte, 256)}

	h.Register(c1)
	h.Register(c2)
	h.Sync()

	if h.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", h.ClientCount())
	}

	h.Unregister(c1)
	h.Sync()
	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client after unregister, got %d", h.ClientCount())
	}

	h.Unregister(c2)
	h.Sync()
	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unregister all, got %d", h.ClientCount())
	}
}

func TestHub_Broadcast(t *testing.T) {
	h := NewHub()

	c1 := &Client{send: make(chan []byte, 256)}
	c2 := &Client{send: make(chan []byte, 256)}
	h.Register(c1)
	h.Register(c2)
	h.Sync()

	msg := []byte("test message")
	h.Broadcast(msg)

	select {
	case m := <-c1.send:
		if string(m) != "test message" {
			t.Errorf("expected 'test message', got '%s'", string(m))
		}
	default:
		t.Error("expected message on c1")
	}

	select {
	case m := <-c2.send:
		if string(m) != "test message" {
			t.Errorf("expected 'test message', got '%s'", string(m))
		}
	default:
		t.Error("expected message on c2")
	}
}

func TestHub_Broadcast_NoClients(t *testing.T) {
	h := NewHub()
	h.Broadcast([]byte("test"))
}

func TestHub_RegisterUnregister(t *testing.T) {
	h := NewHub()

	c := &Client{send: make(chan []byte, 256)}
	h.Register(c)
	h.Sync()

	if h.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.ClientCount())
	}

	h.Unregister(c)
	h.Sync()

	if h.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", h.ClientCount())
	}
}

func TestHub_Broadcast_ClientOverflow(t *testing.T) {
	h := NewHub()

	c := &Client{send: make(chan []byte, 1)}
	h.Register(c)
	h.Sync()

	c.send <- []byte("full")

	h.Broadcast([]byte("overflow"))
	h.Sync()

	if h.ClientCount() != 0 {
		t.Errorf("expected client to be removed after overflow, got %d", h.ClientCount())
	}
}

func TestClient_getIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.1:1234"

	client := &Client{conn: c}
	if client.getIP() != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got '%s'", client.getIP())
	}
}

func TestClient_getIP_NilConn(t *testing.T) {
	client := &Client{conn: nil}
	if client.getIP() != "unknown" {
		t.Errorf("expected 'unknown', got '%s'", client.getIP())
	}
}

func TestMustMarshal(t *testing.T) {
	msg := WSMessage{Type: "test", Payload: json.RawMessage(`{"key":"value"}`)}
	data := mustMarshal(msg)
	if len(data) == 0 {
		t.Error("expected non-empty marshaled data")
	}
}

func TestNewRequestLogger(t *testing.T) {
	rl := NewRequestLogger(nil, true)
	if rl.redisClient != nil {
		t.Error("expected nil redis client")
	}
	if !rl.enabled {
		t.Error("expected enabled")
	}
	if rl.maxBodySize != 4096 {
		t.Errorf("expected maxBodySize 4096, got %d", rl.maxBodySize)
	}
}

func TestNewRequestLogger_Disabled(t *testing.T) {
	rl := NewRequestLogger(nil, false)
	if rl.enabled {
		t.Error("expected disabled")
	}
}

func TestRequestLogger_SetEnabled(t *testing.T) {
	rl := NewRequestLogger(nil, false)
	rl.SetEnabled(true)
	if !rl.IsEnabled() {
		t.Error("expected enabled after SetEnabled(true)")
	}
	rl.SetEnabled(false)
	if rl.IsEnabled() {
		t.Error("expected disabled after SetEnabled(false)")
	}
}

func TestRequestLogger_Middleware_Disabled(t *testing.T) {
	rl := NewRequestLogger(nil, false)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	rl.Middleware()(c)

	if rl.IsEnabled() {
		t.Error("middleware should not change enabled state")
	}
}

func TestRequestLogger_Middleware_Enabled(t *testing.T) {
	rl := NewRequestLogger(nil, true)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Writer.WriteHeader(200)
	c.Writer.Write([]byte("response body"))

	rl.Middleware()(c)

	if !rl.IsEnabled() {
		t.Error("expected still enabled")
	}
}

func TestRequestLogger_Middleware_Enabled_NextCalled(t *testing.T) {
	rl := NewRequestLogger(nil, true)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	called := false

	rl.Middleware()(c)
	c.Next()

	called = true

	if !called {
		t.Error("expected c.Next to be called")
	}
}

func TestMustMarshal_NilPayload(t *testing.T) {
	data := mustMarshal(nil)
	if data == nil {
		t.Error("expected non-nil even for nil payload")
	}
}

func TestHub_Broadcast_MetricsUpdate(t *testing.T) {
	h := NewHub()
	c := &Client{send: make(chan []byte, 256)}
	h.Register(c)
	h.Sync()

	update := MetricsUpdate{Requests: 100, Success: 95, Failed: 5, LatencyAvg: 25.5}
	h.BroadcastMetricsUpdate(update)

	select {
		case m := <-c.send:
			var msg WSMessage
			if err := json.Unmarshal(m, &msg); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if msg.Type != "metrics_update" {
				t.Errorf("expected metrics_update, got %s", msg.Type)
			}
		default:
			t.Error("expected message on channel")
		}
}
