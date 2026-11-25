package web

import (
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
	"time"
)

// MockMatchmaker implements the matchmaker interface for testing
type MockMatchmaker struct {
	nodes                []string
	nodeStatuses         map[string]*models.Node
	nodeLoads            map[string]int64
	cooldownEntries      map[string][]string
	healthPeers          []map[string]interface{}
	healthScores         map[string]*matchmaker.PeerHealth
	healthHistories      map[string][]float64
	resetCircuitBreakerCalls []string
	deregisterNodeCalls  []string
	deregisterNodeError  error
	mockRedis            *MockRedisClient
}

func NewMockMatchmaker() *MockMatchmaker {
	mockRedis := NewMockRedisClient()
	return &MockMatchmaker{
		nodes:                []string{"node-1", "node-2", "node-3"},
		nodeStatuses:         make(map[string]*models.Node),
		nodeLoads:            make(map[string]int64),
		cooldownEntries:      make(map[string][]string),
		healthScores:         make(map[string]*matchmaker.PeerHealth),
		healthHistories:      make(map[string][]float64),
		mockRedis:            mockRedis,
	}
}

func NewMockMatchmakerNoRedis() *MockMatchmaker {
	return &MockMatchmaker{
		nodes:                []string{"node-1", "node-2", "node-3"},
		nodeStatuses:         make(map[string]*models.Node),
		nodeLoads:            make(map[string]int64),
		cooldownEntries:      make(map[string][]string),
		healthScores:         make(map[string]*matchmaker.PeerHealth),
		healthHistories:      make(map[string][]float64),
		mockRedis:            nil,
	}
}

func (m *MockMatchmaker) GetAllNodes() ([]string, error) {
	return m.nodes, nil
}

func (m *MockMatchmaker) GetNodeStatus(nodeID string) (*models.Node, error) {
	if status, ok := m.nodeStatuses[nodeID]; ok {
		return status, nil
	}
	// Return a default node
	return &models.Node{
		ID:         nodeID,
		Country:    "US",
		City:       "New York",
		ISP:        "Test ISP",
		OS:         "Linux",
		Reputation: 100.0,
		LastSeen:   time.Now(),
	}, nil
}

func (m *MockMatchmaker) GetRedis() matchmaker.RedisClientInterface {
	if m.mockRedis == nil {
		return nil
	}
	return m.mockRedis
}

func (m *MockMatchmaker) GetCooldownEntries() (map[string][]string, error) {
	return m.cooldownEntries, nil
}

func (m *MockMatchmaker) GetTopHealthPeers(count int) []*matchmaker.PeerHealth {
	if m.healthPeers != nil {
		// Convert []map[string]interface{} to []*PeerHealth
		result := make([]*matchmaker.PeerHealth, len(m.healthPeers))
		for i, p := range m.healthPeers {
			result[i] = &matchmaker.PeerHealth{
				NodeID:       p["node_id"].(string),
				OverallScore: p["score"].(float64),
				LastChecked:  time.Now(),
			}
		}
		return result
	}
	return []*matchmaker.PeerHealth{
		{NodeID: "node-1", OverallScore: 95.0, LastChecked: time.Now()},
		{NodeID: "node-2", OverallScore: 85.0, LastChecked: time.Now()},
	}
}

func (m *MockMatchmaker) GetHealthScore(nodeID string) *matchmaker.PeerHealth {
	if score, ok := m.healthScores[nodeID]; ok {
		return score
	}
	return nil
}

func (m *MockMatchmaker) GetHealthScoreHistory(nodeID string) []matchmaker.HealthScorePoint {
	if history, ok := m.healthHistories[nodeID]; ok {
		// Convert []float64 to []HealthScorePoint
		result := make([]matchmaker.HealthScorePoint, len(history))
		for i, score := range history {
			result[i] = matchmaker.HealthScorePoint{
				OverallScore: score,
				Timestamp:    time.Now(),
			}
		}
		return result
	}
	return []matchmaker.HealthScorePoint{
		{OverallScore: 90.0, Timestamp: time.Now()},
		{OverallScore: 92.0, Timestamp: time.Now()},
		{OverallScore: 94.0, Timestamp: time.Now()},
		{OverallScore: 95.0, Timestamp: time.Now()},
	}
}

func (m *MockMatchmaker) ResetCircuitBreaker(nodeID string) {
	m.resetCircuitBreakerCalls = append(m.resetCircuitBreakerCalls, nodeID)
}

func (m *MockMatchmaker) DeregisterNode(nodeID string) error {
	m.deregisterNodeCalls = append(m.deregisterNodeCalls, nodeID)
	return m.deregisterNodeError
}

func (m *MockMatchmaker) SetNodeStatus(nodeID string, node *models.Node) {
	m.nodeStatuses[nodeID] = node
}

func (m *MockMatchmaker) SetCooldownEntries(entries map[string][]string) {
	m.cooldownEntries = entries
}

func (m *MockMatchmaker) SetHealthPeers(peers []*matchmaker.PeerHealth) {
	m.healthPeers = make([]map[string]interface{}, len(peers))
	for i, p := range peers {
		m.healthPeers[i] = map[string]interface{}{
			"node_id": p.NodeID,
			"score":   p.OverallScore,
		}
	}
}

func (m *MockMatchmaker) SetHealthScore(nodeID string, score *matchmaker.PeerHealth) {
	m.healthScores[nodeID] = score
}

func (m *MockMatchmaker) SetHealthHistory(nodeID string, history []float64) {
	m.healthHistories[nodeID] = history
}

func (m *MockMatchmaker) SetDeregisterNodeError(err error) {
	m.deregisterNodeError = err
}

func (m *MockMatchmaker) GetResetCircuitBreakerCalls() []string {
	return m.resetCircuitBreakerCalls
}

func (m *MockMatchmaker) GetDeregisterNodeCalls() []string {
	return m.deregisterNodeCalls
}

func (m *MockMatchmaker) SetNodeLoad(nodeID string, load int64) {
	m.mockRedis.SetNodeLoad(nodeID, load)
}