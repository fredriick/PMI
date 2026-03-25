package matchmaker

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"proxymesh/internal/models"
)

type Matchmaker struct {
	redis           *RedisClient
	circuitBreakers map[string]*models.CircuitBreaker
	mu              sync.RWMutex
	threshold       int
}

func NewMatchmaker(redis *RedisClient, threshold int) *Matchmaker {
	return &Matchmaker{
		redis:           redis,
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       threshold,
	}
}

func (m *Matchmaker) SelectNode(country, city, targetDomain string) (*models.Node, error) {
	candidates, err := m.redis.GetTopNodes(country, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidates: %w", err)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no nodes available for country: %s", country)
	}

	var eligibleNodes []string
	for _, nodeID := range candidates {
		if m.isNodeHealthy(nodeID) {
			inCooldown, err := m.redis.IsInCooldown(targetDomain, nodeID)
			if err == nil && !inCooldown {
				eligibleNodes = append(eligibleNodes, nodeID)
			}
		}
	}

	if len(eligibleNodes) == 0 {
		return nil, fmt.Errorf("no eligible nodes after filtering")
	}

	selectedID, err := m.randomSelect(eligibleNodes)
	if err != nil {
		return nil, err
	}

	meta, err := m.redis.GetNodeMeta(selectedID)
	if err != nil {
		return nil, err
	}

	return &models.Node{
		ID:       selectedID,
		Country:  country,
		City:     city,
		ISP:      meta.ISP,
		OS:       meta.OS,
		LastSeen: meta.LastSeen,
	}, nil
}

func (m *Matchmaker) randomSelect(nodes []string) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("no nodes to select from")
	}

	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(nodes))))
	if err != nil {
		return "", err
	}

	return nodes[idx.Int64()], nil
}

func (m *Matchmaker) isNodeHealthy(nodeID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cb, exists := m.circuitBreakers[nodeID]
	if !exists {
		return true
	}

	return cb.State == "closed"
}

func (m *Matchmaker) RecordFailure(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cb, exists := m.circuitBreakers[nodeID]
	if !exists {
		cb = &models.CircuitBreaker{
			Threshold: m.threshold,
			State:     "closed",
		}
		m.circuitBreakers[nodeID] = cb
	}

	cb.FailureCount++
	if cb.FailureCount >= cb.Threshold {
		cb.State = "open"
	}
}

func (m *Matchmaker) RecordSuccess(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cb, exists := m.circuitBreakers[nodeID]; exists {
		cb.FailureCount = 0
		cb.State = "closed"
	}
}

func (m *Matchmaker) AddToCooldown(target string, nodeID string) error {
	return m.redis.AddToCooldown(target, nodeID)
}
