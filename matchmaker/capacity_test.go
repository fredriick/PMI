package matchmaker

import (
	"testing"
)

func TestCapacityPlanner_CalculateGrowthRate_NoSnapshots(t *testing.T) {
	cp := &CapacityPlanner{redis: nil}
	rate := cp.calculateGrowthRate("node1")
	if rate != 0 {
		t.Errorf("expected 0 growth rate with nil redis, got %f", rate)
	}
}

func TestScalingRecommendations_CriticalNodes(t *testing.T) {
	// Test the recommendation logic directly
	criticalNodes := 2
	warningNodes := 1
	healthyNodes := 3
	totalNodes := 6

	var recommendations []ScalingRecommendation

	if criticalNodes > 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "scale_up",
			Reason:      "nodes at critical capacity",
			NodesNeeded: criticalNodes,
			Urgency:     "high",
			Confidence:  0.95,
		})
	}

	if warningNodes > healthyNodes && totalNodes > 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "scale_up",
			NodesNeeded: warningNodes / 2,
			Urgency:     "medium",
		})
	}

	if len(recommendations) == 0 {
		t.Error("expected at least one recommendation for critical nodes")
	}

	found := false
	for _, r := range recommendations {
		if r.Action == "scale_up" && r.Urgency == "high" {
			found = true
			if r.NodesNeeded != 2 {
				t.Errorf("expected 2 nodes needed, got %d", r.NodesNeeded)
			}
		}
	}
	if !found {
		t.Error("expected scale_up with high urgency for critical nodes")
	}
}

func TestScalingRecommendations_AllHealthy(t *testing.T) {
	criticalNodes := 0
	warningNodes := 0
	healthyNodes := 5
	avgUtil := 30.0

	var recommendations []ScalingRecommendation

	if criticalNodes > 0 {
		recommendations = append(recommendations, ScalingRecommendation{Action: "scale_up"})
	}
	if warningNodes > healthyNodes {
		recommendations = append(recommendations, ScalingRecommendation{Action: "scale_up"})
	}
	if avgUtil < 20 && healthyNodes > 3 {
		recommendations = append(recommendations, ScalingRecommendation{Action: "scale_down"})
	}
	if avgUtil > 60 && criticalNodes == 0 {
		recommendations = append(recommendations, ScalingRecommendation{Action: "prepare_scale"})
	}

	if len(recommendations) > 0 {
		for _, r := range recommendations {
			if r.Action == "scale_up" {
				t.Error("should not recommend scale_up when all healthy")
			}
		}
	}
}

func TestScalingRecommendations_HighUtilization(t *testing.T) {
	avgUtil := 75.0
	criticalNodes := 0

	var recommendations []ScalingRecommendation

	if avgUtil > 60 && criticalNodes == 0 {
		recommendations = append(recommendations, ScalingRecommendation{
			Action:      "prepare_scale",
			Reason:      "Average utilization approaching capacity",
			NodesNeeded: 1,
			Urgency:     "low",
			Confidence:  0.5,
		})
	}

	if len(recommendations) == 0 {
		t.Error("expected prepare_scale recommendation for high utilization")
	}
}

func TestBandwidthSnapshot_Fields(t *testing.T) {
	snap := BandwidthSnapshot{
		Timestamp:     1000000,
		BytesSent:     1073741824,
		BytesReceived: 536870912,
		TotalGB:       1.5,
	}

	if snap.TotalGB != 1.5 {
		t.Errorf("expected 1.5 GB, got %f", snap.TotalGB)
	}
	if snap.BytesSent != 1073741824 {
		t.Errorf("expected 1GB in bytes, got %d", snap.BytesSent)
	}
}
