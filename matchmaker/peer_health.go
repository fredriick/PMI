package matchmaker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type PeerHealthScore struct {
	mu     sync.RWMutex
	scores map[string]*PeerHealth
	redis  *redis.Client
	ctx    context.Context
}

type PeerHealth struct {
	NodeID           string    `json:"node_id"`
	ConnectionScore  float64   `json:"connection_score"`
	LatencyScore     float64   `json:"latency_score"`
	BandwidthScore   float64   `json:"bandwidth_score"`
	ReliabilityScore float64   `json:"reliability_score"`
	OverallScore     float64   `json:"overall_score"`
	LastChecked      time.Time `json:"last_checked"`
}

// HealthScorePoint is a single composite-score sample used to render decay trends.
type HealthScorePoint struct {
	OverallScore float64   `json:"overall_score"`
	Timestamp    time.Time `json:"timestamp"`
}

func NewPeerHealthScore(redisClient *redis.Client) *PeerHealthScore {
	return &PeerHealthScore{
		scores: make(map[string]*PeerHealth),
		redis:  redisClient,
		ctx:    context.Background(),
	}
}

func (phs *PeerHealthScore) UpdateScore(nodeID string, latencyMs int64, bandwidthMbps float64, reliability float64) {
	latencyScore := phs.calculateLatencyScore(latencyMs)
	bandwidthScore := phs.calculateBandwidthScore(bandwidthMbps)
	reliabilityScore := reliability * 100
	connectionScore := (latencyScore + bandwidthScore + reliabilityScore) / 3

	overall := connectionScore*0.3 + latencyScore*0.25 + bandwidthScore*0.2 + reliabilityScore*0.25

	phs.mu.Lock()
	phs.scores[nodeID] = &PeerHealth{
		NodeID:           nodeID,
		ConnectionScore:  connectionScore,
		LatencyScore:     latencyScore,
		BandwidthScore:   bandwidthScore,
		ReliabilityScore: reliabilityScore,
		OverallScore:     overall,
		LastChecked:      time.Now(),
	}
	phs.mu.Unlock()

	if phs.redis != nil {
		phs.redis.HSet(phs.ctx, fmt.Sprintf("peer:health:%s", nodeID), map[string]interface{}{
			"connection_score":  connectionScore,
			"latency_score":     latencyScore,
			"bandwidth_score":   bandwidthScore,
			"reliability_score": reliabilityScore,
			"overall_score":     overall,
		})
		phs.RecordHistory(nodeID, overall)
	}
}

const healthScoreHistoryMax = 100

func (phs *PeerHealthScore) RecordHistory(nodeID string, overall float64) {
	if phs.redis == nil {
		return
	}
	point := HealthScorePoint{OverallScore: overall, Timestamp: time.Now()}
	data, err := json.Marshal(point)
	if err != nil {
		return
	}
	key := fmt.Sprintf("healthscore:history:%s", nodeID)
	phs.redis.RPush(phs.ctx, key, string(data))
	phs.redis.LTrim(phs.ctx, key, int64(-healthScoreHistoryMax), -1)
}

func (phs *PeerHealthScore) GetHistory(nodeID string) []HealthScorePoint {
	if phs.redis == nil {
		return nil
	}
	key := fmt.Sprintf("healthscore:history:%s", nodeID)
	data, err := phs.redis.LRange(phs.ctx, key, int64(-healthScoreHistoryMax), -1).Result()
	if err != nil {
		return nil
	}
	out := make([]HealthScorePoint, 0, len(data))
	for _, d := range data {
		var p HealthScorePoint
		if json.Unmarshal([]byte(d), &p) == nil {
			out = append(out, p)
		}
	}
	return out
}

func (phs *PeerHealthScore) calculateLatencyScore(latencyMs int64) float64 {
	if latencyMs <= 30 {
		return 100
	}
	if latencyMs >= 500 {
		return 0
	}
	return 100 - ((float64(latencyMs) - 30) * 100 / 470)
}

func (phs *PeerHealthScore) calculateBandwidthScore(mbps float64) float64 {
	if mbps >= 100 {
		return 100
	}
	if mbps <= 1 {
		return 0
	}
	return (mbps / 100) * 100
}

func (phs *PeerHealthScore) GetScore(nodeID string) *PeerHealth {
	phs.mu.RLock()
	defer phs.mu.RUnlock()
	return phs.scores[nodeID]
}

func (phs *PeerHealthScore) GetTopPeers(count int) []*PeerHealth {
	phs.mu.RLock()
	defer phs.mu.RUnlock()

	peers := make([]*PeerHealth, 0, len(phs.scores))
	for _, p := range phs.scores {
		peers = append(peers, p)
	}

	for i := 0; i < len(peers)-1; i++ {
		for j := i + 1; j < len(peers); j++ {
			if peers[j].OverallScore > peers[i].OverallScore {
				peers[i], peers[j] = peers[j], peers[i]
			}
		}
	}

	if count < len(peers) {
		return peers[:count]
	}
	return peers
}
