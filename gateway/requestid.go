package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RequestIDGenerator struct {
	prefix   string
	mu       sync.RWMutex
	counters map[string]uint64
	redis    *redis.Client
	ctx      context.Context
}

func NewRequestIDGenerator(prefix string, redisClient *redis.Client) *RequestIDGenerator {
	return &RequestIDGenerator{
		prefix:   prefix,
		counters: make(map[string]uint64),
		redis:    redisClient,
		ctx:      context.Background(),
	}
}

func (rig *RequestIDGenerator) Generate() string {
	rig.mu.Lock()
	defer rig.mu.Unlock()

	rig.counters["global"]++
	counter := rig.counters["global"]

	timestamp := time.Now().UnixNano()
	requestID := fmt.Sprintf("%s-%d-%d", rig.prefix, timestamp, counter)

	if rig.redis != nil {
		rig.redis.HIncrBy(rig.ctx, "request_id_counter", "global", 1)
	}

	return requestID
}

func (rig *RequestIDGenerator) GenerateUUID() string {
	return uuid.New().String()
}

func (rig *RequestIDGenerator) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = rig.Generate()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

func (rig *RequestIDGenerator) GetTraceID(c *gin.Context) string {
	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = rig.GenerateUUID()
	}
	return traceID
}

type AdminMetrics struct {
	mu        sync.RWMutex
	apiCalls  map[string]int64
	errors    map[string]int64
	latencies map[string][]int64
	redis     *redis.Client
	ctx       context.Context
}

func NewAdminMetrics(redisClient *redis.Client) *AdminMetrics {
	return &AdminMetrics{
		apiCalls:  make(map[string]int64),
		errors:    make(map[string]int64),
		latencies: make(map[string][]int64),
		redis:     redisClient,
		ctx:       context.Background(),
	}
}

func (am *AdminMetrics) RecordCall(endpoint string, latencyMs int64, isError bool) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.apiCalls[endpoint]++

	if isError {
		am.errors[endpoint]++
	}

	am.latencies[endpoint] = append(am.latencies[endpoint], latencyMs)
	if len(am.latencies[endpoint]) > 1000 {
		am.latencies[endpoint] = am.latencies[endpoint][1:]
	}
}

func (am *AdminMetrics) GetMetrics() map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	metrics := map[string]interface{}{
		"api_calls": make(map[string]int64),
		"errors":    make(map[string]int64),
		"latencies": make(map[string]map[string]float64),
	}

	for endpoint, count := range am.apiCalls {
		metrics["api_calls"].(map[string]int64)[endpoint] = count
	}

	for endpoint, count := range am.errors {
		metrics["errors"].(map[string]int64)[endpoint] = count
	}

	for endpoint, latencies := range am.latencies {
		var sum int64
		for _, l := range latencies {
			sum += l
		}
		avg := float64(sum) / float64(len(latencies))

		metrics["latencies"].(map[string]map[string]float64)[endpoint] = map[string]float64{
			"avg_ms": avg,
			"count":  float64(len(latencies)),
		}
	}

	return metrics
}

func (am *AdminMetrics) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/admin/metrics", am.metricsHandler)
}

func (am *AdminMetrics) metricsHandler(c *gin.Context) {
	c.JSON(200, am.GetMetrics())
}

func (am *AdminMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		endpoint := c.Request.URL.Path

		c.Next()

		latencyMs := time.Since(start).Milliseconds()
		isError := c.Writer.Status() >= 400

		am.RecordCall(endpoint, latencyMs, isError)
	}
}

type NodeTimeouts struct {
	mu         sync.RWMutex
	timeouts   map[string]NodeTimeoutConfig
	defaultCfg NodeTimeoutConfig
}

type NodeTimeoutConfig struct {
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

func NewNodeTimeouts(defaultDial, defaultRead, defaultWrite, defaultIdle time.Duration) *NodeTimeouts {
	return &NodeTimeouts{
		timeouts: make(map[string]NodeTimeoutConfig),
		defaultCfg: NodeTimeoutConfig{
			DialTimeout:  defaultDial,
			ReadTimeout:  defaultRead,
			WriteTimeout: defaultWrite,
			IdleTimeout:  defaultIdle,
		},
	}
}

func (nt *NodeTimeouts) SetNodeTimeout(nodeID string, config NodeTimeoutConfig) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.timeouts[nodeID] = config
}

func (nt *NodeTimeouts) GetNodeTimeout(nodeID string) NodeTimeoutConfig {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	if config, exists := nt.timeouts[nodeID]; exists {
		return config
	}
	return nt.defaultCfg
}

func (nt *NodeTimeouts) SetDefaultTimeout(config NodeTimeoutConfig) {
	nt.mu.Lock()
	defer nt.mu.Unlock()
	nt.defaultCfg = config
}

func (nt *NodeTimeouts) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/timeouts", nt.setTimeoutHandler)
	r.GET("/api/admin/timeouts", nt.listTimeoutsHandler)
}

func (nt *NodeTimeouts) setTimeoutHandler(c *gin.Context) {
	var req struct {
		NodeID string            `json:"node_id"`
		Config NodeTimeoutConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	nt.SetNodeTimeout(req.NodeID, req.Config)
	c.JSON(200, gin.H{"status": "success"})
}

func (nt *NodeTimeouts) listTimeoutsHandler(c *gin.Context) {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	timeouts := make(map[string]NodeTimeoutConfig)
	for nodeID, config := range nt.timeouts {
		timeouts[nodeID] = config
	}

	c.JSON(200, gin.H{
		"timeouts": timeouts,
		"default":  nt.defaultCfg,
	})
}
