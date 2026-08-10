package gateway

import (
	"testing"
)

func TestHashKey(t *testing.T) {
	h := hashKey("test-key")
	if len(h) != 64 {
		t.Fatalf("expected SHA256 hex length 64, got %d", len(h))
	}
}

func TestGenerateKey(t *testing.T) {
	k := generateKey(32)
	if len(k) != 64 {
		t.Fatalf("expected 32 bytes = 64 hex chars, got %d", len(k))
	}
}

func TestAPIKeyScopeValidation(t *testing.T) {
	svc := NewAPIKeyService(nil)

	if svc.HasScope("", "read") {
		t.Fatal("expected false for empty key")
	}
}

func TestQoSHealthScore(t *testing.T) {
	reporter := NewQoSReporter()

	metrics := &QoSMetrics{
		LatencyMs:  10,
		JitterMs:   5,
		PacketLoss: 0.1,
	}
	score := reporter.CalculateHealthScore(metrics)
	if score < 0 || score > 100 {
		t.Fatalf("expected score 0-100, got %f", score)
	}
}

func TestQoSHealthScoreNil(t *testing.T) {
	reporter := NewQoSReporter()
	score := reporter.CalculateHealthScore(nil)
	if score != 0 {
		t.Fatalf("expected 0 for nil metrics, got %f", score)
	}
}

func TestQoSHealthScoreHighLoss(t *testing.T) {
	reporter := NewQoSReporter()

	metrics := &QoSMetrics{
		LatencyMs:  1000,
		JitterMs:   500,
		PacketLoss: 50,
	}
	score := reporter.CalculateHealthScore(metrics)
	if score != 0 {
		t.Fatalf("expected 0 for extreme metrics, got %f", score)
	}
}
