package gateway

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string]*clientRateLimit
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type clientRateLimit struct {
	count     int
	resetTime time.Time
}

func NewRateLimiter(requestsPerWindow int, windowSeconds int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRateLimit),
		limit:    requestsPerWindow,
		window:   time.Duration(windowSeconds) * time.Second,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
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

func (rl *RateLimiter) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.requests[clientID]

	if !exists || now.After(client.resetTime) {
		rl.requests[clientID] = &clientRateLimit{
			count:     1,
			resetTime: now.Add(rl.window),
		}
		return true
	}

	if client.count >= rl.limit {
		metrics.IncRateLimited()
		return false
	}

	client.count++
	return true
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		clientID := c.ClientIP()
		if authHeader != "" {
			clientID = clientID + ":" + authHeader
		}

		if !rl.Allow(clientID) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
