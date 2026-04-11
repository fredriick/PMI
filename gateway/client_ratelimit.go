package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type ClientRateLimiter struct {
	redis         *redis.Client
	defaultLimit  int
	windowSeconds int
	localCache    map[string]*clientLimit
	mu            sync.RWMutex
	ctx           context.Context
}

type clientLimit struct {
	count     int
	expiresAt time.Time
}

func NewClientRateLimiter(redisClient *redis.Client, defaultLimit, windowSeconds int) *ClientRateLimiter {
	crl := &ClientRateLimiter{
		redis:         redisClient,
		defaultLimit:  defaultLimit,
		windowSeconds: windowSeconds,
		localCache:    make(map[string]*clientLimit),
		ctx:           context.Background(),
	}

	go crl.cleanupLoop()

	return crl
}

func (crl *ClientRateLimiter) Middleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if crl.isWhitelisted(clientIP) {
			c.Next()
			return
		}

		limit := crl.getClientLimit(clientIP)
		if limit == 0 {
			limit = crl.defaultLimit
		}

		allowed, err := crl.checkRateLimit(clientIP, limit)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			return
		}

		if !allowed {
			metrics.IncRateLimited()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": crl.windowSeconds,
			})
			return
		}

		c.Next()
	}
}

func (crl *ClientRateLimiter) checkRateLimit(clientIP string, limit int) (bool, error) {
	if crl.redis == nil {
		return crl.localCheck(clientIP, limit)
	}

	key := fmt.Sprintf("client_rate:%s", clientIP)

	count, err := crl.redis.Incr(crl.ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		crl.redis.Expire(crl.ctx, key, time.Duration(crl.windowSeconds)*time.Second)
	}

	return count <= int64(limit), nil
}

func (crl *ClientRateLimiter) localCheck(clientIP string, limit int) (bool, error) {
	crl.mu.RLock()
	cl, exists := crl.localCache[clientIP]
	crl.mu.RUnlock()

	now := time.Now()

	if exists && now.Before(cl.expiresAt) {
		if cl.count >= limit {
			return false, nil
		}
		cl.count++
		return true, nil
	}

	crl.mu.Lock()
	crl.localCache[clientIP] = &clientLimit{
		count:     1,
		expiresAt: now.Add(time.Duration(crl.windowSeconds) * time.Second),
	}
	crl.mu.Unlock()

	return true, nil
}

func (crl *ClientRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		crl.mu.Lock()
		now := time.Now()
		for ip, cl := range crl.localCache {
			if now.After(cl.expiresAt) {
				delete(crl.localCache, ip)
			}
		}
		crl.mu.Unlock()
	}
}

func (crl *ClientRateLimiter) isWhitelisted(clientIP string) bool {
	if ip := net.ParseIP(clientIP); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return true
		}
	}
	return false
}

func (crl *ClientRateLimiter) SetClientLimit(clientIP string, limit int) {
	if crl.redis != nil {
		key := fmt.Sprintf("client_limit:%s", clientIP)
		crl.redis.Set(crl.ctx, key, limit, 0)
	}
}

func (crl *ClientRateLimiter) GetClientLimit(clientIP string) (int, error) {
	if crl.redis == nil {
		return crl.defaultLimit, nil
	}

	key := fmt.Sprintf("client_limit:%s", clientIP)
	limit, err := crl.redis.Get(crl.ctx, key).Int()
	if err == redis.Nil {
		return crl.defaultLimit, nil
	}
	return limit, err
}

func (crl *ClientRateLimiter) getClientLimit(clientIP string) int {
	if limit, err := crl.GetClientLimit(clientIP); err == nil {
		return limit
	}
	return 0
}

func (crl *ClientRateLimiter) RemoveClientLimit(clientIP string) {
	if crl.redis != nil {
		key := fmt.Sprintf("client_limit:%s", clientIP)
		crl.redis.Del(crl.ctx, key)
	}
}
