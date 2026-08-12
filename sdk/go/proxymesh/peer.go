package proxymesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PeerClient struct {
	baseURL string
	token   string
	nodeID  string
	client  *http.Client
}

type LoginRequest struct {
	NodeID string `json:"node_id"`
}

type LoginResponse struct {
	Token  string `json:"token"`
	NodeID string `json:"node_id"`
}

type NodeStatus struct {
	Node NodeInfo `json:"node"`
	Load float64  `json:"load,omitempty"`
}

type NodeInfo struct {
	ID         string  `json:"id"`
	Online     *bool   `json:"online,omitempty"`
	Battery    *float64 `json:"battery,omitempty"`
	CPUUsage   *float64 `json:"cpu_usage,omitempty"`
	Country    string  `json:"country,omitempty"`
	City       string  `json:"city,omitempty"`
	IP         string  `json:"ip,omitempty"`
	OS         string  `json:"os,omitempty"`
	NodeType   string  `json:"node_type,omitempty"`
	LastSeen   string  `json:"last_seen,omitempty"`
	Reputation float64 `json:"reputation,omitempty"`
	ISP        string  `json:"isp,omitempty"`
}

type BandwidthData struct {
	Current  BandwidthCurrent            `json:"current"`
	History  map[string]BandwidthSnapshot `json:"history"`
}

type BandwidthCurrent struct {
	BytesSent     int64 `json:"bytes_sent"`
	BytesReceived int64 `json:"bytes_received"`
	DurationSeconds int64 `json:"duration_seconds"`
}

type BandwidthSnapshot struct {
	BytesSent     int64 `json:"bytes_sent"`
	BytesReceived int64 `json:"bytes_received"`
}

type PayoutData struct {
	Payout PayoutInfo            `json:"payout"`
	Rates  map[string]float64   `json:"rates"`
	Tiers  []PayoutTier         `json:"tiers"`
}

type PayoutInfo struct {
	Amount       float64 `json:"amount"`
	Period       string  `json:"period"`
	Tier         string  `json:"tier"`
	GBSent       float64 `json:"gb_sent"`
	GBReceived   float64 `json:"gb_received"`
}

type PayoutTier struct {
	Name         string  `json:"name"`
	RatePerGBSent float64 `json:"rate_per_gb_sent"`
	RatePerGBRecv float64 `json:"rate_per_gb_recv"`
}

type HealthScore struct {
	OverallScore      float64 `json:"overall_score"`
	LatencyScore      float64 `json:"latency_score,omitempty"`
	BandwidthScore    float64 `json:"bandwidth_score,omitempty"`
	ReliabilityScore  float64 `json:"reliability_score,omitempty"`
	ReputationScore   float64 `json:"reputation_score,omitempty"`
}

type TelemetryEvent struct {
	Score      *HealthScore       `json:"score,omitempty"`
	Bandwidth  *BandwidthData     `json:"bandwidth,omitempty"`
	Earnings   *PayoutData        `json:"earnings,omitempty"`
	Node       *NodeInfo          `json:"node,omitempty"`
	Load       float64            `json:"load,omitempty"`
}

func NewPeerClient(baseURL string) *PeerClient {
	return &PeerClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *PeerClient) SetAuth(token, nodeID string) {
	p.token = token
	p.nodeID = nodeID
}

func (p *PeerClient) ClearAuth() {
	p.token = ""
	p.nodeID = ""
}

func (p *PeerClient) request(ctx context.Context, path, method string, body interface{}) ([]byte, error) {
	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, &reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("X-Peer-Token", p.token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := json.Marshal(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

func (p *PeerClient) Login(ctx context.Context, nodeID string) (*LoginResponse, error) {
	req := LoginRequest{NodeID: nodeID}
	data, err := p.request(ctx, "/auth", "POST", req)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(data, &loginResp); err != nil {
		return nil, fmt.Errorf("decode login response: %w", err)
	}

	p.SetAuth(loginResp.Token, loginResp.NodeID)
	return &loginResp, nil
}

func (p *PeerClient) Disconnect(ctx context.Context) error {
	if p.token == "" {
		return nil
	}

	_, err := p.request(ctx, "/disconnect", "POST", map[string]string{})
	if err != nil {
		return err
	}

	p.ClearAuth()
	return nil
}

func (p *PeerClient) GetStatus(ctx context.Context) (*NodeStatus, error) {
	data, err := p.request(ctx, "/status", "GET", nil)
	if err != nil {
		return nil, err
	}

	var status NodeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("decode status response: %w", err)
	}

	return &status, nil
}

func (p *PeerClient) GetBandwidth(ctx context.Context) (*BandwidthData, error) {
	data, err := p.request(ctx, "/bandwidth", "GET", nil)
	if err != nil {
		return nil, err
	}

	var bandwidth BandwidthData
	if err := json.Unmarshal(data, &bandwidth); err != nil {
		return nil, fmt.Errorf("decode bandwidth response: %w", err)
	}

	return &bandwidth, nil
}

func (p *PeerClient) GetEarnings(ctx context.Context) (*PayoutData, error) {
	data, err := p.request(ctx, "/earnings", "GET", nil)
	if err != nil {
		return nil, err
	}

	var earnings PayoutData
	if err := json.Unmarshal(data, &earnings); err != nil {
		return nil, fmt.Errorf("decode earnings response: %w", err)
	}

	return &earnings, nil
}

func (p *PeerClient) GetHealth(ctx context.Context) (*HealthScore, error) {
	data, err := p.request(ctx, "/health", "GET", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Score *HealthScore `json:"score"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode health response: %w", err)
	}

	return resp.Score, nil
}

func (p *PeerClient) SetConsent(ctx context.Context, enabled bool) error {
	_, err := p.request(ctx, "/consent", "POST", map[string]bool{"enabled": enabled})
	return err
}

func (p *PeerClient) UpdateNode(ctx context.Context, updates map[string]interface{}) error {
	_, err := p.request(ctx, "/node", "PATCH", updates)
	return err
}
