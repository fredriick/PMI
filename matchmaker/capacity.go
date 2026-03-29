package matchmaker

import (
	"fmt"
	"time"
)

type CapacityPlanner struct {
	redis *RedisClient
}

type NodeCapacity struct {
	NodeID          string  `json:"node_id"`
	BandwidthGBDay  float64 `json:"bandwidth_gb_day"`
	BandwidthGBWeek float64 `json:"bandwidth_gb_week"`
	GrowthRate      float64 `json:"growth_rate_percent"`
	CurrentLoad     int64   `json:"current_load"`
	UtilizationPct  float64 `json:"utilization_percent"`
	PredictedFull   string  `json:"predicted_full_at,omitempty"`
	Status          string  `json:"status"`
}

type CapacityReport struct {
	Nodes         []NodeCapacity `json:"nodes"`
	TotalNodes    int            `json:"total_nodes"`
	HealthyNodes  int            `json:"healthy_nodes"`
	WarningNodes  int            `json:"warning_nodes"`
	CriticalNodes int            `json:"critical_nodes"`
	GeneratedAt   string         `json:"generated_at"`
}

func NewCapacityPlanner(redis *RedisClient) *CapacityPlanner {
	return &CapacityPlanner{redis: redis}
}

// AnalyzeNode calculates capacity metrics for a single node.
func (cp *CapacityPlanner) AnalyzeNode(nodeID string) (*NodeCapacity, error) {
	bandwidth, err := cp.redis.GetBandwidth(nodeID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get bandwidth: %w", err)
	}

	load, _ := cp.redis.GetNodeLoad(nodeID)

	gbSent := float64(bandwidth.BytesSent) / 1_073_741_824
	gbReceived := float64(bandwidth.BytesReceived) / 1_073_741_824
	totalGB := gbSent + gbReceived

	dailyRate := totalGB
	if bandwidth.DurationSeconds > 0 {
		hoursRunning := float64(bandwidth.DurationSeconds) / 3600.0
		if hoursRunning > 0 && hoursRunning < 24 {
			dailyRate = totalGB / hoursRunning * 24
		}
	}

	weeklyRate := dailyRate * 7

	maxLoadGB := 100.0
	utilization := (dailyRate / maxLoadGB) * 100
	if utilization > 100 {
		utilization = 100
	}

	status := "healthy"
	predictedFull := ""
	if utilization > 80 {
		status = "critical"
		daysUntilFull := (maxLoadGB - dailyRate) / (dailyRate * 0.1)
		if daysUntilFull > 0 {
			predictedFull = time.Now().Add(time.Duration(daysUntilFull*24) * time.Hour).Format("2006-01-02")
		}
	} else if utilization > 50 {
		status = "warning"
	}

	return &NodeCapacity{
		NodeID:          nodeID,
		BandwidthGBDay:  dailyRate,
		BandwidthGBWeek: weeklyRate,
		GrowthRate:      0,
		CurrentLoad:     load,
		UtilizationPct:  utilization,
		PredictedFull:   predictedFull,
		Status:          status,
	}, nil
}

// GenerateReport creates a capacity report for all nodes.
func (cp *CapacityPlanner) GenerateReport() (*CapacityReport, error) {
	nodeIDs, err := cp.redis.GetAllNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var nodes []NodeCapacity
	healthy, warning, critical := 0, 0, 0

	for _, nodeID := range nodeIDs {
		capacity, err := cp.AnalyzeNode(nodeID)
		if err != nil {
			continue
		}
		nodes = append(nodes, *capacity)

		switch capacity.Status {
		case "healthy":
			healthy++
		case "warning":
			warning++
		case "critical":
			critical++
		}
	}

	return &CapacityReport{
		Nodes:         nodes,
		TotalNodes:    len(nodes),
		HealthyNodes:  healthy,
		WarningNodes:  warning,
		CriticalNodes: critical,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// GetOverloadedNodes returns nodes that are at or near capacity.
func (cp *CapacityPlanner) GetOverloadedNodes(threshold float64) ([]NodeCapacity, error) {
	report, err := cp.GenerateReport()
	if err != nil {
		return nil, err
	}

	var overloaded []NodeCapacity
	for _, node := range report.Nodes {
		if node.UtilizationPct >= threshold {
			overloaded = append(overloaded, node)
		}
	}
	return overloaded, nil
}
