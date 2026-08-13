package gateway

import (
	"sync"
	"testing"
)

func TestNewQoSReporter(t *testing.T) {
	r := NewQoSReporter()
	if r == nil {
		t.Fatal("expected non-nil reporter")
	}
}

func TestQoS_RecordMetrics(t *testing.T) {
	r := NewQoSReporter()
	r.RecordMetrics("node-1", 0.1, 5.0, 100.0, 10.0)
}

func TestQoS_RecordMetricsConcurrent(t *testing.T) {
	r := NewQoSReporter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.RecordMetrics("node-1", float64(i), float64(i), float64(i), float64(i))
		}(i)
	}
	wg.Wait()
}

func TestQoS_CalculateHealthScore_PerfectMetrics(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      0,
		JitterMs:       0,
		PacketLoss:     0,
		ThroughputMbps: 1000,
	})
	if score != 100 {
		t.Fatalf("expected 100 for perfect metrics, got %f", score)
	}
}

func TestQoS_CalculateHealthScore_GoodMetrics(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      20,
		JitterMs:       5,
		PacketLoss:     0.5,
		ThroughputMbps: 100,
	})
	if score < 80 || score > 100 {
		t.Fatalf("expected score 80-100 for good metrics, got %f", score)
	}
}

func TestQoS_CalculateHealthScore_PoorMetrics(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      500,
		JitterMs:       200,
		PacketLoss:     20,
		ThroughputMbps: 1,
	})
	if score < 0 || score > 20 {
		t.Fatalf("expected score 0-20 for poor metrics, got %f", score)
	}
}

func TestQoS_CalculateHealthScore_ClampsToZero(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      2000,
		JitterMs:       500,
		PacketLoss:     100,
		ThroughputMbps: 0,
	})
	if score != 0 {
		t.Fatalf("expected 0 for extreme metrics, got %f", score)
	}
}

func TestQoS_CalculateHealthScore_WeightedAverage(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      50,
		JitterMs:       25,
		PacketLoss:     5,
		ThroughputMbps: 50,
	})
	latencyScore := 100.0 - 50.0/10.0
	jitterScore := 100.0 - 25.0
	lossScore := 100.0 - 5.0*10.0
	expected := (latencyScore + jitterScore + lossScore) / 3.0
	expected = max(0, min(100, expected))
	if score != expected {
		t.Fatalf("expected %f, got %f", expected, score)
	}
}

func TestQoS_CalculateHealthScore_NegativeValues(t *testing.T) {
	r := NewQoSReporter()
	score := r.CalculateHealthScore(&QoSMetrics{
		LatencyMs:      -10,
		JitterMs:       -5,
		PacketLoss:     -1,
		ThroughputMbps: 10,
	})
	if score < 0 || score > 100 {
		t.Fatalf("expected score 0-100 for negative metrics, got %f", score)
	}
}
