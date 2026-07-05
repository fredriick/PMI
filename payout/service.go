package payout

import (
	"fmt"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)

type BandwidthData = models.BandwidthData

type PayoutService struct {
	matchmaker *matchmaker.Matchmaker
	redis      *matchmaker.RedisClient
	rates      PayoutRates
	tiers      []config.PricingTier
}

type PayoutRates struct {
	RatePerGBSent     float64
	RatePerGBReceived float64
	MinPayoutAmount   float64
}

type Payout struct {
	NodeID        string    `json:"node_id"`
	Period        string    `json:"period"`
	BytesSent     int64     `json:"bytes_sent"`
	BytesReceived int64     `json:"bytes_received"`
	GBSent        float64   `json:"gb_sent"`
	GBReceived    float64   `json:"gb_received"`
	Amount        float64   `json:"amount"`
	Tier          string    `json:"tier,omitempty"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

func NewPayoutService(mm *matchmaker.Matchmaker, redis *matchmaker.RedisClient) *PayoutService {
	return &PayoutService{
		matchmaker: mm,
		redis:      redis,
		rates: PayoutRates{
			RatePerGBSent:     0.50,
			RatePerGBReceived: 0.30,
			MinPayoutAmount:   10.00,
		},
		tiers: []config.PricingTier{
			{Name: "Basic", MinGBMonthly: 0, MaxGBMonthly: 10, RatePerGBSent: 0.30, RatePerGBRecv: 0.15},
			{Name: "Premium", MinGBMonthly: 10, MaxGBMonthly: 100, RatePerGBSent: 0.50, RatePerGBRecv: 0.30},
			{Name: "Enterprise", MinGBMonthly: 100, MaxGBMonthly: 10000, RatePerGBSent: 0.80, RatePerGBRecv: 0.50},
		},
	}
}

func (p *PayoutService) SetTiers(tiers []config.PricingTier) {
	p.tiers = tiers
}

func (p *PayoutService) GetTiers() []config.PricingTier {
	return p.tiers
}

func (p *PayoutService) CalculatePayout(nodeID string, period time.Time) (*Payout, error) {
	payout, err := p.calculatePayoutInternal(nodeID, period)
	if err != nil {
		return nil, err
	}

	if payout.Amount > 0 {
		_ = p.redis.RecordPayout(nodeID, payout)
	}

	return payout, nil
}

func (p *PayoutService) calculatePayoutInternal(nodeID string, period time.Time) (*Payout, error) {
	bandwidth, err := p.redis.GetBandwidth(nodeID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get bandwidth: %w", err)
	}

	gbSent := float64(bandwidth.BytesSent) / 1_073_741_824
	gbReceived := float64(bandwidth.BytesReceived) / 1_073_741_824
	totalGB := gbSent + gbReceived

	tier := p.findTier(totalGB)
	rateSent, rateRecv := p.getTierRates(tier)

	amount := (gbSent * rateSent) + (gbReceived * rateRecv)

	if amount < p.rates.MinPayoutAmount {
		amount = 0
	}

	return &Payout{
		NodeID:        nodeID,
		Period:        period.Format("2006-01"),
		BytesSent:     bandwidth.BytesSent,
		BytesReceived: bandwidth.BytesReceived,
		GBSent:        gbSent,
		GBReceived:    gbReceived,
		Amount:        amount,
		Tier:          tier,
		CalculatedAt:  time.Now(),
	}, nil
}

func (p *PayoutService) findTier(totalGB float64) string {
	for _, tier := range p.tiers {
		if totalGB >= float64(tier.MinGBMonthly) && totalGB < float64(tier.MaxGBMonthly) {
			return tier.Name
		}
	}
	return "Basic"
}

func (p *PayoutService) getTierRates(tierName string) (float64, float64) {
	for _, tier := range p.tiers {
		if tier.Name == tierName {
			return tier.RatePerGBSent, tier.RatePerGBRecv
		}
	}
	return p.rates.RatePerGBSent, p.rates.RatePerGBReceived
}

func (p *PayoutService) CalculateAllPayouts(period time.Time) ([]Payout, error) {
	nodes, err := p.redis.GetAllNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var payouts []Payout
	for _, nodeID := range nodes {
		payout, err := p.CalculatePayout(nodeID, period)
		if err != nil {
			continue
		}
		if payout.Amount > 0 {
			payouts = append(payouts, *payout)
		}
	}

	return payouts, nil
}

func (p *PayoutService) SetRates(rates PayoutRates) {
	p.rates = rates
}

func (p *PayoutService) GetRates() PayoutRates {
	return p.rates
}

func (p *PayoutService) GetPayoutHistory(nodeID string, limit int) ([]map[string]interface{}, error) {
	return p.redis.GetPayoutHistory(nodeID, int64(limit))
}

func (p *PayoutService) GetBandwidthBreakdown(nodeID string) (map[string]models.BandwidthData, error) {
	return p.redis.GetBandwidthHistory(nodeID)
}
