package payout

import (
	"fmt"
	"time"

	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)

type BandwidthData = models.BandwidthData

type PayoutService struct {
	matchmaker *matchmaker.Matchmaker
	redis      *matchmaker.RedisClient
	rates      PayoutRates
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
	}
}

func (p *PayoutService) CalculatePayout(nodeID string, period time.Time) (*Payout, error) {
	bandwidth, err := p.redis.GetBandwidth(nodeID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get bandwidth: %w", err)
	}

	gbSent := float64(bandwidth.BytesSent) / 1_073_741_824
	gbReceived := float64(bandwidth.BytesReceived) / 1_073_741_824

	amount := (gbSent * p.rates.RatePerGBSent) + (gbReceived * p.rates.RatePerGBReceived)

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
		CalculatedAt:  time.Now(),
	}, nil
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

func (p *PayoutService) GetBandwidthBreakdown(nodeID string) (map[string]models.BandwidthData, error) {
	return p.redis.GetBandwidthHistory(nodeID)
}
