package gateway

import (
	"testing"
)

func TestNewCostEstimator_Defaults(t *testing.T) {
	ce := NewCostEstimator(nil)
	if ce == nil {
		t.Fatal("NewCostEstimator(nil) returned nil")
	}
	if ce.baseCosts == nil {
		t.Error("expected baseCosts map to be initialized")
	}
	if ce.clientCosts == nil {
		t.Error("expected clientCosts map to be initialized")
	}
	if ce.redis != nil {
		t.Error("expected nil redis when nil passed")
	}
}

func TestCostEstimator_EstimateRequest_Base(t *testing.T) {
	ce := NewCostEstimator(nil)

	cost := ce.EstimateRequest("client-1", 0, 0, 0)

	expected := ce.baseCosts["default"].BaseCost + ce.baseCosts["default"].PerRequest
	if cost != expected {
		t.Errorf("expected cost %f, got %f", expected, cost)
	}
}

func TestCostEstimator_EstimateRequest_WithBytesSent(t *testing.T) {
	ce := NewCostEstimator(nil)

	costNoBytes := ce.EstimateRequest("client-1", 0, 0, 0)
	costWithBytes := ce.EstimateRequest("client-1", 0, 1024*1024*1024, 0)

	if costWithBytes <= costNoBytes {
		t.Error("expected cost to increase with bytes sent")
	}
}

func TestCostEstimator_EstimateRequest_WithBytesReceived(t *testing.T) {
	ce := NewCostEstimator(nil)

	costNoBytes := ce.EstimateRequest("client-1", 0, 0, 0)
	costWithBytes := ce.EstimateRequest("client-1", 1024*1024*1024, 0, 0)

	if costWithBytes <= costNoBytes {
		t.Error("expected cost to increase with bytes received")
	}
}

func TestCostEstimator_EstimateRequest_WithDuration(t *testing.T) {
	ce := NewCostEstimator(nil)

	costNoDuration := ce.EstimateRequest("client-1", 0, 0, 0)
	costWithDuration := ce.EstimateRequest("client-1", 0, 0, 60)

	if costWithDuration <= costNoDuration {
		t.Error("expected cost to increase with duration")
	}
}

func TestCostEstimator_EstimateRequest_ClientMultiplier(t *testing.T) {
	ce := NewCostEstimator(nil)

	baseCost := ce.EstimateRequest("client-1", 0, 0, 0)

	ce.SetClientCost("client-2", 2.0)
	costWithMultiplier := ce.EstimateRequest("client-2", 0, 0, 0)

	if costWithMultiplier != baseCost*2.0 {
		t.Errorf("expected cost %f with 2x multiplier, got %f", baseCost*2.0, costWithMultiplier)
	}
}

func TestCostEstimator_EstimateMonthly(t *testing.T) {
	ce := NewCostEstimator(nil)

	monthlyCost := ce.EstimateMonthly("client-1", 100, 1024*1024, 1024*1024, 10)

	if monthlyCost <= 0 {
		t.Error("expected monthly cost to be positive")
	}

	// Monthly cost should be roughly 30x daily cost
	dailyCost := ce.EstimateRequest("client-1", 1024*1024, 1024*1024, 10) * 100
	if monthlyCost < dailyCost*29 || monthlyCost > dailyCost*31 {
		t.Errorf("expected monthly cost close to %f, got %f", dailyCost*30, monthlyCost)
	}
}

func TestCostEstimator_SetAndGetClientCost(t *testing.T) {
	ce := NewCostEstimator(nil)

	// Default multiplier is 1.0
	if ce.GetClientCost("unknown-client") != 1.0 {
		t.Error("expected default client cost multiplier 1.0")
	}

	ce.SetClientCost("client-1", 0.5)
	if ce.GetClientCost("client-1") != 0.5 {
		t.Errorf("expected client cost 0.5, got %f", ce.GetClientCost("client-1"))
	}
}

func TestCostEstimator_NilRedis_SetClientCost(t *testing.T) {
	ce := NewCostEstimator(nil)

	// Should not panic when redis is nil
	ce.SetClientCost("client-1", 2.0)
	if ce.GetClientCost("client-1") != 2.0 {
		t.Error("expected client cost to be stored in memory")
	}
}
