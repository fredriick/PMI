package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/gateway"
)

func TestNodeWebhook_RegisterGetClear(t *testing.T) {
	nw := gateway.NewNodeWebhook(3)
	nw.RegisterWebhook("node-1", gateway.WebhookConfig{URL: "http://example.com/hook", Events: []string{"node.registered"}})

	if len(nw.GetWebhooks("node-1")) != 1 {
		t.Fatal("expected 1 webhook for node-1")
	}
	if len(nw.GetAllWebhooks()["node-1"]) != 1 {
		t.Fatal("expected 1 webhook in GetAllWebhooks")
	}

	nw.ClearWebhooks("node-1")
	if len(nw.GetWebhooks("node-1")) != 0 {
		t.Fatal("expected 0 webhooks after clear")
	}
}

func TestNodeWebhook_TriggerEvent(t *testing.T) {
	done := make(chan gateway.WebhookPayload, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p gateway.WebhookPayload
		_ = json.NewDecoder(r.Body).Decode(&p)
		done <- p
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	nw := gateway.NewNodeWebhook(3)
	nw.RegisterWebhook("node-2", gateway.WebhookConfig{URL: srv.URL, Events: []string{"node.registered"}})
	nw.TriggerEvent("node-2", "node.registered", "registered")

	select {
	case got := <-done:
		if got.NodeID != "node-2" || got.Event != "node.registered" {
			t.Fatalf("unexpected payload: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook was not delivered")
	}

	// Unsubscribed event should not be delivered.
	nw.TriggerEvent("node-2", "node.deregistered", "deregistered")
	select {
	case <-done:
		t.Fatal("unsubscribed event should not be delivered")
	case <-time.After(500 * time.Millisecond):
	}
}

func TestWebhookHTTPRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	nw := gateway.NewNodeWebhook(3)
	r := gin.New()
	r.POST("/api/admin/webhooks", registerWebhookHandler(nw))
	r.GET("/api/admin/webhooks", listWebhooksHandler(nw))
	r.GET("/api/admin/webhooks/:nodeID", getWebhooksHandler(nw))
	r.DELETE("/api/admin/webhooks/:nodeID", deleteWebhooksHandler(nw))

	// Missing node_id -> 400
	body, _ := json.Marshal(gateway.WebhookConfig{URL: "http://x/h", Events: []string{"*"}})
	req := httptest.NewRequest("POST", "/api/admin/webhooks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("register without node_id status = %d", w.Code)
	}

	// Register
	req = httptest.NewRequest("POST", "/api/admin/webhooks?node_id=n1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status = %d", w.Code)
	}

	// List
	req = httptest.NewRequest("GET", "/api/admin/webhooks", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}

	// Get
	req = httptest.NewRequest("GET", "/api/admin/webhooks/n1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d", w.Code)
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/admin/webhooks/n1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete status = %d", w.Code)
	}
	if len(nw.GetWebhooks("n1")) != 0 {
		t.Fatal("expected webhooks cleared after delete")
	}
}
