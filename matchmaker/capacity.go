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

type ScalingRecommendation struct {
	Action      string  `json:"action"`
	Reason      string  `json:"reason"`
	Country     string  `json:"country,omitempty"`
	NodesNeeded int     `json:"nodes_needed"`
	Urgency     string  `json:"urgency"`
	Confidence  float64 `json:"confidence"`
}

type ScalingReport struct {
	Recommendations []ScalingRecommendation `json:"recommendations"`
	TotalLoad       int64                   `json:"total_load"`
	AvgUtilization  float64                 `json:"avg_utilization"`
	GeneratedAt     string                  `json:"generated_at"`
}

type BandwidthSnapshot struct {
	Timestamp     int64   `json:"timestamp"`
	BytesSent     int64   `json:"bytes_sent"`
	BytesReceived int64   `json:"bytes_received"`
	TotalGB       float64 `json:"total_gb"`
}

func NewCapacityPlanner(redis *RedisClient) *CapacityPlanner {
	return &CapacityPlanner{redis: redis}
}

// StoreBandwidthSnapshot records a bandwidth data point for trend analysis.
func (cp *CapacityPlanner) StoreBandwidthSnapshot(nodeID string) error {
	bandwidth, err := cp.redis.GetBandwidth(nodeID, time.Now())
	if err != nil {
		return err
	}

	gbSent := float64(bandwidth.BytesSent) / 1_073_741_824
	gbReceived := float64(bandwidth.BytesReceived) / 1_073_741_824
	totalGB := gbSent + gbReceived

	return cp.redis.StoreBandwidthSnapshot(nodeID, BandwidthSnapshot{
		Timestamp:     time.Now().Unix(),
		BytesSent:     bandwidth.BytesSent,
		BytesReceived: bandwidth.BytesReceived,
		TotalGB:       totalGB,
	})
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

	growthRate := cp.calculateGrowthRate(nodeID)

	status := "healthy"
	predictedFull := ""
	if utilization > 80 {
		status = "critical"
		if growthRate > 0 {
			daysUntilFull := (maxLoadGB - dailyRate) / (dailyRate * growthRate / 100)
			if daysUntilFull > 0 && daysUntilFull < 365 {
				predictedFull = time.Now().Add(time.Duration(daysUntilFull*24) * time.Hour).Format("2006-01-02")
			}
		}
	} else if utilization > 50 {
		status = "warning"
	}

	return &NodeCapacity{
		NodeID:          nodeID,
		BandwidthGBDay:  dailyRate,
		BandwidthGBWeek: weeklyRate,
		GrowthRate:      growthRate,
		CurrentLoad:     load,
		UtilizationPct:  utilization,
		PredictedFull:   predictedFull,
		Status:          status,
	}, nil
}

// calculateGrowthRate computes bandwidth growth from stored snapshots.
func (cp *CapacityPlanner) calculateGrowthRate(nodeID string) float64 {
	if cp.redis == nil {
		return 0
	}
	snapshots, err := cp.redis.GetBandwidthSnapshots(nodeID, 10)
	if err != nil || len(snapshots) < 2 {
		return 0
	}

	first := snapshots[0]
	last := snapshots[len(snapshots)-1]

	if first.TotalGB == 0 || first.Timestamp == last.Timestamp {
		return 0
	}

	growthPct := ((last.TotalGB - first.TotalGB) / first.TotalGB) * 100
	return growthPct
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

// GetScalingRecommendations analyzes capacity and suggests scaling actions.
func (cp *CapacityPlanner) GetScalingRecommendations() (*ScalingReport, error) {
	report, err := cp.GenerateReport()
	if err != nil {
		return nil, err
	}

	var recommendations []ScalingRecommendation
	var totalLoad int64
	var totalUtil float64

	for _, node := range report.Nodes {
		totalLoad += node.CurrentLoad
		totalUtil += node.UtilizationPct
	}

	avgUtil := 0.0
	if len(report.Nodes) > 0 {
		avgUtil = totalUtil / float64(len(report.Nodes))
	}

	if report.CriticalNodes > 0 {
		nodesNeeded := report.CriticalNodes
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "scale_up",
			Reason:      fmt.Sprintf("%d nodes at critical capacity (>80%%)", report.CriticalNodes),
			NodesNeeded: nodesNeeded,
			Urgency:     "high",
			Confidence:  0.95,
		})
	}

	if report.WarningNodes > report.HealthyNodes && len(report.Nodes) > 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "scale_up",
			Reason:      fmt.Sprintf("%d of %d nodes in warning state", report.WarningNodes, len(report.Nodes)),
			NodesNeeded: report.WarningNodes / 2,
			Urgency:     "medium",
			Confidence:  0.75,
		})
	}

	if avgUtil < 20 && len(report.Nodes) > 3 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "scale_down",
			Reason:      fmt.Sprintf("Average utilization %.1f%% - nodes underutilized", avgUtil),
			NodesNeeded: 0,
			Urgency:     "low",
			Confidence:  0.6,
		})
	}

	if avgUtil > 60 && report.CriticalNodes == 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "prepare_scale",
			Reason:      fmt.Sprintf("Average utilization %.1f%% approaching capacity", avgUtil),
			NodesNeeded: 1,
			Urgency:     "low",
			Confidence:  0.5,
		})
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "none",
			Reason:      "Fleet capacity is healthy",
			NodesNeeded: 0,
			Urgency:     "none",
			Confidence:  1.0,
		})
	}

	return &ScalingReport{
		Recommendations: recommendations,
		TotalLoad:       totalLoad,
		AvgUtilization:  avgUtil,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}
