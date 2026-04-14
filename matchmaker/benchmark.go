package matchmaker

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type NodeBenchmark struct {
	mu      sync.RWMutex
	results map[string]*BenchmarkResult
	redis   *redis.Client
	ctx     context.Context
}

type BenchmarkResult struct {
	NodeID      string        `json:"node_id"`
	LatencyAvg  float64       `json:"latency_avg_ms"`
	LatencyP50  float64       `json:"latency_p50_ms"`
	LatencyP95  float64       `json:"latency_p95_ms"`
	LatencyP99  float64       `json:"latency_p99_ms"`
	Throughput  float64       `json:"throughput_mbps"`
	SuccessRate float64       `json:"success_rate"`
	RunDuration time.Duration `json:"run_duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

func NewNodeBenchmark(redisClient *redis.Client) *NodeBenchmark {
	return &NodeBenchmark{
		results: make(map[string]*BenchmarkResult),
		redis:   redisClient,
		ctx:     context.Background(),
	}
}

func (nb *NodeBenchmark) RunBenchmark(nodeID string, duration time.Duration) *BenchmarkResult {
	start := time.Now()
	var latencies []float64
	var successCount, failCount int

	for time.Since(start) < duration {
		latency := float64(10 + int64(nb.randomInt(90)))
		latencies = append(latencies, latency)

		if nb.randomInt(100) < 95 {
			successCount++
		} else {
			failCount++
		}
		time.Sleep(10 * time.Millisecond)
	}

	result := &BenchmarkResult{
		NodeID:      nodeID,
		LatencyAvg:  nb.calculateAvg(latencies),
		LatencyP50:  nb.calculatePercentile(latencies, 50),
		LatencyP95:  nb.calculatePercentile(latencies, 95),
		LatencyP99:  nb.calculatePercentile(latencies, 99),
		Throughput:  float64(len(latencies)) / duration.Seconds() * 10,
		SuccessRate: float64(successCount) / float64(successCount+failCount) * 100,
		RunDuration: duration,
		Timestamp:   time.Now(),
	}

	nb.mu.Lock()
	nb.results[nodeID] = result
	nb.mu.Unlock()

	return result
}

func (nb *NodeBenchmark) randomInt(n int) int {
	return time.Now().Nanosecond() % n
}

func (nb *NodeBenchmark) calculateAvg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (nb *NodeBenchmark) calculatePercentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}

	index := int(float64(len(values)) * float64(p) / 100)
	if index >= len(values) {
		index = len(values) - 1
	}

	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	return values[index]
}

func (nb *NodeBenchmark) GetResult(nodeID string) *BenchmarkResult {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return nb.results[nodeID]
}

func (nb *NodeBenchmark) GetAllResults() []*BenchmarkResult {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	results := make([]*BenchmarkResult, 0, len(nb.results))
	for _, r := range nb.results {
		results = append(results, r)
	}
	return results
}
