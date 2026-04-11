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

type TrafficShaper struct {
	redis        *redis.Client
	ctx          context.Context
	bandwidths   map[string]*clientBandwidth
	mu           sync.RWMutex
	defaultRate  int64
	defaultBurst int64
}

type clientBandwidth struct {
	tokens     int64
	lastUpdate time.Time
	rate       int64
	burst      int64
}

func NewTrafficShaper(redisClient *redis.Client, defaultRate, defaultBurst int64) *TrafficShaper {
	ts := &TrafficShaper{
		redis:        redisClient,
		ctx:          context.Background(),
		bandwidths:   make(map[string]*clientBandwidth),
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
	}

	go ts.cleanupLoop()

	return ts
}

func (ts *TrafficShaper) SetClientRate(clientIP string, rate, burst int64) {
	key := fmt.Sprintf("shaper:%s", clientIP)
	if ts.redis != nil {
		ts.redis.HSet(ts.ctx, key, "rate", rate)
		ts.redis.HSet(ts.ctx, key, "burst", burst)
		ts.redis.Expire(ts.ctx, key, 24*time.Hour)
	}
}

func (ts *TrafficShaper) Middleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if ts.isWhitelisted(clientIP) {
			c.Next()
			return
		}

		allowed, err := ts.checkBandwidth(clientIP, 0)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Bandwidth check failed",
			})
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Bandwidth limit exceeded",
				"retry_after": 1,
			})
			return
		}

		c.Next()
	}
}

func (ts *TrafficShaper) checkBandwidth(clientIP string, bytes int64) (bool, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	bw, exists := ts.bandwidths[clientIP]
	now := time.Now()

	if !exists {
		bw = &clientBandwidth{
			tokens:     ts.defaultBurst,
			lastUpdate: now,
			rate:       ts.defaultRate,
			burst:      ts.defaultBurst,
		}
		ts.bandwidths[clientIP] = bw
		return true, nil
	}

	elapsed := now.Sub(bw.lastUpdate).Seconds()
	bw.tokens += int64(float64(bw.rate) * elapsed)
	if bw.tokens > bw.burst {
		bw.tokens = bw.burst
	}

	if bw.tokens >= bytes {
		bw.tokens -= bytes
		bw.lastUpdate = now
		return true, nil
	}

	return false, nil
}

func (ts *TrafficShaper) isWhitelisted(clientIP string) bool {
	if ip := net.ParseIP(clientIP); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return true
		}
	}
	return false
}

func (ts *TrafficShaper) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ts.mu.Lock()
		now := time.Now()
		for ip, bw := range ts.bandwidths {
			if now.Sub(bw.lastUpdate) > 10*time.Minute {
				delete(ts.bandwidths, ip)
			}
		}
		ts.mu.Unlock()
	}
}

type DDoSProtection struct {
	redis          *redis.Client
	ctx            context.Context
	failedRequests map[string]*failedRequestStats
	mu             sync.RWMutex
	threshold      int
	windowSeconds  int
	blockDuration  time.Duration
	whitelist      map[string]bool
	whitelistMu    sync.RWMutex
}

type failedRequestStats struct {
	count     int
	firstFail time.Time
	lastFail  time.Time
}

func NewDDoSProtection(redisClient *redis.Client, threshold int, windowSeconds int, blockDuration time.Duration) *DDoSProtection {
	return &DDoSProtection{
		redis:          redisClient,
		ctx:            context.Background(),
		failedRequests: make(map[string]*failedRequestStats),
		threshold:      threshold,
		windowSeconds:  windowSeconds,
		blockDuration:  blockDuration,
		whitelist:      make(map[string]bool),
	}
}

func (ddos *DDoSProtection) Middleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if ddos.isWhitelisted(clientIP) {
			c.Next()
			return
		}

		if ddos.isBlocked(clientIP) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "IP blocked due to suspected DDoS",
			})
			return
		}

		c.Next()
	}
}

func (ddos *DDoSProtection) RecordFailure(clientIP string) {
	key := fmt.Sprintf("ddos:failed:%s", clientIP)

	if ddos.redis != nil {
		count, _ := ddos.redis.Incr(ddos.ctx, key).Result()
		if count == 1 {
			ddos.redis.Expire(ddos.ctx, key, time.Duration(ddos.windowSeconds)*time.Second)
		}

		if int(count) >= ddos.threshold {
			blockKey := fmt.Sprintf("ddos:blocked:%s", clientIP)
			ddos.redis.Set(ddos.ctx, blockKey, 1, ddos.blockDuration)
		}
		return
	}

	ddos.mu.Lock()
	defer ddos.mu.Unlock()

	stats, exists := ddos.failedRequests[clientIP]
	if !exists {
		stats = &failedRequestStats{
			firstFail: time.Now(),
		}
		ddos.failedRequests[clientIP] = stats
	}

	stats.count++
	stats.lastFail = time.Now()

	if stats.count >= ddos.threshold {
		blockKey := fmt.Sprintf("ddos:blocked:%s", clientIP)
		if ddos.redis != nil {
			ddos.redis.Set(ddos.ctx, blockKey, 1, ddos.blockDuration)
		}
	}
}

func (ddos *DDoSProtection) isBlocked(clientIP string) bool {
	if ddos.redis != nil {
		blockKey := fmt.Sprintf("ddos:blocked:%s", clientIP)
		exists, _ := ddos.redis.Exists(ddos.ctx, blockKey).Result()
		return exists > 0
	}

	ddos.mu.RLock()
	defer ddos.mu.RUnlock()

	if stats, exists := ddos.failedRequests[clientIP]; exists {
		if stats.count >= ddos.threshold {
			windowStart := time.Now().Add(-time.Duration(ddos.windowSeconds) * time.Second)
			if stats.firstFail.After(windowStart) {
				return true
			}
		}
	}

	return false
}

func (ddos *DDoSProtection) BlockIP(clientIP string) {
	key := fmt.Sprintf("ddos:blocked:%s", clientIP)
	if ddos.redis != nil {
		ddos.redis.Set(ddos.ctx, key, 1, ddos.blockDuration)
	}
}

func (ddos *DDoSProtection) UnblockIP(clientIP string) {
	key := fmt.Sprintf("ddos:blocked:%s", clientIP)
	if ddos.redis != nil {
		ddos.redis.Del(ddos.ctx, key)
	}
}

func (ddos *DDoSProtection) AddToWhitelist(ip string) {
	ddos.whitelistMu.Lock()
	defer ddos.whitelistMu.Unlock()
	ddos.whitelist[ip] = true
}

func (ddos *DDoSProtection) RemoveFromWhitelist(ip string) {
	ddos.whitelistMu.Lock()
	defer ddos.whitelistMu.Unlock()
	delete(ddos.whitelist, ip)
}

func (ddos *DDoSProtection) isWhitelisted(clientIP string) bool {
	ddos.whitelistMu.RLock()
	defer ddos.whitelistMu.RUnlock()
	return ddos.whitelist[clientIP]
}
