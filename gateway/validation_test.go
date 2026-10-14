package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestWebSocketHeartbeat_New(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)
	if wsh == nil {
		t.Fatal("NewWebSocketHeartbeat returned nil")
	}
	if wsh.connections == nil {
		t.Error("expected connections map to be initialized")
	}
	if wsh.pingInterval != 30*time.Second {
		t.Errorf("expected pingInterval 30s, got %v", wsh.pingInterval)
	}
	if wsh.timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", wsh.timeout)
	}
}

func TestWebSocketHeartbeat_RegisterConnection(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	wsh.RegisterConnection("conn-1", "node-1", "127.0.0.1")

	if !wsh.IsAlive("conn-1") {
		t.Error("expected connection to be alive")
	}
}

func TestWebSocketHeartbeat_RecordPong(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	wsh.RegisterConnection("conn-1", "node-1", "127.0.0.1")
	wsh.RecordPong("conn-1")

	if !wsh.IsAlive("conn-1") {
		t.Error("expected connection to be alive after pong")
	}
}

func TestWebSocketHeartbeat_RecordPong_Nonexistent(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	// Should not panic
	wsh.RecordPong("nonexistent")
}

func TestWebSocketHeartbeat_UnregisterConnection(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	wsh.RegisterConnection("conn-1", "node-1", "127.0.0.1")
	wsh.UnregisterConnection("conn-1")

	if wsh.IsAlive("conn-1") {
		t.Error("expected connection to not be alive after unregister")
	}
}

func TestWebSocketHeartbeat_GetStats(t *testing.T) {
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	wsh.RegisterConnection("conn-1", "node-1", "127.0.0.1")
	wsh.RegisterConnection("conn-2", "node-2", "127.0.0.1")

	stats := wsh.GetStats()
	if stats["total_connections"] != 2 {
		t.Errorf("expected total_connections 2, got %v", stats["total_connections"])
	}
	if stats["alive_connections"] != 2 {
		t.Errorf("expected alive_connections 2, got %v", stats["alive_connections"])
	}
}

func TestWebSocketHeartbeat_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	wsh.Middleware()(c)
	c.Next()

	connID, exists := c.Get("connection_id")
	if !exists {
		t.Error("expected connection_id to be set")
	}
	if connID == "" {
		t.Error("expected non-empty connection_id")
	}
}

func TestWebSocketHeartbeat_Middleware_WithConnectionID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	wsh := NewWebSocketHeartbeat(30*time.Second, 60*time.Second, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("X-Connection-ID", "custom-conn")

	wsh.Middleware()(c)
	c.Next()

	connID, exists := c.Get("connection_id")
	if !exists {
		t.Error("expected connection_id to be set")
	}
	if connID != "custom-conn" {
		t.Errorf("expected connection_id custom-conn, got %v", connID)
	}
}

func TestRequestValidator_New(t *testing.T) {
	rv := NewRequestValidator(1024)
	if rv == nil {
		t.Fatal("NewRequestValidator returned nil")
	}
	if rv.rules == nil {
		t.Error("expected rules map to be initialized")
	}
	if rv.maxBodySize != 1024 {
		t.Errorf("expected maxBodySize 1024, got %d", rv.maxBodySize)
	}
}

func TestRequestValidator_AddRule(t *testing.T) {
	rv := NewRequestValidator(1024)

	rv.AddRule("/api/test", "POST", []string{"field1"}, 512)

	key := "POST:/api/test"
	if _, exists := rv.rules[key]; !exists {
		t.Error("expected rule to be added")
	}
}

func TestRequestValidator_Middleware_NoRule(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	rv.Middleware()(c)
	c.Next()

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequestValidator_Middleware_WithRule_OverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	rv.AddRule("/api/test", "POST", []string{"field1"}, 100)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/test", nil)
	c.Request.ContentLength = 200

	rv.Middleware()(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d", w.Code)
	}
}

func TestRequestValidator_Middleware_WithRule_UnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	rv.AddRule("/api/test", "POST", []string{"field1"}, 1000)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/test", nil)
	c.Request.ContentLength = 100
	c.Request.Body = nil

	rv.Middleware()(c)
	c.Next()

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRequestValidator_ValidateBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)

	err := rv.ValidateBody(c)
	if err != nil {
		t.Errorf("expected no error for nil context, got %v", err)
	}
}

func TestRequestValidator_ValidateBody_UnsupportedType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/test", nil)
	c.Request.Header.Set("Content-Type", "text/plain")

	err := rv.ValidateBody(c)
	if err == nil {
		t.Error("expected error for unsupported content type")
	}
}

func TestRequestValidator_ValidateBody_SupportedType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rv := NewRequestValidator(1024)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/test", nil)
	c.Request.Header.Set("Content-Type", "application/json")

	err := rv.ValidateBody(c)
	if err != nil {
		t.Errorf("expected no error for application/json, got %v", err)
	}
}

func TestNodeSelector_New(t *testing.T) {
	ns := NewNodeSelector(StrategyRandom, nil)
	if ns == nil {
		t.Fatal("NewNodeSelector returned nil")
	}
	if ns.strategy != StrategyRandom {
		t.Errorf("expected strategy random, got %s", ns.strategy)
	}
}

func TestNodeSelector_Select_Empty(t *testing.T) {
	ns := NewNodeSelector(StrategyRandom, nil)

	result := ns.Select([]string{})
	if result != "" {
		t.Errorf("expected empty string for no nodes, got %q", result)
	}
}

func TestNodeSelector_Select_Single(t *testing.T) {
	ns := NewNodeSelector(StrategyRandom, nil)

	result := ns.Select([]string{"node-1"})
	if result != "node-1" {
		t.Errorf("expected node-1, got %q", result)
	}
}

func TestNodeSelector_Select_Random(t *testing.T) {
	ns := NewNodeSelector(StrategyRandom, nil)

	nodes := []string{"node-1", "node-2", "node-3"}
	result := ns.Select(nodes)

	found := false
	for _, n := range nodes {
		if result == n {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected result to be one of %v, got %q", nodes, result)
	}
}

func TestNodeSelector_Select_LeastLoad(t *testing.T) {
	ns := NewNodeSelector(StrategyLeastLoad, nil)

	nodes := []string{"node-1", "node-2"}
	result := ns.Select(nodes)

	if result != "node-1" {
		t.Errorf("expected node-1 for least_load, got %q", result)
	}
}

func TestNodeSelector_Select_LowestLatency_NoTracker(t *testing.T) {
	ns := NewNodeSelector(StrategyLowestLatency, nil)

	nodes := []string{"node-1", "node-2"}
	result := ns.Select(nodes)

	if result != "node-1" {
		t.Errorf("expected node-1 when no latency tracker, got %q", result)
	}
}

func TestNodeSelector_Select_HighestRep(t *testing.T) {
	ns := NewNodeSelector(StrategyHighestRep, nil)

	nodes := []string{"node-1", "node-2"}
	result := ns.Select(nodes)

	if result != "node-1" {
		t.Errorf("expected node-1 for highest_reputation, got %q", result)
	}
}

func TestNodeSelector_Select_RoundRobin(t *testing.T) {
	ns := NewNodeSelector(StrategyRoundRobin, nil)

	nodes := []string{"node-1", "node-2", "node-3"}

	r1 := ns.Select(nodes)
	r2 := ns.Select(nodes)
	r3 := ns.Select(nodes)

	if r1 != "node-1" {
		t.Errorf("expected node-1 for first round robin, got %q", r1)
	}
	if r2 != "node-2" {
		t.Errorf("expected node-2 for second round robin, got %q", r2)
	}
	if r3 != "node-3" {
		t.Errorf("expected node-3 for third round robin, got %q", r3)
	}
}

func TestNodeSelector_SetStrategy(t *testing.T) {
	ns := NewNodeSelector(StrategyRandom, nil)

	ns.SetStrategy(StrategyLeastLoad)
	if ns.GetStrategy() != StrategyLeastLoad {
		t.Errorf("expected strategy least_load, got %s", ns.GetStrategy())
	}
}

func TestNodeSelectorConstants(t *testing.T) {
	if StrategyRandom != "random" {
		t.Errorf("expected StrategyRandom to be random")
	}
	if StrategyLeastLoad != "least_load" {
		t.Errorf("expected StrategyLeastLoad to be least_load")
	}
	if StrategyLowestLatency != "lowest_latency" {
		t.Errorf("expected StrategyLowestLatency to be lowest_latency")
	}
	if StrategyHighestRep != "highest_reputation" {
		t.Errorf("expected StrategyHighestRep to be highest_reputation")
	}
	if StrategyRoundRobin != "round_robin" {
		t.Errorf("expected StrategyRoundRobin to be round_robin")
	}
}
