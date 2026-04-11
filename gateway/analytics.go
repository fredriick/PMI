package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type TrafficAnalytics struct {
	redis *redis.Client
	ctx   context.Context
}

func NewTrafficAnalytics(redisClient *redis.Client) *TrafficAnalytics {
	return &TrafficAnalytics{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

type CountryStats struct {
	Country       string  `json:"country"`
	TotalRequests int64   `json:"total_requests"`
	BytesSent     int64   `json:"bytes_sent"`
	BytesReceived int64   `json:"bytes_received"`
	AvgLatency    float64 `json:"avg_latency_ms"`
	ActiveNodes   int     `json:"active_nodes"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
}

type AnalyticsSummary struct {
	TotalRequests    int64             `json:"total_requests"`
	TotalBytesSent   int64             `json:"total_bytes_sent"`
	TotalBytesRecv   int64             `json:"total_bytes_received"`
	ActiveNodes      int               `json:"active_nodes"`
	TopCountries     []CountryStats    `json:"top_countries"`
	RequestsOverTime []TimeSeriesPoint `json:"requests_over_time"`
}

func (ta *TrafficAnalytics) GetSummary(start, end time.Time) (*AnalyticsSummary, error) {
	summary := &AnalyticsSummary{
		TopCountries:     make([]CountryStats, 0),
		RequestsOverTime: make([]TimeSeriesPoint, 0),
	}

	totalReqs, _ := ta.redis.Get(ta.ctx, "analytics:total_requests").Int64()
	summary.TotalRequests = totalReqs

	summary.TotalBytesSent, _ = ta.redis.Get(ta.ctx, "analytics:total_bytes_sent").Int64()
	summary.TotalBytesRecv, _ = ta.redis.Get(ta.ctx, "analytics:total_bytes_recv").Int64()

	nodeKeys, _ := ta.redis.Keys(ta.ctx, "node_meta:*").Result()
	summary.ActiveNodes = len(nodeKeys)

	countryKeys, _ := ta.redis.Keys(ta.ctx, "analytics:country:*").Result()
	for _, key := range countryKeys {
		country := key[len("analytics:country:"):]
		stats, _ := ta.getCountryStats(country)
		if stats.TotalRequests > 0 {
			summary.TopCountries = append(summary.TopCountries, *stats)
		}
	}

	summary.RequestsOverTime = ta.getTimeSeries("analytics:requests:", start, end)

	return summary, nil
}

func (ta *TrafficAnalytics) getCountryStats(country string) (*CountryStats, error) {
	key := fmt.Sprintf("analytics:country:%s", country)
	data, err := ta.redis.HGetAll(ta.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	stats := &CountryStats{Country: country}
	if v, ok := data["requests"]; ok {
		fmt.Sscanf(v, "%d", &stats.TotalRequests)
	}
	if v, ok := data["bytes_sent"]; ok {
		fmt.Sscanf(v, "%d", &stats.BytesSent)
	}
	if v, ok := data["bytes_recv"]; ok {
		fmt.Sscanf(v, "%d", &stats.BytesReceived)
	}
	if v, ok := data["avg_latency"]; ok {
		fmt.Sscanf(v, "%f", &stats.AvgLatency)
	}

	activeKey := fmt.Sprintf("nodes:%s", country)
	active, _ := ta.redis.ZCard(ta.ctx, activeKey).Result()
	stats.ActiveNodes = int(active)

	return stats, nil
}

func (ta *TrafficAnalytics) getTimeSeries(prefix string, start, end time.Time) []TimeSeriesPoint {
	var points []TimeSeriesPoint

	iter := ta.redis.Scan(ta.ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ta.ctx) {
		key := iter.Val()
		data, _ := ta.redis.HGetAll(ta.ctx, key).Result()

		if v, ok := data["timestamp"]; ok {
			var ts int64
			fmt.Sscanf(v, "%d", &ts)
			point := TimeSeriesPoint{
				Timestamp: time.Unix(ts, 0),
			}
			if val, ok := data["value"]; ok {
				fmt.Sscanf(val, "%d", &point.Value)
			}
			if point.Timestamp.After(start) && point.Timestamp.Before(end) {
				points = append(points, point)
			}
		}
	}

	return points
}

func (ta *TrafficAnalytics) RecordRequest(country string, bytesSent, bytesReceived int64, latencyMs int64) {
	ta.redis.Incr(ta.ctx, "analytics:total_requests")
	ta.redis.IncrBy(ta.ctx, "analytics:total_bytes_sent", bytesSent)
	ta.redis.IncrBy(ta.ctx, "analytics:total_bytes_recv", bytesReceived)

	countryKey := fmt.Sprintf("analytics:country:%s", country)
	ta.redis.HIncrBy(ta.ctx, countryKey, "requests", 1)
	ta.redis.HIncrBy(ta.ctx, countryKey, "bytes_sent", bytesSent)
	ta.redis.HIncrBy(ta.ctx, countryKey, "bytes_recv", bytesReceived)

	today := time.Now().Format("2006-01-02")
	tsKey := fmt.Sprintf("analytics:requests:%s", today)
	ts := time.Now().Unix()
	ta.redis.HSet(ta.ctx, tsKey, "timestamp", fmt.Sprintf("%d", ts))
	ta.redis.HIncrBy(ta.ctx, tsKey, "value", 1)
}

func (ta *TrafficAnalytics) RegisterRoutes(r *gin.Engine) {
	r.GET("/v1/analytics/summary", ta.serveSummary)
	r.GET("/v1/analytics/country/:country", ta.serveCountryStats)
	r.GET("/v1/analytics/node/:nodeId", ta.serveNodeStats)
}

func (ta *TrafficAnalytics) serveSummary(c *gin.Context) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	if s := c.Query("start"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			start = t
		}
	}
	if e := c.Query("end"); e != "" {
		if t, err := time.Parse(time.RFC3339, e); err == nil {
			end = t
		}
	}

	summary, err := ta.GetSummary(start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (ta *TrafficAnalytics) serveCountryStats(c *gin.Context) {
	country := c.Param("country")
	if country == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country required"})
		return
	}

	stats, err := ta.getCountryStats(country)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no data for country"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (ta *TrafficAnalytics) serveNodeStats(c *gin.Context) {
	nodeID := c.Param("nodeId")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
		return
	}

	bandwidth, err := ta.redis.Get(ta.ctx, fmt.Sprintf("bandwidth:%s", nodeID)).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no data for node"})
		return
	}

	var data map[string]int64
	json.Unmarshal([]byte(bandwidth), &data)

	c.JSON(http.StatusOK, gin.H{
		"node_id":        nodeID,
		"bytes_sent":     data["bytes_sent"],
		"bytes_received": data["bytes_received"],
	})
}
