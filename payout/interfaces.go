package payout

import (
	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"time"
)

// RedisClientInterface defines the methods needed by PayoutService
// This allows mocking for tests
type RedisClientInterface interface {
	GetBandwidth(nodeID string, period time.Time) (*models.BandwidthData, error)
	RecordPayout(nodeID string, payout *Payout) error
	GetAllNodes() ([]string, error)
	GetPayoutHistory(nodeID string, count int64) ([]map[string]interface{}, error)
	GetBandwidthHistory(nodeID string) (map[string]models.BandwidthData, error)
}

// PayoutServiceInterface defines the public interface for PayoutService
type PayoutServiceInterface interface {
	CalculatePayout(nodeID string, period time.Time) (*Payout, error)
	CalculateAllPayouts(period time.Time) ([]Payout, error)
	GetPayoutHistory(nodeID string, limit int) ([]map[string]interface{}, error)
	GetBandwidthBreakdown(nodeID string) (map[string]models.BandwidthData, error)
	SetTiers(tiers []config.PricingTier)
	GetTiers() []config.PricingTier
	SetRates(rates PayoutRates)
	GetRates() PayoutRates
	findTier(totalGB float64) string
	getTierRates(tierName string) (float64, float64)
}