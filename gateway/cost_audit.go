package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type CostEstimator struct {
	mu          sync.RWMutex
	baseCosts   map[string]CostConfig
	clientCosts map[string]float64
	redis       *redis.Client
	ctx         context.Context
}

type CostConfig struct {
	BaseCost   float64 `json:"base_cost"`
	PerGBRecv  float64 `json:"per_gb_recv"`
	PerGBSent  float64 `json:"per_gb_sent"`
	PerRequest float64 `json:"per_request"`
	PerMinute  float64 `json:"per_minute"`
}

func NewCostEstimator(redisClient *redis.Client) *CostEstimator {
	ce := &CostEstimator{
		baseCosts:   make(map[string]CostConfig),
		clientCosts: make(map[string]float64),
		redis:       redisClient,
		ctx:         context.Background(),
	}

	ce.baseCosts["default"] = CostConfig{
		BaseCost:   0.0,
		PerGBRecv:  0.15,
		PerGBSent:  0.30,
		PerRequest: 0.001,
		PerMinute:  0.01,
	}

	return ce
}

func (ce *CostEstimator) EstimateRequest(clientID string, bytesReceived, bytesSent int64, durationSeconds int) float64 {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	cost := ce.baseCosts["default"].BaseCost

	cost += ce.baseCosts["default"].PerRequest

	if bytesSent > 0 {
		cost += float64(bytesSent) / (1024 * 1024 * 1024) * ce.baseCosts["default"].PerGBSent
	}

	if bytesReceived > 0 {
		cost += float64(bytesReceived) / (1024 * 1024 * 1024) * ce.baseCosts["default"].PerGBRecv
	}

	if durationSeconds > 0 {
		cost += float64(durationSeconds) / 60.0 * ce.baseCosts["default"].PerMinute
	}

	if clientCost, exists := ce.clientCosts[clientID]; exists {
		cost *= clientCost
	}

	return cost
}

func (ce *CostEstimator) EstimateMonthly(clientID string, dailyRequests int, avgBytesSent, avgBytesReceived int64, avgDurationSeconds int) float64 {
	monthlyCost := 0.0
	for i := 0; i < 30; i++ {
		monthlyCost += ce.EstimateRequest(clientID, avgBytesReceived, avgBytesSent, avgDurationSeconds) * float64(dailyRequests)
	}
	return monthlyCost
}

func (ce *CostEstimator) SetClientCost(clientID string, multiplier float64) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.clientCosts[clientID] = multiplier

	if ce.redis != nil {
		ce.redis.Set(ce.ctx, fmt.Sprintf("cost:%s", clientID), multiplier, 0)
	}
}

func (ce *CostEstimator) GetClientCost(clientID string) float64 {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	if cost, exists := ce.clientCosts[clientID]; exists {
		return cost
	}
	return 1.0
}

func (ce *CostEstimator) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/costs", ce.setCostHandler)
	r.GET("/api/admin/costs", ce.listCostsHandler)
	r.POST("/api/admin/costs/estimate", ce.estimateHandler)
}

func (ce *CostEstimator) setCostHandler(c *gin.Context) {
	var req struct {
		ClientID   string  `json:"client_id"`
		Multiplier float64 `json:"multiplier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ce.SetClientCost(req.ClientID, req.Multiplier)
	c.JSON(200, gin.H{"status": "success"})
}

func (ce *CostEstimator) listCostsHandler(c *gin.Context) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	c.JSON(200, gin.H{"costs": ce.clientCosts})
}

func (ce *CostEstimator) estimateHandler(c *gin.Context) {
	var req struct {
		ClientID           string `json:"client_id"`
		DailyRequests      int    `json:"daily_requests"`
		AvgBytesSent       int64  `json:"avg_bytes_sent"`
		AvgBytesReceived   int64  `json:"avg_bytes_received"`
		AvgDurationSeconds int    `json:"avg_duration_seconds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	monthlyCost := ce.EstimateMonthly(req.ClientID, req.DailyRequests, req.AvgBytesSent, req.AvgBytesReceived, req.AvgDurationSeconds)

	c.JSON(200, gin.H{
		"client_id":    req.ClientID,
		"monthly_cost": monthlyCost,
	})
}

type AuditRetention struct {
	mu        sync.RWMutex
	retention map[string]RetentionPolicy
	redis     *redis.Client
	ctx       context.Context
}

type RetentionPolicy struct {
	Action        string `json:"action"`
	RetentionDays int    `json:"retention_days"`
	MaxEntries    int    `json:"max_entries"`
}

func NewAuditRetention(redisClient *redis.Client) *AuditRetention {
	ar := &AuditRetention{
		retention: make(map[string]RetentionPolicy),
		redis:     redisClient,
		ctx:       context.Background(),
	}

	ar.retention["default"] = RetentionPolicy{
		RetentionDays: 90,
		MaxEntries:    10000,
	}

	go ar.cleanupLoop()

	return ar
}

func (ar *AuditRetention) SetRetentionPolicy(action string, retentionDays, maxEntries int) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	ar.retention[action] = RetentionPolicy{
		Action:        action,
		RetentionDays: retentionDays,
		MaxEntries:    maxEntries,
	}
}

func (ar *AuditRetention) GetRetentionPolicy(action string) RetentionPolicy {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	if policy, exists := ar.retention[action]; exists {
		return policy
	}
	return ar.retention["default"]
}

func (ar *AuditRetention) Cleanup() error {
	if ar.redis == nil {
		return nil
	}

	keys, err := ar.redis.Keys(ar.ctx, "audit:*").Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		action := key[6:]
		policy := ar.GetRetentionPolicy(action)

		ar.redis.LTrim(ar.ctx, key, int64(-policy.MaxEntries), -1)

		cutoff := time.Now().AddDate(0, 0, -policy.RetentionDays)
		ar.redis.ZRemRangeByScore(ar.ctx, key, "0", fmt.Sprintf("%d", cutoff.Unix()))
	}

	return nil
}

func (ar *AuditRetention) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ar.Cleanup()
	}
}

func (ar *AuditRetention) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/audit/retention", ar.setRetentionHandler)
	r.GET("/api/admin/audit/retention", ar.getRetentionHandler)
	r.POST("/api/admin/audit/cleanup", ar.runCleanupHandler)
}

func (ar *AuditRetention) setRetentionHandler(c *gin.Context) {
	var req struct {
		Action        string `json:"action"`
		RetentionDays int    `json:"retention_days"`
		MaxEntries    int    `json:"max_entries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ar.SetRetentionPolicy(req.Action, req.RetentionDays, req.MaxEntries)
	c.JSON(200, gin.H{"status": "success"})
}

func (ar *AuditRetention) getRetentionHandler(c *gin.Context) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	policies := make(map[string]RetentionPolicy)
	for action, policy := range ar.retention {
		policies[action] = policy
	}

	c.JSON(200, gin.H{"policies": policies})
}

func (ar *AuditRetention) runCleanupHandler(c *gin.Context) {
	if err := ar.Cleanup(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "cleanup complete"})
}
