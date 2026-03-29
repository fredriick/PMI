package gateway

import (
	"fmt"
	"net/http"
	"time"

	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type DistributedRateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	ctx    context.Context
}

func NewDistributedRateLimiter(client *redis.Client, requestsPerWindow int, windowSeconds int) *DistributedRateLimiter {
	return &DistributedRateLimiter{
		client: client,
		limit:  requestsPerWindow,
		window: time.Duration(windowSeconds) * time.Second,
		ctx:    context.Background(),
	}
}

func (rl *DistributedRateLimiter) Allow(clientID string) (bool, error) {
	key := fmt.Sprintf("ratelimit:%s", clientID)
	now := time.Now()
	windowStart := now.Add(-rl.window)

	pipe := rl.client.Pipeline()

	pipe.ZRemRangeByScore(rl.ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	pipe.ZCard(rl.ctx, key)

	pipe.ZAdd(rl.ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	pipe.Expire(rl.ctx, key, rl.window+time.Second)

	results, err := pipe.Exec(rl.ctx)
	if err != nil {
		return true, err
	}

	count := results[1].(*redis.IntCmd).Val()

	return count < int64(rl.limit), nil
}

func (rl *DistributedRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			authHeader = c.Request.Header.Get("Proxy-Authorization")
		}

		clientID := c.ClientIP()
		if authHeader != "" {
			clientID = clientID + ":" + authHeader
		}

		allowed, err := rl.Allow(clientID)
		if err != nil {
			c.Next()
			return
		}

		if !allowed {
			metrics.IncRateLimited()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
