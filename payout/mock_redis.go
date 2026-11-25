package payout

import (
	"proxymesh/internal/models"
	"time"
)

type mockRedisClient struct {
	bandwidth        map[string]models.BandwidthData
	payouts          []map[string]interface{}
	nodes            []string
	bandwidthHistory map[string]map[string]models.BandwidthData
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		bandwidth:        make(map[string]models.BandwidthData),
		payouts:          make([]map[string]interface{}, 0),
		nodes:            []string{"node-1", "node-2", "node-3"},
		bandwidthHistory: make(map[string]map[string]models.BandwidthData),
	}
}

func (m *mockRedisClient) GetBandwidth(nodeID string, period time.Time) (*models.BandwidthData, error) {
	if bw, ok := m.bandwidth[nodeID]; ok {
		return &bw, nil
	}
	return &models.BandwidthData{}, nil
}

func (m *mockRedisClient) RecordPayout(nodeID string, payout *Payout) error {
	m.payouts = append(m.payouts, map[string]interface{}{
		"node_id":     payout.NodeID,
		"period":      payout.Period,
		"amount":      payout.Amount,
		"gb_sent":     payout.GBSent,
		"gb_received": payout.GBReceived,
		"tier":        payout.Tier,
	})
	return nil
}

func (m *mockRedisClient) GetAllNodes() ([]string, error) {
	return m.nodes, nil
}

func (m *mockRedisClient) GetPayoutHistory(nodeID string, count int64) ([]map[string]interface{}, error) {
	return m.payouts, nil
}

func (m *mockRedisClient) GetBandwidthHistory(nodeID string) (map[string]models.BandwidthData, error) {
	if history, ok := m.bandwidthHistory[nodeID]; ok {
		return history, nil
	}
	return make(map[string]models.BandwidthData), nil
}

func (m *mockRedisClient) SetBandwidth(nodeID string, bw models.BandwidthData) {
	m.bandwidth[nodeID] = bw
}

func (m *mockRedisClient) SetNodes(nodes []string) {
	m.nodes = nodes
}

func (m *mockRedisClient) SetBandwidthHistory(nodeID string, history map[string]models.BandwidthData) {
	m.bandwidthHistory[nodeID] = history
}