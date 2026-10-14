package gateway

import (
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMetrics_Inc(t *testing.T) {
	m := &Metrics{startTime: time.Now()}
	m.IncRequestsTotal()
	m.IncRequestsTotal()
	m.IncRequestsSuccess()
	m.IncRequestsFailed()
	m.IncRateLimited()
	m.IncNodesSelected()
	m.IncActiveSessions()
	m.DecActiveSessions()
	m.AddLatency(10)
	m.AddLatency(20)
}

func TestMetrics_String(t *testing.T) {
	m := &Metrics{startTime: time.Now()}
	m.IncRequestsTotal()
	m.IncRequestsSuccess()
	s := m.String()
	if !strings.Contains(s, "proxymesh_requests_total") {
		t.Error("expected requests_total in metrics output")
	}
	if !strings.Contains(s, "proxymesh_requests_success") {
		t.Error("expected requests_success in metrics output")
	}
}

func TestMetrics_AddLatency(t *testing.T) {
	m := &Metrics{startTime: time.Now()}
	m.AddLatency(50)
	m.AddLatency(100)
	s := m.String()
	if !strings.Contains(s, "proxymesh_avg_latency_ms") {
		t.Error("expected avg_latency_ms in metrics output")
	}
}

func TestMetrics_GlobalInstance(t *testing.T) {
	if metrics == nil {
		t.Fatal("expected global metrics instance")
	}
	if metrics.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
}

func TestSetupMetricsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	SetupMetricsRoutes(router)
}
