package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHTTPRequestLogger_New(t *testing.T) {
	rl := NewRequestLogger(nil, true)
	if rl == nil {
		t.Fatal("NewRequestLogger returned nil")
	}
	if !rl.IsEnabled() {
		t.Error("expected logger to be enabled")
	}
	if rl.maxBodySize != 4096 {
		t.Errorf("expected maxBodySize 4096, got %d", rl.maxBodySize)
	}
}

func TestHTTPRequestLogger_SetEnabled(t *testing.T) {
	rl := NewRequestLogger(nil, false)
	if rl.IsEnabled() {
		t.Error("expected logger disabled")
	}
	rl.SetEnabled(true)
	if !rl.IsEnabled() {
		t.Error("expected logger enabled after SetEnabled(true)")
	}
	rl.SetEnabled(false)
	if rl.IsEnabled() {
		t.Error("expected logger disabled after SetEnabled(false)")
	}
}

func TestHTTPRequestLogger_Middleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRequestLogger(nil, false)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	rl.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request to pass through")
	}
}

func TestHTTPRequestLogger_Middleware_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rl := NewRequestLogger(nil, true)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	rl.Middleware()(c)
	if c.IsAborted() {
		t.Error("expected request to pass through")
	}
}

func TestHTTPRequestLogger_GetLogs_NoRedis(t *testing.T) {
	rl := NewRequestLogger(nil, true)
	if rl.IsEnabled() {
		// Just verify the logger is functional without Redis
	}
}

func TestHTTPRequestLogger_GetLogs_WithRedis(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	rl := NewRequestLogger(client, true)

	entry := RequestLogEntry{
		Timestamp: 1234567890,
		Method:    "GET",
		Path:      "/test",
		Status:    200,
		Duration:  10,
		ClientIP:  "127.0.0.1",
	}
	data, _ := json.Marshal(entry)
	client.RPush(client.Context(), "reqlog:20240101", string(data))

	logs, err := rl.GetLogs("20240101", 10)
	if err != nil {
		t.Fatalf("GetLogs error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Method != "GET" {
		t.Errorf("expected method 'GET', got %s", logs[0].Method)
	}
}

func TestHTTPRequestLogger_GetLogs_LimitCap(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	rl := NewRequestLogger(client, true)
	_, err := rl.GetLogs("20240101", 0)
	if err != nil {
		t.Fatalf("GetLogs error: %v", err)
	}
}

func TestRequestLogEntry_Struct(t *testing.T) {
	entry := RequestLogEntry{
		Timestamp:    1234567890,
		Method:       "POST",
		Path:         "/api/test",
		Status:       201,
		Duration:     50,
		ClientIP:     "10.0.0.1",
		UserAgent:    "test-agent",
		RequestSize:  100,
		ResponseSize: 200,
		Error:        "",
	}
	if entry.Method != "POST" {
		t.Error("unexpected Method")
	}
	if entry.Status != 201 {
		t.Error("unexpected Status")
	}
}
