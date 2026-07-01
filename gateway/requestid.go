package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	DefaultRequestIDPrefix = "req"
	DefaultRequestIDFormat = "unix"

	RequestIDFormatUnix      = "unix"
	RequestIDFormatTimestamp = "timestamp"
	RequestIDFormatUUID      = "uuid"
)

type RequestIDConfig struct {
	mu     sync.RWMutex
	prefix string
	format string
}

type RequestIDGenerator struct {
	config   *RequestIDConfig
	mu       sync.Mutex
	counters map[string]uint64
	redis    *redis.Client
	ctx      context.Context
}

func NewRequestIDConfig(prefix, format string) *RequestIDConfig {
	ric := &RequestIDConfig{
		prefix: DefaultRequestIDPrefix,
		format: DefaultRequestIDFormat,
	}
	ric.SetPrefix(prefix)
	ric.SetFormat(format)
	return ric
}

func NewRequestIDGenerator(prefix, format string, redisClient *redis.Client) *RequestIDGenerator {
	return &RequestIDGenerator{
		config:   NewRequestIDConfig(prefix, format),
		counters: make(map[string]uint64),
		redis:    redisClient,
		ctx:      context.Background(),
	}
}

func normalizeRequestIDPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return DefaultRequestIDPrefix
	}
	return prefix
}

func normalizeRequestIDFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case RequestIDFormatUnix, RequestIDFormatTimestamp, RequestIDFormatUUID:
		return strings.ToLower(strings.TrimSpace(format))
	default:
		return ""
	}
}

func (ric *RequestIDConfig) SetPrefix(prefix string) {
	ric.mu.Lock()
	defer ric.mu.Unlock()
	ric.prefix = normalizeRequestIDPrefix(prefix)
}

func (ric *RequestIDConfig) SetFormat(format string) bool {
	normalized := normalizeRequestIDFormat(format)
	if normalized == "" {
		return false
	}

	ric.mu.Lock()
	defer ric.mu.Unlock()
	ric.format = normalized
	return true
}

func (ric *RequestIDConfig) Prefix() string {
	ric.mu.RLock()
	defer ric.mu.RUnlock()
	return ric.prefix
}

func (ric *RequestIDConfig) FormatName() string {
	ric.mu.RLock()
	defer ric.mu.RUnlock()
	return ric.format
}

func (ric *RequestIDConfig) Snapshot() map[string]string {
	ric.mu.RLock()
	defer ric.mu.RUnlock()
	return map[string]string{
		"prefix": ric.prefix,
		"format": ric.format,
	}
}

func (ric *RequestIDConfig) Format(value string) string {
	ric.mu.RLock()
	format := ric.format
	prefix := ric.prefix
	ric.mu.RUnlock()

	switch format {
	case RequestIDFormatUUID:
		return fmt.Sprintf("%s-%s", prefix, value)
	case RequestIDFormatTimestamp:
		return fmt.Sprintf("%s-%s", prefix, value)
	default:
		return fmt.Sprintf("%s-%s", prefix, value)
	}
}

func (rig *RequestIDGenerator) SetPrefix(prefix string) {
	rig.config.SetPrefix(prefix)
}

func (rig *RequestIDGenerator) SetFormat(format string) bool {
	return rig.config.SetFormat(format)
}

func (rig *RequestIDGenerator) Config() map[string]string {
	return rig.config.Snapshot()
}

func (rig *RequestIDGenerator) Generate() string {
	rig.mu.Lock()
	rig.counters["global"]++
	counter := rig.counters["global"]
	rig.mu.Unlock()

	value := fmt.Sprintf("%d-%d", time.Now().UnixNano(), counter)
	format := rig.config.FormatName()
	if format == RequestIDFormatTimestamp {
		value = fmt.Sprintf("%d-%d", time.Now().Unix(), counter)
	}
	if format == RequestIDFormatUUID {
		value = uuid.New().String()
	}

	requestID := rig.config.Format(value)

	if rig.redis != nil {
		rig.redis.HIncrBy(rig.ctx, "request_id_counter", "global", 1)
	}

	return requestID
}

func (rig *RequestIDGenerator) GenerateUUID() string {
	return rig.config.Format(uuid.New().String())
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

func (rig *RequestIDGenerator) RegisterRoutes(r gin.IRoutes) {
	r.POST("/request-id", rig.setConfigHandler)
	r.GET("/request-id", rig.getConfigHandler)
}

func (rig *RequestIDGenerator) setConfigHandler(c *gin.Context) {
	var req struct {
		Prefix string `json:"prefix"`
		Format string `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Prefix != "" {
		rig.SetPrefix(req.Prefix)
	}
	if req.Format != "" && !rig.SetFormat(req.Format) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id format"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"prefix": rig.config.Prefix(),
		"format": rig.config.FormatName(),
	})
}

func (rig *RequestIDGenerator) getConfigHandler(c *gin.Context) {
	c.JSON(http.StatusOK, rig.Config())
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
