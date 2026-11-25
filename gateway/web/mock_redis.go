package web

import (
	"github.com/go-redis/redis/v8"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
	"sync"
	"time"
)

// MockRedisClient implements matchmaker.RedisClient for testing
type MockRedisClient struct {
	mu            sync.RWMutex
	nodeLoads     map[string]int64
	nodeMeta      map[string]*models.Node
	bandwidthData map[string]*models.BandwidthData
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		nodeLoads:     make(map[string]int64),
		nodeMeta:      make(map[string]*models.Node),
		bandwidthData: make(map[string]*models.BandwidthData),
	}
}

func (m *MockRedisClient) GetNodeLoad(nodeID string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if load, ok := m.nodeLoads[nodeID]; ok {
		return load, nil
	}
	return 0, nil
}

func (m *MockRedisClient) GetNode(nodeID string) (*models.Node, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if node, ok := m.nodeMeta[nodeID]; ok {
		return node, nil
	}
	return &models.Node{ID: nodeID}, nil
}

func (m *MockRedisClient) SetNodeLoad(nodeID string, load int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeLoads[nodeID] = load
	return nil
}

func (m *MockRedisClient) SetNodeMeta(nodeID string, node *models.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeMeta[nodeID] = node
}

// Implement other required methods as no-ops
func (m *MockRedisClient) AddNode(node *models.Node) error              { return nil }
func (m *MockRedisClient) GetTopNodes(country string, count int64) ([]string, error) { return nil, nil }
func (m *MockRedisClient) GetNodesByCity(country, city string, count int64) ([]string, error) { return nil, nil }
func (m *MockRedisClient) GetNodeMeta(nodeID string) (*models.NodeMeta, error) { return nil, nil }
func (m *MockRedisClient) AddToCooldown(target string, nodeID string) error { return nil }
func (m *MockRedisClient) IsInCooldown(target string, nodeID string) (bool, error) { return false, nil }
func (m *MockRedisClient) AddToCooldownWithTTL(target string, nodeID string, ttl time.Duration) error { return nil }
func (m *MockRedisClient) RemoveFromCooldown(target string, nodeID string) error { return nil }
func (m *MockRedisClient) GetCooldownEntries() (map[string][]string, error) { return nil, nil }
func (m *MockRedisClient) Close() error { return nil }
func (m *MockRedisClient) Client() *redis.Client { return nil }
func (m *MockRedisClient) UpdateNodeStatus(nodeID string, battery int, cpuUsage float64, isCharging bool) error { return nil }
func (m *MockRedisClient) RemoveNode(nodeID string) error { return nil }
func (m *MockRedisClient) GetSessionNode(sessionID string) (string, error) { return "", nil }
func (m *MockRedisClient) SetSessionNode(sessionID, nodeID string, ttlSeconds int) error { return nil }
func (m *MockRedisClient) GetNodeReputation(nodeID string, country string) (float64, error) { return 0, nil }
func (m *MockRedisClient) UpdateNodeReputation(nodeID string, country string, score float64) error { return nil }
func (m *MockRedisClient) IncrementNodeLoad(nodeID string) error { return nil }
func (m *MockRedisClient) DecrementNodeLoad(nodeID string) error { return nil }
func (m *MockRedisClient) RecordBandwidth(nodeID string, bytesSent, bytesReceived, durationSeconds int64) error { return nil }
func (m *MockRedisClient) GetBandwidth(nodeID string, period time.Time) (*models.BandwidthData, error) { return nil, nil }
func (m *MockRedisClient) GetBandwidthHistory(nodeID string) (map[string]models.BandwidthData, error) { return nil, nil }
func (m *MockRedisClient) GetAllNodes() ([]string, error) { return nil, nil }
func (m *MockRedisClient) ListSessions() ([]map[string]string, error) { return nil, nil }
func (m *MockRedisClient) DeleteSession(sessionID string) error { return nil }
func (m *MockRedisClient) StoreBandwidthSnapshot(nodeID string, snapshot interface{}) error { return nil }
func (m *MockRedisClient) GetBandwidthSnapshots(nodeID string, count int64) ([]matchmaker.BandwidthSnapshot, error) { return nil, nil }
func (m *MockRedisClient) RecordPayout(nodeID string, payout interface{}) error { return nil }
func (m *MockRedisClient) GetPayoutHistory(nodeID string, count int64) ([]map[string]interface{}, error) { return nil, nil }
func (m *MockRedisClient) RecordReferral(referrerCode, newNodeID string) error { return nil }
func (m *MockRedisClient) GetReferrals(referrerCode string) ([]string, error) { return nil, nil }
func (m *MockRedisClient) GetReferralCount(referrerCode string) (int64, error) { return 0, nil }