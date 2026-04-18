package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type KeyAnalytics struct {
	mu    sync.RWMutex
	stats map[string]*KeyStat
	redis *redis.Client
	ctx   context.Context
}

type KeyStat struct {
	Key       string    `json:"key"`
	Requests  int64     `json:"requests"`
	BytesSent int64     `json:"bytes_sent"`
	BytesRecv int64     `json:"bytes_recv"`
	Errors    int64     `json:"errors"`
	LastUsed  time.Time `json:"last_used"`
	FirstUsed time.Time `json:"first_used"`
}

func NewKeyAnalytics(redisClient *redis.Client) *KeyAnalytics {
	return &KeyAnalytics{
		stats: make(map[string]*KeyStat),
		redis: redisClient,
		ctx:   context.Background(),
	}
}

func (ka *KeyAnalytics) RecordRequest(key string, bytesSent, bytesRecv int64, isError bool) {
	ka.mu.Lock()
	defer ka.mu.Unlock()

	stat, exists := ka.stats[key]
	if !exists {
		stat = &KeyStat{
			Key:       key,
			FirstUsed: time.Now(),
		}
		ka.stats[key] = stat
	}

	stat.Requests++
	stat.BytesSent += bytesSent
	stat.BytesRecv += bytesRecv
	stat.LastUsed = time.Now()

	if isError {
		stat.Errors++
	}

	if ka.redis != nil {
		ka.redis.HIncrBy(ka.ctx, fmt.Sprintf("key:stat:%s", key), "requests", 1)
		ka.redis.HIncrBy(ka.ctx, fmt.Sprintf("key:stat:%s", key), "bytes_sent", bytesSent)
		ka.redis.HIncrBy(ka.ctx, fmt.Sprintf("key:stat:%s", key), "bytes_recv", bytesRecv)
		if isError {
			ka.redis.HIncrBy(ka.ctx, fmt.Sprintf("key:stat:%s", key), "errors", 1)
		}
	}
}

func (ka *KeyAnalytics) GetStats(key string) *KeyStat {
	ka.mu.RLock()
	defer ka.mu.RUnlock()
	return ka.stats[key]
}

func (ka *KeyAnalytics) GetTopKeys(limit int) []*KeyStat {
	ka.mu.RLock()
	defer ka.mu.RUnlock()

	stats := make([]*KeyStat, 0, len(ka.stats))
	for _, s := range ka.stats {
		stats = append(stats, s)
	}

	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Requests > stats[i].Requests {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if limit > len(stats) {
		limit = len(stats)
	}
	return stats[:limit]
}

func (ka *KeyAnalytics) Middleware() gin.HandlerFunc {
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

			c.Set("api_key", key)
		}

		c.Next()

		if key, exists := c.Get("api_key"); exists {
			ka.RecordRequest(key.(string), 0, 0, c.Writer.Status() >= 500)
		}
	}
}

func (ka *KeyAnalytics) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/admin/analytics/keys", ka.listKeysHandler)
	r.GET("/api/admin/analytics/keys/:key", ka.getKeyHandler)
}

func (ka *KeyAnalytics) listKeysHandler(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	stats := ka.GetTopKeys(limit)
	c.JSON(200, gin.H{"keys": stats})
}

func (ka *KeyAnalytics) getKeyHandler(c *gin.Context) {
	key := c.Param("key")
	stat := ka.GetStats(key)
	if stat == nil {
		c.JSON(404, gin.H{"error": "key not found"})
		return
	}
	c.JSON(200, stat)
}

type RequestIDConfig struct {
	mu     sync.RWMutex
	prefix string
	format string
}

func NewRequestIDConfig() *RequestIDConfig {
	return &RequestIDConfig{
		prefix: "req",
		format: "unix",
	}
}

func (ric *RequestIDConfig) SetPrefix(prefix string) {
	ric.mu.Lock()
	defer ric.mu.Unlock()
	ric.prefix = prefix
}

func (ric *RequestIDConfig) SetFormat(format string) {
	ric.mu.Lock()
	defer ric.mu.Unlock()
	ric.format = format
}

func (ric *RequestIDConfig) Generate() string {
	ric.mu.RLock()
	defer ric.mu.RUnlock()

	switch ric.format {
	case "uuid":
		return ric.prefix + "-" + fmt.Sprintf("-%d", time.Now().UnixNano())
	case "timestamp":
		return ric.prefix + fmt.Sprintf("-%d", time.Now().Unix())
	default:
		return ric.prefix + fmt.Sprintf("-%d", time.Now().UnixNano())
	}
}

func (ric *RequestIDConfig) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/request-id", ric.setConfigHandler)
	r.GET("/api/admin/request-id", ric.getConfigHandler)
}

func (ric *RequestIDConfig) setConfigHandler(c *gin.Context) {
	var req struct {
		Prefix string `json:"prefix"`
		Format string `json:"format"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if req.Prefix != "" {
		ric.SetPrefix(req.Prefix)
	}
	if req.Format != "" {
		ric.SetFormat(req.Format)
	}

	c.JSON(200, gin.H{"status": "success"})
}

func (ric *RequestIDConfig) getConfigHandler(c *gin.Context) {
	ric.mu.RLock()
	defer ric.mu.RUnlock()

	c.JSON(200, gin.H{
		"prefix": ric.prefix,
		"format": ric.format,
	})
}
