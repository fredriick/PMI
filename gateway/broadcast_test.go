package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestPeerBroadcaster_New(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 50)
	if pb == nil {
		t.Fatal("NewPeerBroadcaster returned nil")
	}
	if pb.maxHistory != 50 {
		t.Errorf("expected maxHistory 50, got %d", pb.maxHistory)
	}
}

func TestPeerBroadcaster_New_DefaultMaxHistory(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 0)
	if pb.maxHistory != 100 {
		t.Errorf("expected default maxHistory 100, got %d", pb.maxHistory)
	}
}

func TestPeerBroadcaster_SubscribeAndUnsubscribe(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 10)
	ch := pb.Subscribe("node-1")
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}

	pb.Unsubscribe("node-1")
}

func TestPeerBroadcaster_Broadcast_Basic(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 10)
	ch := pb.Subscribe("node-1")

	pb.Broadcast("test-type", "test-payload")

	select {
	case msg := <-ch:
		if string(msg) != "test-payload" {
			t.Errorf("expected 'test-payload', got %s", string(msg))
		}
	case <-time.After(time.Second):
		t.Error("expected message on channel")
	}

	history := pb.GetHistory(10)
	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}
	if history[0].Type != "test-type" {
		t.Errorf("expected type 'test-type', got %s", history[0].Type)
	}
}

func TestPeerBroadcaster_Broadcast_HistoryLimit(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 3)
	for i := 0; i < 5; i++ {
		pb.Broadcast("type", "payload")
	}
	history := pb.GetHistory(10)
	if len(history) != 3 {
		t.Errorf("expected 3 history entries (max), got %d", len(history))
	}
}

func TestPeerBroadcaster_SendToNode(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 10)
	pb.Subscribe("node-1")

	err := pb.SendToNode("node-1", "direct", "direct-payload")
	if err != nil {
		t.Fatalf("SendToNode error: %v", err)
	}

	err = pb.SendToNode("node-2", "direct", "payload")
	if err == nil {
		t.Error("expected error for disconnected node")
	}
}

func TestPeerBroadcaster_GetHistory_LimitBasic(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 10)
	pb.Broadcast("type", "payload")
	pb.Broadcast("type", "payload")

	history := pb.GetHistory(1)
	if len(history) != 1 {
		t.Errorf("expected 1 history entry with limit 1, got %d", len(history))
	}
}

func TestPeerBroadcaster_GetHistory_Empty(t *testing.T) {
	pb := NewPeerBroadcaster(nil, 10)
	history := pb.GetHistory(10)
	if len(history) != 0 {
		t.Errorf("expected 0 history entries, got %d", len(history))
	}
}

func TestPeerBroadcaster_BroadcastHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/broadcast", strings.NewReader(`{"type":"alert","payload":"hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	pb.broadcastHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPeerBroadcaster_BroadcastHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/broadcast", strings.NewReader(`{invalid}`))
	c.Request.Header.Set("Content-Type", "application/json")

	pb.broadcastHandler(c)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestPeerBroadcaster_SendToNodeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)
	pb.Subscribe("node-1")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/broadcast/node-1", strings.NewReader(`{"type":"alert","payload":"hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "nodeId", Value: "node-1"}}

	pb.sendToNodeHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPeerBroadcaster_SendToNodeHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/broadcast/node-2", strings.NewReader(`{"type":"alert","payload":"hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "nodeId", Value: "node-2"}}

	pb.sendToNodeHandler(c)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestPeerBroadcaster_HistoryHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)
	pb.Broadcast("type", "payload")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/broadcast/history?limit=5", nil)

	pb.historyHandler(c)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPeerBroadcaster_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pb := NewPeerBroadcaster(nil, 10)
	router := gin.New()
	pb.RegisterRoutes(router)
}

func TestBroadcastMessage_Struct(t *testing.T) {
	msg := BroadcastMessage{
		ID:      "msg-1",
		Type:    "alert",
		Payload: "hello",
		Target:  "node-1",
		SentAt:  time.Now(),
	}
	if msg.ID != "msg-1" {
		t.Error("unexpected ID")
	}
	if msg.Type != "alert" {
		t.Error("unexpected Type")
	}
}
