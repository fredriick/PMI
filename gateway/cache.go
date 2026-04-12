package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type ResponseCache struct {
	redis      *redis.Client
	ctx        context.Context
	mu         sync.RWMutex
	localCache map[string]*cacheEntry
	ttl        time.Duration
}

type cacheEntry struct {
	Response    string
	ContentType string
	Expiry      time.Time
}

func NewResponseCache(redisClient *redis.Client, ttl time.Duration) *ResponseCache {
	rc := &ResponseCache{
		redis:      redisClient,
		ctx:        context.Background(),
		localCache: make(map[string]*cacheEntry),
		ttl:        ttl,
	}

	go rc.cleanupLoop()

	return rc
}

func (rc *ResponseCache) Get(key string) (string, string, bool) {
	if rc.redis != nil {
		data, err := rc.redis.Get(rc.ctx, fmt.Sprintf("cache:%s", key)).Result()
		if err == nil {
			var entry cacheEntry
			if json.Unmarshal([]byte(data), &entry) == nil && time.Now().Before(entry.Expiry) {
				return entry.Response, entry.ContentType, true
			}
		}
	}

	rc.mu.RLock()
	defer rc.mu.RUnlock()
	if entry, exists := rc.localCache[key]; exists {
		if time.Now().Before(entry.Expiry) {
			return entry.Response, entry.ContentType, true
		}
		delete(rc.localCache, key)
	}
	return "", "", false
}

func (rc *ResponseCache) Set(key, response, contentType string) {
	entry := cacheEntry{
		Response:    response,
		ContentType: contentType,
		Expiry:      time.Now().Add(rc.ttl),
	}

	if rc.redis != nil {
		data, _ := json.Marshal(entry)
		rc.redis.Set(rc.ctx, fmt.Sprintf("cache:%s", key), string(data), rc.ttl)
	}

	rc.mu.Lock()
	rc.localCache[key] = &entry
	rc.mu.Unlock()
}

func (rc *ResponseCache) Invalidate(pattern string) {
	if rc.redis != nil {
		keys, _ := rc.redis.Keys(rc.ctx, fmt.Sprintf("cache:%s", pattern)).Result()
		for _, key := range keys {
			rc.redis.Del(rc.ctx, key)
		}
	}

	rc.mu.Lock()
	if strings.Contains(pattern, "*") {
		pattern = strings.ReplaceAll(pattern, "*", "")
		for key := range rc.localCache {
			if strings.Contains(key, pattern) {
				delete(rc.localCache, key)
			}
		}
	} else {
		delete(rc.localCache, pattern)
	}
	rc.mu.Unlock()
}

func (rc *ResponseCache) cleanupLoop() {
	ticker := time.NewTicker(rc.ttl)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, entry := range rc.localCache {
			if now.After(entry.Expiry) {
				delete(rc.localCache, key)
			}
		}
		rc.mu.Unlock()
	}
}

func (rc *ResponseCache) Middleware(ttl time.Duration, cacheKeyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		key := cacheKeyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		if response, contentType, found := rc.Get(key); found {
			c.Header("X-Cache", "HIT")
			c.Header("Content-Type", contentType)
			c.String(http.StatusOK, response)
			return
		}

		c.Header("X-Cache", "MISS")
		c.Next()
	}
}

type NodeRetryHandler struct {
	mu            sync.RWMutex
	maxRetries    int
	backoffBase   time.Duration
	fallbackNodes map[string][]string
}

func NewNodeRetryHandler(maxRetries int, backoffBase time.Duration) *NodeRetryHandler {
	return &NodeRetryHandler{
		maxRetries:    maxRetries,
		backoffBase:   backoffBase,
		fallbackNodes: make(map[string][]string),
	}
}

func (nrh *NodeRetryHandler) SetFallback(primaryNode string, fallbacks []string) {
	nrh.mu.Lock()
	defer nrh.mu.Unlock()
	nrh.fallbackNodes[primaryNode] = fallbacks
}

func (nrh *NodeRetryHandler) GetFallback(primaryNode string) string {
	nrh.mu.RLock()
	defer nrh.mu.RUnlock()

	fallbacks, exists := nrh.fallbackNodes[primaryNode]
	if !exists || len(fallbacks) == 0 {
		return ""
	}
	return fallbacks[0]
}

func (nrh *NodeRetryHandler) ExecuteWithRetry(fn func(nodeID string) error) error {
	var lastErr error

	for attempt := 0; attempt <= nrh.maxRetries; attempt++ {
		err := fn("")
		if err == nil {
			return nil
		}

		lastErr = err

		if attempt < nrh.maxRetries {
			delay := nrh.backoffBase * time.Duration(attempt+1)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

type RateLimitHeaders struct {
	Enabled          bool
	HeaderLimit      string
	HeaderRemaining  string
	HeaderReset      string
	HeaderRetryAfter string
}

func NewRateLimitHeaders(enabled bool) *RateLimitHeaders {
	return &RateLimitHeaders{
		Enabled:          enabled,
		HeaderLimit:      "X-RateLimit-Limit",
		HeaderRemaining:  "X-RateLimit-Remaining",
		HeaderReset:      "X-RateLimit-Reset",
		HeaderRetryAfter: "Retry-After",
	}
}

func (rlh *RateLimitHeaders) SetHeaders(c *gin.Context, limit, remaining int, reset time.Time) {
	if !rlh.Enabled {
		return
	}

	c.Header(rlh.HeaderLimit, fmt.Sprintf("%d", limit))
	c.Header(rlh.HeaderRemaining, fmt.Sprintf("%d", remaining))
	c.Header(rlh.HeaderReset, fmt.Sprintf("%d", reset.Unix()))

	if remaining == 0 {
		retryAfter := reset.Sub(time.Now()).Seconds()
		if retryAfter > 0 {
			c.Header(rlh.HeaderRetryAfter, fmt.Sprintf("%.0f", retryAfter))
		}
	}
}

func (rlh *RateLimitHeaders) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rlh.Enabled {
			c.Next()
			return
		}

		c.Next()

		limit := 100
		remaining := limit - 1

		rlh.SetHeaders(c, limit, remaining, time.Now().Add(time.Minute))
	}
}
