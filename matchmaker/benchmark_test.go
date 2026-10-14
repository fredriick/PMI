package matchmaker

import (
	"testing"
	"time"
)

func TestNodeBenchmark_New(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	if nb == nil {
		t.Fatal("NewNodeBenchmark returned nil")
	}
	if nb.results == nil {
		t.Error("expected results map to be initialized")
	}
}

func TestNodeBenchmark_CalculateAvg(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	if nb.calculateAvg(nil) != 0 {
		t.Error("expected 0 for empty values")
	}
	if nb.calculateAvg([]float64{10, 20, 30}) != 20 {
		t.Errorf("expected avg 20, got %f", nb.calculateAvg([]float64{10, 20, 30}))
	}
	if nb.calculateAvg([]float64{5}) != 5 {
		t.Errorf("expected avg 5, got %f", nb.calculateAvg([]float64{5}))
	}
}

func TestNodeBenchmark_CalculatePercentile(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	if nb.calculatePercentile(nil, 50) != 0 {
		t.Error("expected 0 for empty values")
	}
	values := []float64{10, 20, 30, 40, 50}
	p50 := nb.calculatePercentile(values, 50)
	if p50 != 30 {
		t.Errorf("expected p50=30, got %f", p50)
	}
}

func TestNodeBenchmark_CalculatePercentile_Single(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	p := nb.calculatePercentile([]float64{42}, 99)
	if p != 42 {
		t.Errorf("expected 42, got %f", p)
	}
}

func TestNodeBenchmark_GetResult_NotFound(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	result := nb.GetResult("nonexistent")
	if result != nil {
		t.Error("expected nil for nonexistent result")
	}
}

func TestNodeBenchmark_GetAllResults(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	results := nb.GetAllResults()
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestNodeBenchmark_RunBenchmark(t *testing.T) {
	nb := NewNodeBenchmark(nil)
	result := nb.RunBenchmark("node-1", 50*time.Millisecond)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.NodeID != "node-1" {
		t.Errorf("expected node-1, got %s", result.NodeID)
	}
	if result.LatencyAvg == 0 {
		t.Error("expected non-zero latency avg")
	}
	if result.SuccessRate == 0 {
		t.Error("expected non-zero success rate")
	}

	got := nb.GetResult("node-1")
	if got == nil {
		t.Error("expected result to be stored")
	}
	if got.NodeID != "node-1" {
		t.Errorf("expected stored result for node-1")
	}
}

func TestBenchmarkResult_Struct(t *testing.T) {
	br := &BenchmarkResult{
		NodeID:      "node-x",
		LatencyAvg:  25.5,
		LatencyP50:  20.0,
		LatencyP95:  45.0,
		LatencyP99:  60.0,
		Throughput:  100.0,
		SuccessRate: 98.5,
		RunDuration: time.Second,
		Timestamp:   time.Now(),
	}
	if br.NodeID != "node-x" {
		t.Error("unexpected NodeID")
	}
	if br.LatencyAvg != 25.5 {
		t.Error("unexpected LatencyAvg")
	}
}
