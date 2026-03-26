package gateway

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	requestsTotal   uint64
	requestsSuccess uint64
	requestsFailed  uint64
	rateLimited     uint64
	nodesSelected   uint64
	activeSessions  uint64
	startTime       time.Time

	latencyTotalMs uint64
	latencyCount   uint64
}

var metrics = &Metrics{
	startTime: time.Now(),
}

func (m *Metrics) IncRequestsTotal() {
	atomic.AddUint64(&m.requestsTotal, 1)
}

func (m *Metrics) IncRequestsSuccess() {
	atomic.AddUint64(&m.requestsSuccess, 1)
}

func (m *Metrics) IncRequestsFailed() {
	atomic.AddUint64(&m.requestsFailed, 1)
}

func (m *Metrics) IncRateLimited() {
	atomic.AddUint64(&m.rateLimited, 1)
}

func (m *Metrics) IncNodesSelected() {
	atomic.AddUint64(&m.nodesSelected, 1)
}

func (m *Metrics) IncActiveSessions() {
	atomic.AddUint64(&m.activeSessions, 1)
}

func (m *Metrics) DecActiveSessions() {
	atomic.AddUint64(&m.activeSessions, ^uint64(0))
}

func (m *Metrics) AddLatency(ms int64) {
	atomic.AddUint64(&m.latencyTotalMs, uint64(ms))
	atomic.AddUint64(&m.latencyCount, 1)
}

func (m *Metrics) String() string {
	uptime := time.Since(m.startTime).Seconds()
	total := atomic.LoadUint64(&m.requestsTotal)
	success := atomic.LoadUint64(&m.requestsSuccess)
	failed := atomic.LoadUint64(&m.requestsFailed)
	limited := atomic.LoadUint64(&m.rateLimited)
	nodes := atomic.LoadUint64(&m.nodesSelected)
	sessions := atomic.LoadUint64(&m.activeSessions)
	latencyTotal := atomic.LoadUint64(&m.latencyTotalMs)
	latencyCount := atomic.LoadUint64(&m.latencyCount)

	var avgLatency float64
	if latencyCount > 0 {
		avgLatency = float64(latencyTotal) / float64(latencyCount)
	}

	return fmt.Sprintf(`# HELP proxymesh_requests_total Total requests received
# TYPE proxymesh_requests_total counter
proxymesh_requests_total %d

# HELP proxymesh_requests_success Successful requests
# TYPE proxymesh_requests_success counter
proxymesh_requests_success %d

# HELP proxymesh_requests_failed Failed requests
# TYPE proxymesh_requests_failed counter
proxymesh_requests_failed %d

# HELP proxymesh_rate_limited Rate limited requests
# TYPE proxymesh_rate_limited counter
proxymesh_rate_limited %d

# HELP proxymesh_nodes_selected Nodes selected for proxy
# TYPE proxymesh_nodes_selected counter
proxymesh_nodes_selected %d

# HELP proxymesh_active_sessions Active proxy sessions
# TYPE proxymesh_active_sessions gauge
proxymesh_active_sessions %d

# HELP proxymesh_avg_latency_ms Average request latency in milliseconds
# TYPE proxymesh_avg_latency_ms gauge
proxymesh_avg_latency_ms %.2f

# HELP proxymesh_uptime_seconds Gateway uptime in seconds
# TYPE proxymesh_uptime_seconds gauge
proxymesh_uptime_seconds %.2f
`,
		total, success, failed, limited, nodes, sessions, avgLatency, uptime)
}

func SetupMetricsRoutes(r *gin.Engine) {
	r.GET("/metrics", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(200, metrics.String())
	})
}
