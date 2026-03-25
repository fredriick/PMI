package models

import "time"

type NodeType string

const (
	NodeTypeDatacenter  NodeType = "datacenter"
	NodeTypeResidential NodeType = "residential"
)

type Node struct {
	ID         string            `json:"id"`
	NodeType   NodeType          `json:"node_type"`
	Country    string            `json:"country"`
	City       string            `json:"city"`
	ISP        string            `json:"isp"`
	IP         string            `json:"ip"`
	IPv6Subnet string            `json:"ipv6_subnet,omitempty"`
	OS         string            `json:"os"`
	Battery    int               `json:"battery,omitempty"`
	IsCharging bool              `json:"is_charging,omitempty"`
	CPUUsage   float64           `json:"cpu_usage,omitempty"`
	LastSeen   time.Time         `json:"last_seen"`
	Reputation float64           `json:"reputation"`
	Metadata   map[string]string `json:"metadata"`
}

type NodeMeta struct {
	NodeID   string    `json:"node_id"`
	ISP      string    `json:"isp"`
	Battery  int       `json:"battery"`
	OS       string    `json:"os"`
	LastSeen time.Time `json:"last_seen"`
}

type ProxyRequest struct {
	User      string `json:"user"`
	Password  string `json:"password"`
	Target    string `json:"target"`
	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type ProxyResponse struct {
	NodeID    string `json:"node_id"`
	Latency   int64  `json:"latency"`
	LocalAddr string `json:"local_addr"`
}

type User struct {
	APIKey      string   `json:"api_key"`
	Username    string   `json:"username"`
	AllowedIPs  []string `json:"allowed_ips"`
	KYCVerified bool     `json:"kyc_verified"`
}

type CircuitBreaker struct {
	FailureCount int
	Threshold    int
	LastFailure  time.Time
	State        string
}
