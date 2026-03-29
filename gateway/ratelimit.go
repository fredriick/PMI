package gateway

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter interface {
	Allow(clientID string) (bool, error)
	Middleware() gin.HandlerFunc
}

type LocalRateLimiter struct {
	requests map[string]*clientRateLimit
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type clientRateLimit struct {
	count     int
	resetTime time.Time
}

func NewLocalRateLimiter(requestsPerWindow int, windowSeconds int) *LocalRateLimiter {
	rl := &LocalRateLimiter{
		requests: make(map[string]*clientRateLimit),
		limit:    requestsPerWindow,
		window:   time.Duration(windowSeconds) * time.Second,
	}

	go rl.cleanup()

	return rl
}

func (rl *LocalRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, client := range rl.requests {
			if now.After(client.resetTime) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *LocalRateLimiter) Allow(clientID string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.requests[clientID]

	if !exists || now.After(client.resetTime) {
		rl.requests[clientID] = &clientRateLimit{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true, nil
	}

	if client.count >= rl.limit {
		metrics.IncRateLimited()
		return false, nil
	}

	client.count++
	return true, nil
}

func (rl *LocalRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		clientID := c.ClientIP()
		if authHeader != "" {
			clientID = clientID + ":" + authHeader
		}

		allowed, _ := rl.Allow(clientID)
		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
