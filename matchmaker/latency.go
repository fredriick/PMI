package matchmaker

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type LatencyTracker struct {
	redis      *redis.Client
	ctx        context.Context
	latencies  map[string]*nodeLatency
	mu         sync.RWMutex
	windowSize time.Duration
	maxSamples int
}

type nodeLatency struct {
	samples    []int64
	sum        int64
	avgLatency int64
	lastUpdate time.Time
}

func NewLatencyTracker(redisClient *redis.Client, windowSize time.Duration, maxSamples int) *LatencyTracker {
	lt := &LatencyTracker{
		redis:      redisClient,
		ctx:        context.Background(),
		latencies:  make(map[string]*nodeLatency),
		windowSize: windowSize,
		maxSamples: maxSamples,
	}

	go lt.cleanupLoop()

	return lt
}

func (lt *LatencyTracker) RecordLatency(nodeID string, latencyMs int64) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	latency, exists := lt.latencies[nodeID]
	if !exists {
		latency = &nodeLatency{
			samples:    make([]int64, 0, lt.maxSamples),
			lastUpdate: time.Now(),
		}
		lt.latencies[nodeID] = latency
	}

	latency.samples = append(latency.samples, latencyMs)
	latency.sum += latencyMs

	if len(latency.samples) > lt.maxSamples {
		oldest := latency.samples[0]
		latency.samples = latency.samples[1:]
		latency.sum -= oldest
	}

	latency.avgLatency = latency.sum / int64(len(latency.samples))
	latency.lastUpdate = time.Now()

	if lt.redis != nil {
		key := "latency:ranking"
		lt.redis.HSet(lt.ctx, key, nodeID, latency.avgLatency)
		lt.redis.Expire(lt.ctx, key, lt.windowSize)
	}
}

func (lt *LatencyTracker) GetAvgLatency(nodeID string) int64 {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if latency, exists := lt.latencies[nodeID]; exists {
		return latency.avgLatency
	}
	return 0
}

func (lt *LatencyTracker) GetTopNodes(count int) []string {
	if lt.redis != nil {
		results, err := lt.redis.ZRange(lt.ctx, "latency:ranking", 0, int64(count-1)).Result()
		if err == nil {
			return results
		}
	}

	lt.mu.RLock()
	defer lt.mu.RUnlock()

	type latencyPair struct {
		nodeID  string
		latency int64
	}

	var pairs []latencyPair
	for nodeID, latency := range lt.latencies {
		pairs = append(pairs, latencyPair{nodeID, latency.avgLatency})
	}

	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].latency < pairs[i].latency {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	result := make([]string, 0, count)
	for i := 0; i < len(pairs) && i < count; i++ {
		result = append(result, pairs[i].nodeID)
	}

	return result
}

func (lt *LatencyTracker) GetLatencyStats(nodeID string) (int64, int64, int64) {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if latency, exists := lt.latencies[nodeID]; exists {
		var min, max int64 = latency.samples[0], latency.samples[0]
		for _, s := range latency.samples {
			if s < min {
				min = s
			}
			if s > max {
				max = s
			}
		}
		return latency.avgLatency, min, max
	}

	return 0, 0, 0
}

func (lt *LatencyTracker) cleanupLoop() {
	ticker := time.NewTicker(lt.windowSize)
	defer ticker.Stop()

	for range ticker.C {
		lt.mu.Lock()
		now := time.Now()
		for nodeID, latency := range lt.latencies {
			if now.Sub(latency.lastUpdate) > lt.windowSize*2 {
				delete(lt.latencies, nodeID)
			}
		}
		lt.mu.Unlock()
	}
}

type RequestQueue struct {
	mu           sync.RWMutex
	queues       map[string]chan *QueuedRequest
	maxQueueSize int
}

type QueuedRequest struct {
	Request   interface{}
	Result    chan interface{}
	CreatedAt time.Time
}

func NewRequestQueue(maxQueueSize int) *RequestQueue {
	return &RequestQueue{
		queues:       make(map[string]chan *QueuedRequest),
		maxQueueSize: maxQueueSize,
	}
}

func (rq *RequestQueue) Enqueue(nodeID string, req *QueuedRequest) bool {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	ch, exists := rq.queues[nodeID]
	if !exists {
		size := rq.maxQueueSize
		if size <= 0 {
			size = 100
		}
		rq.queues[nodeID] = make(chan *QueuedRequest, size)
		ch = rq.queues[nodeID]
	}

	select {
	case ch <- req:
		return true
	default:
		return false
	}
}

func (rq *RequestQueue) Dequeue(nodeID string) *QueuedRequest {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	ch, exists := rq.queues[nodeID]
	if !exists {
		return nil
	}

	select {
	case req := <-ch:
		return req
	default:
		return nil
	}
}

func (rq *RequestQueue) GetQueueSize(nodeID string) int {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	ch, exists := rq.queues[nodeID]
	if !exists {
		return 0
	}

	return len(ch)
}

func (rq *RequestQueue) RemoveQueue(nodeID string) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	delete(rq.queues, nodeID)
}

type ReputationDecay struct {
	mu            sync.RWMutex
	reputations   map[string]*reputationState
	decayRate     float64
	decayInterval time.Duration
	minReputation float64
}

type reputationState struct {
	score     float64
	lastDecay time.Time
}

func NewReputationDecay(decayRate float64, decayInterval time.Duration, minReputation float64) *ReputationDecay {
	rd := &ReputationDecay{
		reputations:   make(map[string]*reputationState),
		decayRate:     decayRate,
		decayInterval: decayInterval,
		minReputation: minReputation,
	}

	go rd.decayLoop()

	return rd
}

func (rd *ReputationDecay) GetReputation(nodeID string) float64 {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	if state, exists := rd.reputations[nodeID]; exists {
		return state.score
	}
	return 100.0
}

func (rd *ReputationDecay) SetReputation(nodeID string, score float64) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.reputations[nodeID] = &reputationState{
		score:     score,
		lastDecay: time.Now(),
	}
}

func (rd *ReputationDecay) IncrementReputation(nodeID string, amount float64) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	state, exists := rd.reputations[nodeID]
	if !exists {
		state = &reputationState{
			score:     100.0,
			lastDecay: time.Now(),
		}
		rd.reputations[nodeID] = state
	}

	state.score += amount
	if state.score > 100.0 {
		state.score = 100.0
	}
}

func (rd *ReputationDecay) decayLoop() {
	ticker := time.NewTicker(rd.decayInterval)
	defer ticker.Stop()

	for range ticker.C {
		rd.mu.Lock()
		now := time.Now()
		for _, state := range rd.reputations {
			if now.Sub(state.lastDecay) >= rd.decayInterval {
				state.score *= (1.0 - rd.decayRate)
				if state.score < rd.minReputation {
					state.score = rd.minReputation
				}
				state.lastDecay = now
			}
		}
		rd.mu.Unlock()
	}
}
