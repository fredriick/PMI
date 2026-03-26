package matchmaker

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"proxymesh/internal/models"
)

type Matchmaker struct {
	redis           *RedisClient
	circuitBreakers map[string]*models.CircuitBreaker
	mu              sync.RWMutex
	threshold       int
	stopHealthCheck chan bool
}

func NewMatchmaker(redis *RedisClient, threshold int) *Matchmaker {
	mm := &Matchmaker{
		redis:           redis,
		circuitBreakers: make(map[string]*models.CircuitBreaker),
		threshold:       threshold,
		stopHealthCheck: make(chan bool),
	}

	go mm.healthCheckLoop()

	return mm
}

func (m *Matchmaker) StopHealthCheck() {
	close(m.stopHealthCheck)
}

func (m *Matchmaker) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkNodeHealth()
		case <-m.stopHealthCheck:
			return
		}
	}
}

func (m *Matchmaker) checkNodeHealth() {
	countries := []string{"US", "GB", "DE", "FR", "CA", "AU", "JP"}

	for _, country := range countries {
		nodes, err := m.redis.GetTopNodes(country, 20)
		if err != nil {
			continue
		}

		for _, nodeID := range nodes {
			if !m.pingNode(nodeID) {
				m.RecordFailure(nodeID)
			}
		}
	}
}

func (m *Matchmaker) pingNode(nodeID string) bool {
	meta, err := m.redis.GetNodeMeta(nodeID)
	if err != nil {
		return false
	}

	age := time.Since(meta.LastSeen)
	return age < 5*time.Minute
}

func (m *Matchmaker) SelectNode(country, city, targetDomain string) (*models.Node, error) {
	var candidates []string
	var err error

	if city != "" {
		candidates, err = m.redis.GetNodesByCity(country, city, 50)
		if err != nil {
			return nil, fmt.Errorf("failed to get city nodes: %w", err)
		}
	}

	if len(candidates) == 0 {
		candidates, err = m.redis.GetTopNodes(country, 50)
		if err != nil {
			return nil, fmt.Errorf("failed to get candidates: %w", err)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no nodes available for country: %s", country)
	}

	eligibleNodes := m.filterEligibleNodes(candidates, targetDomain)

	if len(eligibleNodes) == 0 {
		return nil, fmt.Errorf("no eligible nodes after filtering")
	}

	selectedID, err := m.selectByLoad(eligibleNodes)
	if err != nil {
		return nil, err
	}

	m.redis.IncrementNodeLoad(selectedID)

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

func (m *Matchmaker) filterEligibleNodes(candidates []string, targetDomain string) []string {
	var eligible []string
	for _, nodeID := range candidates {
		if m.isNodeHealthy(nodeID) {
			inCooldown, err := m.redis.IsInCooldown(targetDomain, nodeID)
			if err == nil && !inCooldown {
				eligible = append(eligible, nodeID)
			}
		}
	}
	return eligible
}

func (m *Matchmaker) selectByLoad(nodes []string) (string, error) {
	type nodeWithLoad struct {
		id   string
		load int64
	}

	var nodeLoads []nodeWithLoad
	for _, nodeID := range nodes {
		load, _ := m.redis.GetNodeLoad(nodeID)
		nodeLoads = append(nodeLoads, nodeWithLoad{id: nodeID, load: load})
	}

	var lowestLoad int64 = 1 << 60
	var selected string

	for _, nl := range nodeLoads {
		if nl.load < lowestLoad {
			lowestLoad = nl.load
			selected = nl.id
		}
	}

	if selected == "" {
		return m.randomSelect(nodes)
	}

	return selected, nil
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

func (m *Matchmaker) RegisterNode(req *models.NodeRegistrationRequest) error {
	if req.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}
	if req.Country == "" {
		return fmt.Errorf("country is required")
	}
	if req.IP == "" {
		return fmt.Errorf("ip is required")
	}

	node := &models.Node{
		ID:         req.NodeID,
		NodeType:   req.NodeType,
		Country:    req.Country,
		City:       req.City,
		ISP:        req.ISP,
		IP:         req.IP,
		IPv6Subnet: req.IPv6Subnet,
		OS:         req.OS,
		LastSeen:   time.Now(),
		Reputation: 100.0,
	}

	if err := m.redis.AddNode(node); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	return nil
}

func (m *Matchmaker) Heartbeat(req *models.NodeHeartbeatRequest) error {
	if req.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	return m.redis.UpdateNodeStatus(req.NodeID, req.Battery, req.CPUUsage, req.IsCharging)
}

func (m *Matchmaker) DeregisterNode(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	return m.redis.RemoveNode(nodeID)
}

func (m *Matchmaker) GetNodeStatus(nodeID string) (*models.Node, error) {
	return m.redis.GetNode(nodeID)
}

func (m *Matchmaker) GetSessionNode(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}
	return m.redis.GetSessionNode(sessionID)
}

func (m *Matchmaker) SetSessionNode(sessionID, nodeID string, ttlSeconds int) error {
	if sessionID == "" || nodeID == "" {
		return fmt.Errorf("session_id and node_id are required")
	}
	return m.redis.SetSessionNode(sessionID, nodeID, ttlSeconds)
}

func (m *Matchmaker) DecrementNodeLoad(nodeID string) error {
	return m.redis.DecrementNodeLoad(nodeID)
}
