package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"proxymesh/gateway"
	"proxymesh/internal/config"
)

func TestBenchmarkNodeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.POST("/api/admin/benchmark", benchmarkNodeHandler(nil))

	body, _ := json.Marshal(map[string]interface{}{
		"node_id": "test-node",
		"count":   5,
	})
	req := httptest.NewRequest("POST", "/api/admin/benchmark", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("benchmark status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Fatalf("expected success, got %v", resp)
	}
	if resp["iterations"] != float64(5) {
		t.Fatalf("expected 5 iterations, got %v", resp["iterations"])
	}
	if resp["node_id"] != "test-node" {
		t.Fatalf("expected node_id test-node, got %v", resp["node_id"])
	}
}

func TestBenchmarkNodeHandler_MissingNodeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.POST("/api/admin/benchmark", benchmarkNodeHandler(nil))

	body, _ := json.Marshal(map[string]interface{}{})
	req := httptest.NewRequest("POST", "/api/admin/benchmark", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", w.Code)
	}
}

func TestBenchmarkNodeHandler_DefaultCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.POST("/api/admin/benchmark", benchmarkNodeHandler(nil))

	body, _ := json.Marshal(map[string]interface{}{
		"node_id": "test-node",
	})
	req := httptest.NewRequest("POST", "/api/admin/benchmark", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("benchmark status = %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["iterations"] != float64(10) {
		t.Fatalf("expected default 10 iterations, got %v", resp["iterations"])
	}
}

func TestReferralsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/api/admin/referrals/:code", getReferralsHandler(nil))

	req := httptest.NewRequest("GET", "/api/admin/referrals/ABC123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("referrals status = %d, body = %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Fatalf("expected success, got %v", resp)
	}
	if resp["code"] != "ABC123" {
		t.Fatalf("expected code ABC123, got %v", resp["code"])
	}
}

func TestReferralsHandler_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/api/admin/referrals/:code", getReferralsHandler(nil))

	req := httptest.NewRequest("GET", "/api/admin/referrals/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Fatalf("expected bad request or not found, got %d", w.Code)
	}
}

func TestRateLimitTiers(t *testing.T) {
	tiers := gateway.NewRateLimitTiers(nil)

	tier := tiers.GetTier("free")
	if tier.Requests != 10 {
		t.Fatalf("expected free tier 10 requests, got %d", tier.Requests)
	}

	tier = tiers.GetTier("basic")
	if tier.Requests != 100 {
		t.Fatalf("expected basic tier 100 requests, got %d", tier.Requests)
	}

	tier = tiers.GetTier("premium")
	if tier.Requests != 1000 {
		t.Fatalf("expected premium tier 1000 requests, got %d", tier.Requests)
	}

	tier = tiers.GetTier("enterprise")
	if tier.Requests != 10000 {
		t.Fatalf("expected enterprise tier 10000 requests, got %d", tier.Requests)
	}

	tier = tiers.GetTier("unknown")
	if tier.Requests != 10 {
		t.Fatalf("expected unknown tier to default to free (10), got %d", tier.Requests)
	}
}

func TestRateLimitTiers_List(t *testing.T) {
	tiers := gateway.NewRateLimitTiers(nil)
	list := tiers.ListTiers()
	if len(list) != 4 {
		t.Fatalf("expected 4 tiers, got %d", len(list))
	}
}

func TestGenerateReferralCode(t *testing.T) {
	code := gateway.GenerateReferralCode()
	if len(code) < 8 {
		t.Fatalf("expected referral code to be at least 8 chars, got %s", code)
	}
	if code[:4] != "REF-" {
		t.Fatalf("expected referral code to start with REF-, got %s", code)
	}
}

func TestScalingPolicyManager(t *testing.T) {
	spm := gateway.NewScalingPolicyManager()
	policies := spm.ListPolicies()
	if len(policies) != 2 {
		t.Fatalf("expected 2 default policies, got %d", len(policies))
	}

	policy := spm.GetPolicy(policies[0].ID)
	if policy == nil {
		t.Fatal("expected to get policy by ID")
	}
	if policy.Name != "Critical Nodes Alert" {
		t.Fatalf("expected policy name 'Critical Nodes Alert', got %s", policy.Name)
	}
}

func TestScalingEvaluation(t *testing.T) {
	eval := gateway.ScalingEvaluation{
		PolicyID:   "test-1",
		PolicyName: "Test Policy",
		Triggered:  true,
		Action:     gateway.ActionScaleUp,
	}
	if eval.PolicyID != "test-1" {
		t.Fatalf("expected policy_id test-1, got %s", eval.PolicyID)
	}
	if eval.Action != gateway.ActionScaleUp {
		t.Fatalf("expected action scale_up, got %s", eval.Action)
	}
}

func TestConfigRequestLoggingEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.RequestLoggingEnabled = true
	if !cfg.Gateway.RequestLoggingEnabled {
		t.Fatal("expected RequestLoggingEnabled to be true")
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	
	logger := gateway.NewRequestLogger(nil, true)
	r.Use(logger.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequestLoggerDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	
	logger := gateway.NewRequestLogger(nil, false)
	r.Use(logger.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
