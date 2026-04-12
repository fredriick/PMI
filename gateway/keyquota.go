package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type KeyExpirationManager struct {
	mu    sync.RWMutex
	keys  map[string]*KeyInfo
	redis *redis.Client
	ctx   context.Context
}

type KeyInfo struct {
	Key           string    `json:"key"`
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	RenewalPeriod int       `json:"renewal_days"`
	LastRenewed   time.Time `json:"last_renewed"`
}

func NewKeyExpirationManager(redisClient *redis.Client) *KeyExpirationManager {
	kem := &KeyExpirationManager{
		keys:  make(map[string]*KeyInfo),
		redis: redisClient,
		ctx:   context.Background(),
	}

	go kem.expirationLoop()

	return kem
}

func (kem *KeyExpirationManager) CreateKey(key, name string, ttlDays int, renewalPeriod int) error {
	info := &KeyInfo{
		Key:           key,
		Name:          name,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().AddDate(0, 0, ttlDays),
		RenewalPeriod: renewalPeriod,
		LastRenewed:   time.Now(),
	}

	kem.mu.Lock()
	kem.keys[key] = info
	kem.mu.Unlock()

	if kem.redis != nil {
		data := fmt.Sprintf("%s|%s|%d|%d", info.Name, info.ExpiresAt.Format(time.RFC3339), info.RenewalPeriod, info.LastRenewed.Unix())
		kem.redis.Set(kem.ctx, fmt.Sprintf("key:expiry:%s", key), data, time.Duration(ttlDays)*24*time.Hour)
	}

	return nil
}

func (kem *KeyExpirationManager) IsExpired(key string) bool {
	kem.mu.RLock()
	defer kem.mu.RUnlock()

	if info, exists := kem.keys[key]; exists {
		return time.Now().After(info.ExpiresAt)
	}

	if kem.redis != nil {
		exists, _ := kem.redis.Exists(kem.ctx, fmt.Sprintf("key:expiry:%s", key)).Result()
		return exists == 0
	}

	return false
}

func (kem *KeyExpirationManager) RenewKey(key string, newTTLDays int) error {
	kem.mu.Lock()
	defer kem.mu.Unlock()

	if info, exists := kem.keys[key]; exists {
		info.ExpiresAt = time.Now().AddDate(0, 0, newTTLDays)
		info.LastRenewed = time.Now()

		if kem.redis != nil {
			kem.redis.Set(kem.ctx, fmt.Sprintf("key:expiry:%s", key), info.Name, time.Duration(newTTLDays)*24*time.Hour)
		}
		return nil
	}

	return fmt.Errorf("key not found")
}

func (kem *KeyExpirationManager) GetExpiry(key string) time.Time {
	kem.mu.RLock()
	defer kem.mu.RUnlock()

	if info, exists := kem.keys[key]; exists {
		return info.ExpiresAt
	}
	return time.Time{}
}

func (kem *KeyExpirationManager) expirationLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		kem.mu.Lock()
		now := time.Now()
		for key, info := range kem.keys {
			if now.After(info.ExpiresAt) {
				delete(kem.keys, key)
			}
		}
		kem.mu.Unlock()
	}
}

func (kem *KeyExpirationManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		if authHeader != "" {
			key := extractBearerToken(authHeader)
			if key == "" {
				key = authHeader
			}

			if kem.IsExpired(key) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":      "API key expired",
					"expires_at": kem.GetExpiry(key),
				})
				return
			}
		}

		c.Next()
	}
}

type BandwidthQuota struct {
	mu     sync.RWMutex
	quotas map[string]*QuotaInfo
	redis  *redis.Client
	ctx    context.Context
}

type QuotaInfo struct {
	NodeID    string    `json:"node_id"`
	MonthlyGB int64     `json:"monthly_gb"`
	UsedGB    int64     `json:"used_gb"`
	ResetAt   time.Time `json:"reset_at"`
	HardLimit bool      `json:"hard_limit"`
}

func NewBandwidthQuota(redisClient *redis.Client) *BandwidthQuota {
	return &BandwidthQuota{
		quotas: make(map[string]*QuotaInfo),
		redis:  redisClient,
		ctx:    context.Background(),
	}
}

func (bq *BandwidthQuota) SetQuota(nodeID string, monthlyGB int64, hardLimit bool) error {
	quota := &QuotaInfo{
		NodeID:    nodeID,
		MonthlyGB: monthlyGB,
		UsedGB:    0,
		ResetAt:   getNextResetTime(),
		HardLimit: hardLimit,
	}

	bq.mu.Lock()
	bq.quotas[nodeID] = quota
	bq.mu.Unlock()

	if bq.redis != nil {
		bq.redis.HSet(bq.ctx, fmt.Sprintf("quota:%s", nodeID), "monthly_gb", monthlyGB)
		bq.redis.HSet(bq.ctx, fmt.Sprintf("quota:%s", nodeID), "used_gb", 0)
		bq.redis.HSet(bq.ctx, fmt.Sprintf("quota:%s", nodeID), "hard_limit", hardLimit)
	}

	return nil
}

func (bq *BandwidthQuota) CheckQuota(nodeID string, additionalGB int64) (bool, error) {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	quota, exists := bq.quotas[nodeID]
	if !exists {
		return true, nil
	}

	if time.Now().After(quota.ResetAt) {
		quota.UsedGB = 0
		quota.ResetAt = getNextResetTime()
	}

	newUsed := quota.UsedGB + additionalGB

	if newUsed > quota.MonthlyGB {
		if quota.HardLimit {
			return false, fmt.Errorf("bandwidth quota exceeded")
		}
		return true, nil
	}

	return true, nil
}

func (bq *BandwidthQuota) RecordUsage(nodeID string, additionalGB int64) error {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	quota, exists := bq.quotas[nodeID]
	if !exists {
		return nil
	}

	if time.Now().After(quota.ResetAt) {
		quota.UsedGB = 0
		quota.ResetAt = getNextResetTime()
	}

	quota.UsedGB += additionalGB

	if bq.redis != nil {
		bq.redis.HIncrBy(bq.ctx, fmt.Sprintf("quota:%s", nodeID), "used_gb", additionalGB*1024*1024*1024)
	}

	return nil
}

func (bq *BandwidthQuota) GetRemainingQuota(nodeID string) int64 {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	quota, exists := bq.quotas[nodeID]
	if !exists {
		return -1
	}

	remaining := quota.MonthlyGB - quota.UsedGB
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (bq *BandwidthQuota) GetQuotaInfo(nodeID string) *QuotaInfo {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	if quota, exists := bq.quotas[nodeID]; exists {
		return quota
	}
	return nil
}

func getNextResetTime() time.Time {
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, nextMonth.Location())
}

func (bq *BandwidthQuota) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/quotas", bq.setQuotaHandler)
	r.GET("/api/admin/quotas/:nodeId", bq.getQuotaHandler)
}

func (bq *BandwidthQuota) setQuotaHandler(c *gin.Context) {
	var req struct {
		NodeID    string `json:"node_id" binding:"required"`
		MonthlyGB int64  `json:"monthly_gb" binding:"required"`
		HardLimit bool   `json:"hard_limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := bq.SetQuota(req.NodeID, req.MonthlyGB, req.HardLimit); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (bq *BandwidthQuota) getQuotaHandler(c *gin.Context) {
	nodeID := c.Param("nodeId")
	info := bq.GetQuotaInfo(nodeID)
	if info == nil {
		c.JSON(404, gin.H{"error": "quota not found"})
		return
	}
	c.JSON(200, info)
}
