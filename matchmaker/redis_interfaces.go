package matchmaker

import (
	"github.com/go-redis/redis/v8"
	"proxymesh/internal/models"
	"time"
)

// RedisClientInterface defines the methods needed by WebUI
// This allows mocking for tests
type RedisClientInterface interface {
	GetNodeLoad(nodeID string) (int64, error)
	GetNode(nodeID string) (*models.Node, error)
	GetAllNodes() ([]string, error)
	GetNodeMeta(nodeID string) (*models.NodeMeta, error)
	GetCooldownEntries() (map[string][]string, error)
	Close() error
	Client() *redis.Client
	UpdateNodeStatus(nodeID string, battery int, cpuUsage float64, isCharging bool) error
	RemoveNode(nodeID string) error
	GetSessionNode(sessionID string) (string, error)
	SetSessionNode(sessionID, nodeID string, ttlSeconds int) error
	GetNodeReputation(nodeID string, country string) (float64, error)
	UpdateNodeReputation(nodeID string, country string, score float64) error
	IncrementNodeLoad(nodeID string) error
	DecrementNodeLoad(nodeID string) error
	RecordBandwidth(nodeID string, bytesSent, bytesReceived, durationSeconds int64) error
	GetBandwidth(nodeID string, period time.Time) (*models.BandwidthData, error)
	GetBandwidthHistory(nodeID string) (map[string]models.BandwidthData, error)
	ListSessions() ([]map[string]string, error)
	DeleteSession(sessionID string) error
	StoreBandwidthSnapshot(nodeID string, snapshot interface{}) error
	GetBandwidthSnapshots(nodeID string, count int64) ([]BandwidthSnapshot, error)
	RecordPayout(nodeID string, payout interface{}) error
	GetPayoutHistory(nodeID string, count int64) ([]map[string]interface{}, error)
	RecordReferral(referrerCode, newNodeID string) error
	GetReferrals(referrerCode string) ([]string, error)
	GetReferralCount(referrerCode string) (int64, error)
	AddNode(node *models.Node) error
	GetTopNodes(country string, count int64) ([]string, error)
	GetNodesByCity(country, city string, count int64) ([]string, error)
	AddToCooldown(target string, nodeID string) error
	IsInCooldown(target string, nodeID string) (bool, error)
	AddToCooldownWithTTL(target string, nodeID string, ttl time.Duration) error
	RemoveFromCooldown(target string, nodeID string) error
	SetNodeLoad(nodeID string, load int64) error
}