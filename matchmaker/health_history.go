package matchmaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type HealthHistory struct {
	mu         sync.RWMutex
	history    map[string][]HealthStatus
	redis      *redis.Client
	ctx        context.Context
	maxEntries int
}

type HealthStatus struct {
	NodeID    string    `json:"node_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	LatencyMs int64     `json:"latency_ms"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHealthHistory(redisClient *redis.Client, maxEntries int) *HealthHistory {
	if maxEntries <= 0 {
		maxEntries = 100
	}

	return &HealthHistory{
		history:    make(map[string][]HealthStatus),
		redis:      redisClient,
		ctx:        context.Background(),
		maxEntries: maxEntries,
	}
}

func (hh *HealthHistory) RecordHealth(nodeID, status, message string, latencyMs int64) {
	entry := HealthStatus{
		NodeID:    nodeID,
		Status:    status,
		Message:   message,
		LatencyMs: latencyMs,
		Timestamp: time.Now(),
	}

	hh.mu.Lock()
	hh.history[nodeID] = append(hh.history[nodeID], entry)
	if len(hh.history[nodeID]) > hh.maxEntries {
		hh.history[nodeID] = hh.history[nodeID][1:]
	}
	hh.mu.Unlock()

	if hh.redis != nil {
		data := fmt.Sprintf("%s|%s|%s|%d|%d", status, message, entry.Timestamp.Format(time.RFC3339), latencyMs, 0)
		key := fmt.Sprintf("health:%s", nodeID)
		hh.redis.RPush(hh.ctx, key, data)
		hh.redis.LTrim(hh.ctx, key, int64(-hh.maxEntries), -1)
	}
}

func (hh *HealthHistory) GetHistory(nodeID string, limit int) []HealthStatus {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	if limit <= 0 || limit > hh.maxEntries {
		limit = hh.maxEntries
	}

	history := hh.history[nodeID]
	if len(history) == 0 && hh.redis != nil {
		key := fmt.Sprintf("health:%s", nodeID)
		data, err := hh.redis.LRange(hh.ctx, key, 0, int64(limit-1)).Result()
		if err == nil {
			for _, d := range data {
				var status HealthStatus
				status.NodeID = nodeID
				fmt.Sscanf(d, "%s|%s|%s|%d|%d", &status.Status, &status.Message, &status.LatencyMs)
				history = append(history, status)
			}
		}
	}

	if limit < len(history) {
		return history[:limit]
	}
	return history
}

func (hh *HealthHistory) GetStatusTimeline(nodeID string, duration time.Duration) []HealthStatus {
	hh.mu.RLock()
	defer hh.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var result []HealthStatus

	for _, status := range hh.history[nodeID] {
		if status.Timestamp.After(cutoff) {
			result = append(result, status)
		}
	}

	return result
}

func (hh *HealthHistory) GetUptimePercentage(nodeID string, duration time.Duration) float64 {
	timeline := hh.GetStatusTimeline(nodeID, duration)
	if len(timeline) == 0 {
		return 0.0
	}

	healthy := 0
	for _, status := range timeline {
		if status.Status == "healthy" {
			healthy++
		}
	}

	return float64(healthy) / float64(len(timeline)) * 100
}

func (hh *HealthHistory) GetAverageLatency(nodeID string, duration time.Duration) int64 {
	timeline := hh.GetStatusTimeline(nodeID, duration)
	if len(timeline) == 0 {
		return 0
	}

	var sum int64
	for _, status := range timeline {
		sum += status.LatencyMs
	}

	return sum / int64(len(timeline))
}

func (hh *HealthHistory) RegisterRoutes(r *gin.Engine, mm *Matchmaker) {
	r.GET("/api/admin/health/:nodeId/history", func(c *gin.Context) {
		nodeID := c.Param("nodeId")
		if nodeID == "" {
			c.JSON(400, gin.H{"error": "node_id required"})
			return
		}

		limit := 10
		if l := c.Query("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}

		history := hh.GetHistory(nodeID, limit)
		c.JSON(200, gin.H{
			"node_id": nodeID,
			"history": history,
			"count":   len(history),
		})
	})

	r.GET("/api/admin/health/:nodeId/timeline", func(c *gin.Context) {
		nodeID := c.Param("nodeId")
		if nodeID == "" {
			c.JSON(400, gin.H{"error": "node_id required"})
			return
		}

		duration := 24 * time.Hour
		if d := c.Query("duration"); d != "" {
			fmt.Sscanf(d, "%d", &duration)
		}

		timeline := hh.GetStatusTimeline(nodeID, duration)
		uptime := hh.GetUptimePercentage(nodeID, duration)
		avgLatency := hh.GetAverageLatency(nodeID, duration)

		c.JSON(200, gin.H{
			"node_id":     nodeID,
			"timeline":    timeline,
			"uptime_pct":  uptime,
			"avg_latency": avgLatency,
		})
	})
}
