package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RequestDedup struct {
	mu       sync.RWMutex
	requests map[string]*DedupRequest
	ttl      time.Duration
}

type DedupRequest struct {
	Hash      string
	CreatedAt time.Time
	NodeID    string
}

func NewRequestDedup(ttl time.Duration) *RequestDedup {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	rd := &RequestDedup{
		requests: make(map[string]*DedupRequest),
		ttl:      ttl,
	}

	go rd.cleanupLoop()

	return rd
}

func (rd *RequestDedup) ComputeHash(method, url, body, user string) string {
	data := fmt.Sprintf("%s|%s|%s|%s", method, url, body, user)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (rd *RequestDedup) CheckAndMark(hash, nodeID string) (bool, string) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if existing, exists := rd.requests[hash]; exists {
		if time.Since(existing.CreatedAt) < rd.ttl {
			return false, existing.NodeID
		}
	}

	rd.requests[hash] = &DedupRequest{
		Hash:      hash,
		CreatedAt: time.Now(),
		NodeID:    nodeID,
	}

	return true, ""
}

func (rd *RequestDedup) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hash := rd.ComputeHash(
			c.Request.Method,
			c.Request.URL.String(),
			c.GetHeader("Content-Type"),
			c.GetHeader("X-User"),
		)

		nodeID := c.GetHeader("X-Node-ID")
		if nodeID == "" {
			nodeID = "unknown"
		}

		isNew, existingNode := rd.CheckAndMark(hash, nodeID)
		if !isNew {
			c.Header("X-Dedup", "HIT")
			c.Header("X-Dedup-Node", existingNode)
		} else {
			c.Header("X-Dedup", "MISS")
		}

		c.Next()
	}
}

func (rd *RequestDedup) cleanupLoop() {
	ticker := time.NewTicker(rd.ttl)
	defer ticker.Stop()

	for range ticker.C {
		rd.mu.Lock()
		now := time.Now()
		for hash, req := range rd.requests {
			if now.Sub(req.CreatedAt) > rd.ttl {
				delete(rd.requests, hash)
			}
		}
		rd.mu.Unlock()
	}
}
