package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type RequestDeduplicator struct {
	mu       sync.RWMutex
	inflight map[string]*inflightRequest
	ttl      time.Duration
}

type inflightRequest struct {
	done     chan struct{}
	err      error
	nodeID   string
	refCount int
}

func NewRequestDeduplicator(ttl time.Duration) *RequestDeduplicator {
	rd := &RequestDeduplicator{
		inflight: make(map[string]*inflightRequest),
		ttl:      ttl,
	}

	go rd.cleanupLoop()

	return rd
}

func (rd *RequestDeduplicator) getKey(method, target, country, city string) string {
	data := fmt.Sprintf("%s:%s:%s:%s", method, target, country, city)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (rd *RequestDeduplicator) BeginRequest(method, target, country, city string) (key string, done <-chan struct{}, nodeID string, isDuplicate bool) {
	key = rd.getKey(method, target, country, city)

	rd.mu.Lock()
	defer rd.mu.Unlock()

	if existing, ok := rd.inflight[key]; ok {
		existing.refCount++
		return key, existing.done, existing.nodeID, true
	}

	inflight := &inflightRequest{
		done:     make(chan struct{}),
		refCount: 1,
	}
	rd.inflight[key] = inflight

	return key, inflight.done, "", false
}

func (rd *RequestDeduplicator) CompleteRequest(key, nodeID string, err error) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if existing, ok := rd.inflight[key]; ok {
		existing.err = err
		existing.nodeID = nodeID
		close(existing.done)
		delete(rd.inflight, key)
	}
}

func (rd *RequestDeduplicator) GetNodeForRequest(key string) string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if existing, ok := rd.inflight[key]; ok {
		return existing.nodeID
	}
	return ""
}

func (rd *RequestDeduplicator) cleanupLoop() {
	ticker := time.NewTicker(rd.ttl)
	defer ticker.Stop()

	for range ticker.C {
		rd.mu.Lock()
		for key, req := range rd.inflight {
			if req.refCount <= 0 {
				delete(rd.inflight, key)
			}
		}
		rd.mu.Unlock()
	}
}

func (rd *RequestDeduplicator) Stats() (int, int) {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	total := len(rd.inflight)
	pending := 0
	for _, req := range rd.inflight {
		if req.refCount > 1 {
			pending++
		}
	}

	return total, pending
}
