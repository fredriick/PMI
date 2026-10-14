package matchmaker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHealthHistory_New(t *testing.T) {
	hh := NewHealthHistory(nil, 50)
	if hh == nil {
		t.Fatal("NewHealthHistory returned nil")
	}
	if hh.maxEntries != 50 {
		t.Errorf("expected maxEntries 50, got %d", hh.maxEntries)
	}
}

func TestHealthHistory_New_DefaultMaxEntries(t *testing.T) {
	hh := NewHealthHistory(nil, 0)
	if hh.maxEntries != 100 {
		t.Errorf("expected default maxEntries 100, got %d", hh.maxEntries)
	}
}

func TestHealthHistory_RecordAndGet(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)
	hh.RecordHealth("node-1", "degraded", "slow", 100)
	hh.RecordHealth("node-2", "healthy", "ok", 5)

	history := hh.GetHistory("node-1", 10)
	if len(history) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(history))
	}
	if history[0].Status != "healthy" {
		t.Errorf("expected 'healthy', got %s", history[0].Status)
	}
	if history[1].Status != "degraded" {
		t.Errorf("expected 'degraded', got %s", history[1].Status)
	}
}

func TestHealthHistory_GetHistory_Empty(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	history := hh.GetHistory("nonexistent", 10)
	if len(history) != 0 {
		t.Errorf("expected 0 entries, got %d", len(history))
	}
}

func TestHealthHistory_GetHistory_Limit(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	for i := 0; i < 5; i++ {
		hh.RecordHealth("node-1", "healthy", "ok", 10)
	}
	history := hh.GetHistory("node-1", 2)
	if len(history) != 2 {
		t.Errorf("expected 2 entries with limit 2, got %d", len(history))
	}
}

func TestHealthHistory_RecordHealth_MaxEntries(t *testing.T) {
	hh := NewHealthHistory(nil, 3)
	hh.RecordHealth("node-1", "healthy", "1", 10)
	hh.RecordHealth("node-1", "healthy", "2", 10)
	hh.RecordHealth("node-1", "healthy", "3", 10)
	hh.RecordHealth("node-1", "healthy", "4", 10)

	history := hh.GetHistory("node-1", 10)
	if len(history) != 3 {
		t.Errorf("expected max 3 entries, got %d", len(history))
	}
	if history[0].Message != "2" {
		t.Errorf("expected oldest entry '2' to be evicted, got %s", history[0].Message)
	}
}

func TestHealthHistory_GetStatusTimeline(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)
	hh.RecordHealth("node-1", "unhealthy", "down", 100)

	timeline := hh.GetStatusTimeline("node-1", 24*time.Hour)
	if len(timeline) != 2 {
		t.Errorf("expected 2 timeline entries, got %d", len(timeline))
	}
}

func TestHealthHistory_GetUptimePercentage(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)
	hh.RecordHealth("node-1", "unhealthy", "down", 100)

	uptime := hh.GetUptimePercentage("node-1", 24*time.Hour)
	if uptime != 66.66666666666666 {
		t.Errorf("expected 66.67%% uptime, got %f", uptime)
	}
}

func TestHealthHistory_GetUptimePercentage_Empty(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	uptime := hh.GetUptimePercentage("nonexistent", 24*time.Hour)
	if uptime != 0 {
		t.Errorf("expected 0%% uptime for empty, got %f", uptime)
	}
}

func TestHealthHistory_GetAverageLatency(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)
	hh.RecordHealth("node-1", "healthy", "ok", 20)
	hh.RecordHealth("node-1", "healthy", "ok", 30)

	avg := hh.GetAverageLatency("node-1", 24*time.Hour)
	if avg != 20 {
		t.Errorf("expected avg latency 20, got %d", avg)
	}
}

func TestHealthHistory_GetAverageLatency_Empty(t *testing.T) {
	hh := NewHealthHistory(nil, 10)
	avg := hh.GetAverageLatency("nonexistent", 24*time.Hour)
	if avg != 0 {
		t.Errorf("expected 0 avg latency for empty, got %d", avg)
	}
}

func TestHealthHistory_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hh := NewHealthHistory(nil, 10)
	router := gin.New()
	hh.RegisterRoutes(router, nil)
}

func TestHealthHistory_HealthHistoryHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hh := NewHealthHistory(nil, 10)
	hh.RecordHealth("node-1", "healthy", "ok", 10)

	router := gin.New()
	hh.RegisterRoutes(router, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/health/node-1/history?limit=5", nil)

	router.ServeHTTP(w, c.Request)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHealthStatus_Struct(t *testing.T) {
	hs := HealthStatus{
		NodeID:    "node-1",
		Status:    "healthy",
		Message:   "ok",
		LatencyMs: 25,
		Timestamp: time.Now(),
	}
	if hs.NodeID != "node-1" {
		t.Error("unexpected NodeID")
	}
	if hs.Status != "healthy" {
		t.Error("unexpected Status")
	}
	if hs.LatencyMs != 25 {
		t.Error("unexpected LatencyMs")
	}
}
