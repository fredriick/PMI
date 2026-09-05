package matchmaker

import (
	"sync"
	"testing"
	"time"
)

func TestLatencyTracker_NoRedis(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 10)

	if lt.GetAvgLatency("node1") != 0 {
		t.Error("expected 0 latency for unknown node")
	}

	lt.RecordLatency("node1", 100)

	if lt.GetAvgLatency("node1") != 100 {
		t.Errorf("expected avg latency 100, got %d", lt.GetAvgLatency("node1"))
	}
}

func TestLatencyTracker_MultipleSamples(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 100)

	lt.RecordLatency("node1", 100)
	lt.RecordLatency("node1", 200)
	lt.RecordLatency("node1", 300)

	avg := lt.GetAvgLatency("node1")
	if avg != 200 {
		t.Errorf("expected avg latency 200, got %d", avg)
	}
}

func TestLatencyTracker_MaxSamples(t *testing.T) {
	maxSamples := 5
	lt := NewLatencyTracker(nil, 5*time.Minute, maxSamples)

	for i := 1; i <= maxSamples+5; i++ {
		lt.RecordLatency("node1", int64(i*100))
	}

	samples := lt.latencies["node1"].samples
	if len(samples) > maxSamples {
		t.Errorf("expected at most %d samples, got %d", maxSamples, len(samples))
	}
}

func TestLatencyTracker_LatencyStats(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 10)

	lt.RecordLatency("node1", 10)
	lt.RecordLatency("node1", 50)
	lt.RecordLatency("node1", 100)

	avg, min, max := lt.GetLatencyStats("node1")
	if avg != 53 {
		t.Errorf("expected avg 53, got %d", avg)
	}
	if min != 10 {
		t.Errorf("expected min 10, got %d", min)
	}
	if max != 100 {
		t.Errorf("expected max 100, got %d", max)
	}
}

func TestLatencyTracker_LatencyStats_UnknownNode(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 10)

	avg, min, max := lt.GetLatencyStats("unknown")
	if avg != 0 || min != 0 || max != 0 {
		t.Errorf("expected all zeros for unknown node, got avg=%d min=%d max=%d", avg, min, max)
	}
}

func TestLatencyTracker_GetTopNodes_NoRedis(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 10)

	lt.RecordLatency("node1", 50)
	lt.RecordLatency("node2", 30)
	lt.RecordLatency("node3", 100)

	top := lt.GetTopNodes(2)
	if len(top) != 2 {
		t.Fatalf("expected 2 top nodes, got %d", len(top))
	}

	if top[0] != "node2" {
		t.Errorf("expected node2 first (lowest latency), got '%s'", top[0])
	}
}

func TestRequestQueue_Basic(t *testing.T) {
	rq := NewRequestQueue(10)

	req := &QueuedRequest{
		Request:   "test-request",
		Result:    make(chan interface{}, 1),
		CreatedAt: time.Now(),
	}

	if !rq.Enqueue("node1", req) {
		t.Error("expected Enqueue to succeed")
	}

	if rq.GetQueueSize("node1") != 1 {
		t.Errorf("expected queue size 1, got %d", rq.GetQueueSize("node1"))
	}

	dequeued := rq.Dequeue("node1")
	if dequeued == nil {
		t.Fatal("expected dequeued request")
	}
	if dequeued.Request != "test-request" {
		t.Errorf("expected request 'test-request', got %v", dequeued.Request)
	}
}

func TestRequestQueue_DequeueEmpty(t *testing.T) {
	rq := NewRequestQueue(10)

	if rq.Dequeue("nonexistent") != nil {
		t.Error("expected nil for empty queue")
	}
}

func TestRequestQueue_MaxQueueSize(t *testing.T) {
	rq := NewRequestQueue(2)

	req := &QueuedRequest{Result: make(chan interface{}, 1)}
	rq.Enqueue("node1", req)
	rq.Enqueue("node1", req)

	if rq.Enqueue("node1", req) {
		t.Error("expected Enqueue to fail when queue is full (3rd item)")
	}

	if rq.GetQueueSize("node1") != 2 {
		t.Errorf("expected queue size 2, got %d", rq.GetQueueSize("node1"))
	}
}

func TestRequestQueue_RemoveQueue(t *testing.T) {
	rq := NewRequestQueue(10)

	rq.Enqueue("node1", &QueuedRequest{Result: make(chan interface{}, 1)})
	if rq.GetQueueSize("node1") != 1 {
		t.Error("expected queue size 1 before removal")
	}

	rq.RemoveQueue("node1")

	if rq.GetQueueSize("node1") != 0 {
		t.Error("expected queue size 0 after removal")
	}
}

func TestReputationDecay_Basic(t *testing.T) {
	rd := NewReputationDecay(0.1, 1*time.Hour, 10.0)

	if rd.GetReputation("unknown") != 100.0 {
		t.Errorf("expected default reputation 100.0 for unknown node, got %.2f", rd.GetReputation("unknown"))
	}

	rd.SetReputation("node1", 90.0)
	if rd.GetReputation("node1") != 90.0 {
		t.Errorf("expected reputation 90.0, got %.2f", rd.GetReputation("node1"))
	}
}

func TestReputationDecay_Increment(t *testing.T) {
	rd := NewReputationDecay(0.1, 1*time.Hour, 10.0)

	rd.SetReputation("node1", 50.0)
	rd.IncrementReputation("node1", 30.0)

	if rd.GetReputation("node1") != 80.0 {
		t.Errorf("expected reputation 80.0, got %.2f", rd.GetReputation("node1"))
	}
}

func TestReputationDecay_IncrementClamped(t *testing.T) {
	rd := NewReputationDecay(0.1, 1*time.Hour, 10.0)

	rd.IncrementReputation("node1", 60.0)

	if rd.GetReputation("node1") != 100.0 {
		t.Errorf("expected reputation clamped to 100.0, got %.2f", rd.GetReputation("node1"))
	}
}

func TestReputationDecay_IncrementNewNode(t *testing.T) {
	rd := NewReputationDecay(0.1, 1*time.Hour, 10.0)

	rd.IncrementReputation("newnode", 5.0)

	if rd.GetReputation("newnode") != 100.0 {
		t.Errorf("expected reputation 100.0 (clamped from 105.0), got %.2f", rd.GetReputation("newnode"))
	}
}

func TestNodeBenchmark_NoRedis(t *testing.T) {
	nb := NewNodeBenchmark(nil)

	result := nb.RunBenchmark("node1", 100*time.Millisecond)
	if result == nil {
		t.Fatal("expected non-nil benchmark result")
	}
	if result.NodeID != "node1" {
		t.Errorf("expected node_id 'node1', got '%s'", result.NodeID)
	}
	if result.LatencyAvg == 0 {
		t.Error("expected non-zero latency average")
	}
	if result.SuccessRate != 100.0 {
		t.Errorf("expected 100%% success rate, got %.2f", result.SuccessRate)
	}

	got := nb.GetResult("node1")
	if got == nil {
		t.Error("expected result from GetResult")
	}
	if got.NodeID != "node1" {
		t.Errorf("expected node_id 'node1', got '%s'", got.NodeID)
	}

	all := nb.GetAllResults()
	if len(all) != 1 {
		t.Errorf("expected 1 result, got %d", len(all))
	}
}

func TestNewBenchmarkResult_AllResults(t *testing.T) {
	nb := NewNodeBenchmark(nil)

	nb.RunBenchmark("node1", 20*time.Millisecond)
	nb.RunBenchmark("node2", 20*time.Millisecond)

	all := nb.GetAllResults()
	if len(all) != 2 {
		t.Errorf("expected 2 results, got %d", len(all))
	}

	nb.mu.RLock()
	defer nb.mu.RUnlock()
	if nb.results["nonexistent"] != nil {
		t.Error("expected nil for nonexistent node result")
	}
}

func TestBenchmarkResult_ConcurrentAccess(t *testing.T) {
	lt := NewLatencyTracker(nil, 5*time.Minute, 10)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			lt.RecordLatency("node1", int64(n))
		}(i)
	}
	wg.Wait()

	if lt.GetAvgLatency("node1") <= 0 {
		t.Error("expected positive avg latency after concurrent records")
	}
}
