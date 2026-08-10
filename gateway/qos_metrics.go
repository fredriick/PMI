package gateway

import (
	"math"
	"sync"
	"time"
)

type QoSReporter struct {
	mu sync.Mutex
}

type QoSMetrics struct {
	NodeID         string
	PacketLoss     float64
	JitterMs       float64
	ThroughputMbps float64
	LatencyMs      float64
	LastMeasured   time.Time
}

func NewQoSReporter() *QoSReporter {
	return &QoSReporter{}
}

func (q *QoSReporter) RecordMetrics(nodeID string, packetLoss, jitterMs, throughputMbps, latencyMs float64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	_ = QoSMetrics{
		NodeID:         nodeID,
		PacketLoss:     packetLoss,
		JitterMs:       jitterMs,
		ThroughputMbps: throughputMbps,
		LatencyMs:      latencyMs,
		LastMeasured:   time.Now(),
	}
}

func (q *QoSReporter) CalculateHealthScore(metrics *QoSMetrics) float64 {
	if metrics == nil {
		return 0
	}

	latencyScore := 100.0 - math.Min(metrics.LatencyMs/10, 100)
	jitterScore := 100.0 - math.Min(metrics.JitterMs, 100)
	lossScore := 100.0 - math.Min(metrics.PacketLoss*10, 100)

	avg := (latencyScore + jitterScore + lossScore) / 3.0
	return math.Max(0, math.Min(100, avg))
}
