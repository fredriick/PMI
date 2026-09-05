package matchmaker

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
)

func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	_ = rdb.FlushDB(context.Background()).Err()
	return rdb
}

func TestNewPeerHealthScore_NoRedis(t *testing.T) {
	phs := NewPeerHealthScore(nil)
	if phs == nil {
		t.Fatal("expected non-nil PeerHealthScore")
	}
	if phs.scores == nil {
		t.Error("expected non-nil scores map")
	}
	if phs.GetScore("nonexistent") != nil {
		t.Error("expected nil for nonexistent node")
	}
	if phs.GetHistory("nonexistent") != nil {
		t.Error("expected nil history for nonexistent node without Redis")
	}
}

func TestUpdateScore_Scoring(t *testing.T) {
	phs := NewPeerHealthScore(nil)

	phs.UpdateScore("node1", 20, 200, 0.95)

	score := phs.GetScore("node1")
	if score == nil {
		t.Fatal("expected non-nil score for node1")
	}

	if score.ConnectionScore <= 0 {
		t.Error("expected positive connection score")
	}
	if score.OverallScore <= 0 || score.OverallScore > 100 {
		t.Errorf("expected overall score in (0, 100], got %.2f", score.OverallScore)
	}
	if score.NodeID != "node1" {
		t.Errorf("expected node_id 'node1', got '%s'", score.NodeID)
	}
}

func TestCalculateLatencyScore(t *testing.T) {
	phs := NewPeerHealthScore(nil)

	tests := []struct {
		latencyMs int64
		wantMin   float64
		wantMax   float64
	}{
		{0, 100, 100},
		{10, 100, 100},
		{30, 100, 100},
		{100, 0, 100},
		{500, 0, 0},
		{1000, 0, 0},
	}

	for _, tt := range tests {
		score := phs.calculateLatencyScore(tt.latencyMs)
		if score < tt.wantMin || score > tt.wantMax {
			t.Errorf("calculateLatencyScore(%d) = %.2f, expected [%f, %f]", tt.latencyMs, score, tt.wantMin, tt.wantMax)
		}
	}
}

func TestCalculateBandwidthScore(t *testing.T) {
	phs := NewPeerHealthScore(nil)

	tests := []struct {
		mbps  float64
		want  float64
	}{
		{0, 0},
		{0.5, 0},
		{1, 0},
		{50, 50},
		{100, 100},
		{200, 100},
	}

	for _, tt := range tests {
		score := phs.calculateBandwidthScore(tt.mbps)
		if score != tt.want {
			t.Errorf("calculateBandwidthScore(%.1f) = %.2f, expected %.2f", tt.mbps, score, tt.want)
		}
	}
}

func TestGetTopPeers_NoScores(t *testing.T) {
	phs := NewPeerHealthScore(nil)

	top := phs.GetTopPeers(5)
	if len(top) != 0 {
		t.Errorf("expected empty list, got %d", len(top))
	}
}

func TestGetTopPeers_Sorted(t *testing.T) {
	phs := NewPeerHealthScore(nil)

	phs.UpdateScore("node1", 30, 100, 0.99)
	phs.UpdateScore("node2", 200, 50, 0.5)
	phs.UpdateScore("node3", 30, 200, 1.0)

	top := phs.GetTopPeers(2)
	if len(top) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(top))
	}

	if top[0].NodeID != "node3" {
		t.Errorf("expected node3 first (highest score), got '%s'", top[0].NodeID)
	}
}

func TestPeerHealthScore_WithRedis(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	phs := NewPeerHealthScore(rdb)

	phs.UpdateScore("node1", 30, 100, 0.99)

	score := phs.GetScore("node1")
	if score == nil {
		t.Fatal("expected non-nil score")
	}

	history := phs.GetHistory("node1")
	if len(history) == 0 {
		t.Error("expected history to be recorded in Redis")
	}

	if history[0].OverallScore != score.OverallScore {
		t.Error("expected history point to match current score")
	}
}

func TestPeerHealthScore_HistoryCapped(t *testing.T) {
	rdb := redisClient(t)
	defer rdb.Close()

	phs := NewPeerHealthScore(rdb)

	for i := 0; i < 150; i++ {
		phs.RecordHistory("node1", float64(i))
	}

	history := phs.GetHistory("node1")
	if len(history) > healthScoreHistoryMax {
		t.Errorf("expected max %d history entries, got %d", healthScoreHistoryMax, len(history))
	}
	if len(history) != healthScoreHistoryMax {
		t.Errorf("expected exactly %d history entries, got %d", healthScoreHistoryMax, len(history))
	}
}
