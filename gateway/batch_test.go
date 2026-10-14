package gateway

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestBatchProcessor_New(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)
	if bp == nil {
		t.Fatal("NewBatchProcessor returned nil")
	}
	if bp.maxBatchSize != 10 {
		t.Errorf("expected maxBatchSize 10, got %d", bp.maxBatchSize)
	}
	if bp.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", bp.timeout)
	}
	if bp.handlers == nil {
		t.Error("expected handlers map to be initialized")
	}
}

func TestBatchProcessor_RegisterHandler(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	handler := func(ops []map[string]interface{}) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"result": "ok"}}, nil
	}

	bp.RegisterHandler("test-op", handler)

	if _, exists := bp.handlers["test-op"]; !exists {
		t.Error("expected handler to be registered")
	}
}

func TestBatchProcessor_Process_Empty(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	resp, err := bp.Process(context.Background(), BatchRequest{})
	if err != nil {
		t.Fatalf("expected no error for empty batch, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

func TestBatchProcessor_Process_ExceedsMaxSize(t *testing.T) {
	bp := NewBatchProcessor(2, 5*time.Second)

	ops := make([]map[string]interface{}, 3)
	for i := range ops {
		ops[i] = map[string]interface{}{"type": "test"}
	}

	_, err := bp.Process(context.Background(), BatchRequest{Operations: ops})
	if err == nil {
		t.Error("expected error for batch exceeding max size")
	}
}

func TestBatchProcessor_Process_OperationMissingType(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	resp, err := bp.Process(context.Background(), BatchRequest{
		Operations: []map[string]interface{}{{"data": "value"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
}

func TestBatchProcessor_Process_UnknownOperation(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	resp, err := bp.Process(context.Background(), BatchRequest{
		Operations: []map[string]interface{}{{"type": "unknown-op"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
}

func TestBatchProcessor_Process_WithHandler(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	handler := func(ops []map[string]interface{}) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"result": "processed"}}, nil
	}
	bp.RegisterHandler("test-op", handler)

	resp, err := bp.Process(context.Background(), BatchRequest{
		Operations: []map[string]interface{}{{"type": "test-op", "data": "value"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(resp.Errors))
	}
	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
}

func TestBatchProcessor_Process_HandlerError(t *testing.T) {
	bp := NewBatchProcessor(10, 5*time.Second)

	handler := func(ops []map[string]interface{}) ([]map[string]interface{}, error) {
		return nil, &testError{msg: "handler failed"}
	}
	bp.RegisterHandler("test-op", handler)

	resp, err := bp.Process(context.Background(), BatchRequest{
		Operations: []map[string]interface{}{{"type": "test-op"}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(resp.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Errors))
	}
	if resp.Errors[0] != "handler failed" {
		t.Errorf("expected error 'handler failed', got %q", resp.Errors[0])
	}
}

func TestBatchProcessor_Process_ContextTimeout(t *testing.T) {
	bp := NewBatchProcessor(10, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := bp.Process(ctx, BatchRequest{
		Operations: []map[string]interface{}{{"type": "test-op"}},
	})
	if err == nil {
		t.Error("expected error due to context timeout")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestBatchProcessor_batchHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bp := NewBatchProcessor(10, 5*time.Second)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/batch", nil)

	bp.batchHandler(c)

	if w.Code != 400 {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestBatchProcessor_batchHandler_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	bp := NewBatchProcessor(10, 5*time.Second)

	body := `{"operations":[{"type":"test-op","data":"value"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/batch", bytes.NewReader([]byte(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	bp.batchHandler(c)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestKeyPriority_New(t *testing.T) {
	kp := NewKeyPriority(nil)
	if kp == nil {
		t.Fatal("NewKeyPriority returned nil")
	}
	if kp.priorities == nil {
		t.Error("expected priorities map to be initialized")
	}
	if kp.redis != nil {
		t.Error("expected nil redis")
	}
}

func TestKeyPriority_SetGetPriority(t *testing.T) {
	kp := NewKeyPriority(nil)

	kp.SetPriority("api-key-1", 5)
	if kp.GetPriority("api-key-1") != 5 {
		t.Errorf("expected priority 5, got %d", kp.GetPriority("api-key-1"))
	}

	if kp.GetPriority("nonexistent") != 0 {
		t.Errorf("expected priority 0 for nonexistent key, got %d", kp.GetPriority("nonexistent"))
	}
}

func TestKeyPriority_GetWeightedPriority(t *testing.T) {
	kp := NewKeyPriority(nil)

	kp.SetPriority("api-key-1", 2)
	weighted := kp.GetWeightedPriority("api-key-1", 10.0)
	if weighted != 30.0 {
		t.Errorf("expected weighted priority 30.0, got %f", weighted)
	}

	weighted = kp.GetWeightedPriority("nonexistent", 10.0)
	if weighted != 10.0 {
		t.Errorf("expected weighted priority 10.0 for nonexistent, got %f", weighted)
	}
}

func TestKeyPriority_NilRedis_SetPriority(t *testing.T) {
	kp := NewKeyPriority(nil)

	kp.SetPriority("api-key-1", 3)
	if kp.GetPriority("api-key-1") != 3 {
		t.Error("expected priority to be stored in memory")
	}
}

func TestKeyPriority_Middleware_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kp := NewKeyPriority(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)

	kp.Middleware()(c)
	c.Next()
}

func TestKeyPriority_Middleware_WithAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	kp := NewKeyPriority(nil)
	kp.SetPriority("test-key", 3)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/test", nil)
	c.Request.Header.Set("Authorization", "Bearer test-key")

	kp.Middleware()(c)
	c.Next()

	priority, exists := c.Get("request_priority")
	if !exists {
		t.Error("expected request_priority to be set")
	}
	if priority != 3 {
		t.Errorf("expected request_priority 3, got %v", priority)
	}
}

func TestNodeHealthScore_New_DefaultWeights(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})
	if nhs == nil {
		t.Fatal("NewNodeHealthScore returned nil")
	}
	if nhs.scores == nil {
		t.Error("expected scores map to be initialized")
	}
	if nhs.redis != nil {
		t.Error("expected nil redis")
	}
	if nhs.weights.LatencyWeight != 30 {
		t.Errorf("expected default LatencyWeight 30, got %f", nhs.weights.LatencyWeight)
	}
}

func TestNodeHealthScore_CalculateScore(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{
		LatencyWeight:     30,
		LoadWeight:        20,
		SuccessRateWeight: 30,
		ReputationWeight:  15,
		UptimeWeight:      5,
	})

	score := nhs.CalculateScore("node-1", 25, 5, 0.95, 90.0, 720.0)

	if score <= 0 {
		t.Error("expected positive score")
	}
	if score > 100 {
		t.Errorf("expected score <= 100, got %f", score)
	}
}

func TestNodeHealthScore_GetScore(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	nhs.CalculateScore("node-1", 25, 5, 0.95, 90.0, 720.0)

	hs := nhs.GetScore("node-1")
	if hs == nil {
		t.Fatal("expected non-nil HealthScore")
	}
	if hs.NodeID != "node-1" {
		t.Errorf("expected NodeID node-1, got %q", hs.NodeID)
	}
}

func TestNodeHealthScore_GetScore_Nonexistent(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	hs := nhs.GetScore("nonexistent")
	if hs != nil {
		t.Errorf("expected nil for nonexistent node, got %v", hs)
	}
}

func TestNodeHealthScore_GetAllScores(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	nhs.CalculateScore("node-1", 25, 5, 0.95, 90.0, 720.0)
	nhs.CalculateScore("node-2", 100, 10, 0.80, 80.0, 360.0)

	scores := nhs.GetAllScores()
	if len(scores) != 2 {
		t.Errorf("expected 2 scores, got %d", len(scores))
	}
}

func TestNodeHealthScore_GetTopNodes(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{
		LatencyWeight:     30,
		LoadWeight:        20,
		SuccessRateWeight: 30,
		ReputationWeight:  15,
		UptimeWeight:      5,
	})

	nhs.CalculateScore("node-1", 25, 5, 0.95, 90.0, 720.0)  // High score
	nhs.CalculateScore("node-2", 500, 100, 0.50, 10.0, 1.0)  // Low score

	top := nhs.GetTopNodes(1)
	if len(top) != 1 {
		t.Errorf("expected 1 top node, got %d", len(top))
	}
	if top[0] != "node-1" {
		t.Errorf("expected node-1 as top node, got %q", top[0])
	}
}

func TestCalculateLatencyScore(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	if nhs.calculateLatencyScore(25) != 100 {
		t.Error("expected latency score 100 for 25ms")
	}
	if nhs.calculateLatencyScore(500) != 0 {
		t.Error("expected latency score 0 for 500ms")
	}
	if nhs.calculateLatencyScore(50) != 100 {
		t.Error("expected latency score 100 for 50ms")
	}
}

func TestCalculateLoadScore(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	if nhs.calculateLoadScore(5) != 100 {
		t.Error("expected load score 100 for load 5")
	}
	if nhs.calculateLoadScore(100) != 0 {
		t.Error("expected load score 0 for load 100")
	}
	if nhs.calculateLoadScore(10) != 100 {
		t.Error("expected load score 100 for load 10")
	}
}

func TestCalculateUptimeScore(t *testing.T) {
	nhs := NewNodeHealthScore(nil, HealthWeights{})

	if nhs.calculateUptimeScore(720) != 100 {
		t.Error("expected uptime score 100 for 720 hours")
	}
	if nhs.calculateUptimeScore(360) != 50 {
		t.Errorf("expected uptime score 50 for 360 hours, got %f", nhs.calculateUptimeScore(360))
	}
	if nhs.calculateUptimeScore(0) != 0 {
		t.Error("expected uptime score 0 for 0 hours")
	}
}
