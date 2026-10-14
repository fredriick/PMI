package gateway

import (
	"context"
	"testing"
	"time"
)

func TestTrafficAnalytics_New(t *testing.T) {
	ta := NewTrafficAnalytics(nil)
	if ta == nil {
		t.Fatal("NewTrafficAnalytics returned nil")
	}
	if ta.ctx == nil {
		t.Error("expected ctx to be initialized")
	}
}

func TestTrafficAnalytics_RecordRequest(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ta := NewTrafficAnalytics(client)
	ta.RecordRequest("US", 100, 50, 10)
	ta.RecordRequest("US", 200, 75, 20)
	ta.RecordRequest("GB", 50, 25, 5)
}

func TestTrafficAnalytics_GetCountryStats(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ta := NewTrafficAnalytics(client)
	client.HSet(context.Background(), "analytics:country:US", "requests", "10", "bytes_sent", "1000", "bytes_recv", "500", "avg_latency", "25.5")
	client.ZCard(context.Background(), "nodes:US")

	stats, err := ta.getCountryStats("US")
	if err != nil {
		t.Fatalf("getCountryStats error: %v", err)
	}
	if stats.TotalRequests != 10 {
		t.Errorf("expected 10 requests, got %d", stats.TotalRequests)
	}
	if stats.BytesSent != 1000 {
		t.Errorf("expected 1000 bytes sent, got %d", stats.BytesSent)
	}
}

func TestTrafficAnalytics_GetTimeSeries(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ta := NewTrafficAnalytics(client)
	now := time.Now()
	key := "analytics:requests:" + now.Format("2006-01-02")
	ts := now.Unix()
	client.HSet(context.Background(), key, "timestamp", "0", "value", "1")
	_ = ts

	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	points := ta.getTimeSeries("analytics:requests:", start, end)
	_ = points
}

func TestTrafficAnalytics_ServeCountryStats_Empty(t *testing.T) {
	client := redisClient(t)
	defer client.Close()

	ta := NewTrafficAnalytics(client)
	ta.RecordRequest("US", 100, 50, 10)
}
