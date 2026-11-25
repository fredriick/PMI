package matchmaker

import (
	"proxymesh/internal/models"
)

// MatchmakerInterface defines the methods needed by WebUI
// This allows mocking for tests
type MatchmakerInterface interface {
	GetAllNodes() ([]string, error)
	GetNodeStatus(nodeID string) (*models.Node, error)
	GetRedis() RedisClientInterface
	GetCooldownEntries() (map[string][]string, error)
	GetTopHealthPeers(count int) []*PeerHealth
	GetHealthScore(nodeID string) *PeerHealth
	GetHealthScoreHistory(nodeID string) []HealthScorePoint
	ResetCircuitBreaker(nodeID string)
	DeregisterNode(nodeID string) error
}