package gateway

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"proxymesh/matchmaker"
)

type ScalingAction string

const (
	ActionScaleUp   ScalingAction = "scale_up"
	ActionScaleDown ScalingAction = "scale_down"
	ActionNone      ScalingAction = "none"
)

type ScalingPolicy struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Metric      string    `json:"metric"`
	Threshold   float64   `json:"threshold"`
	Comparison  string    `json:"comparison"`
	Action      ScalingAction `json:"action"`
	Cooldown    time.Duration `json:"cooldown"`
	LastTriggered time.Time `json:"last_triggered"`
}

type ScalingPolicyManager struct {
	policies map[string]*ScalingPolicy
	mu       sync.RWMutex
}

func NewScalingPolicyManager() *ScalingPolicyManager {
	spm := &ScalingPolicyManager{
		policies: make(map[string]*ScalingPolicy),
	}
	spm.AddPolicy(&ScalingPolicy{
		Name:       "Critical Nodes Alert",
		Enabled:    true,
		Metric:     "critical_nodes",
		Threshold:  0,
		Comparison: ">",
		Action:     ActionScaleUp,
		Cooldown:    5 * time.Minute,
	})
	spm.AddPolicy(&ScalingPolicy{
		Name:       "High Average Utilization",
		Enabled:    true,
		Metric:     "avg_utilization",
		Threshold:  80,
		Comparison: ">",
		Action:     ActionScaleUp,
		Cooldown:    10 * time.Minute,
	})
	return spm
}

func (spm *ScalingPolicyManager) AddPolicy(policy *ScalingPolicy) {
	spm.mu.Lock()
	defer spm.mu.Unlock()
	if policy.ID == "" {
		policy.ID = generatePolicyID()
	}
	spm.policies[policy.ID] = policy
}

func (spm *ScalingPolicyManager) GetPolicy(id string) *ScalingPolicy {
	spm.mu.RLock()
	defer spm.mu.RUnlock()
	return spm.policies[id]
}

func (spm *ScalingPolicyManager) ListPolicies() []*ScalingPolicy {
	spm.mu.RLock()
	defer spm.mu.RUnlock()
	var policies []*ScalingPolicy
	for _, p := range spm.policies {
		policies = append(policies, p)
	}
	return policies
}

func (spm *ScalingPolicyManager) Evaluate(mm *matchmaker.Matchmaker) []ScalingEvaluation {
	spm.mu.RLock()
	defer spm.mu.RUnlock()

	var evaluations []ScalingEvaluation
	for _, policy := range spm.policies {
		if !policy.Enabled {
			continue
		}
		if time.Since(policy.LastTriggered) < policy.Cooldown {
			continue
		}
		eval := spm.evaluatePolicy(policy, mm)
		if eval.Triggered {
			policy.LastTriggered = time.Now()
		}
		evaluations = append(evaluations, eval)
	}
	return evaluations
}

func (spm *ScalingPolicyManager) evaluatePolicy(policy *ScalingPolicy, mm *matchmaker.Matchmaker) ScalingEvaluation {
	eval := ScalingEvaluation{
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
		Triggered:  false,
	}

	switch policy.Metric {
	case "critical_nodes":
		report, err := mm.GetCapacityReport()
		if err != nil {
			return eval
		}
		value := float64(report.CriticalNodes)
		eval.CurrentValue = value
		eval.Threshold = policy.Threshold
		if policy.Comparison == ">" && value > policy.Threshold {
			eval.Triggered = true
			eval.Action = policy.Action
		}

	case "avg_utilization":
		report, err := mm.GetCapacityReport()
		if err != nil {
			return eval
		}
		var totalUtil float64
		for _, node := range report.Nodes {
			totalUtil += node.UtilizationPct
		}
		if len(report.Nodes) > 0 {
			eval.CurrentValue = totalUtil / float64(len(report.Nodes))
			eval.Threshold = policy.Threshold
			if policy.Comparison == ">" && eval.CurrentValue > policy.Threshold {
				eval.Triggered = true
				eval.Action = policy.Action
			}
		}
	}

	return eval
}

type ScalingEvaluation struct {
	PolicyID      string        `json:"policy_id"`
	PolicyName    string        `json:"policy_name"`
	Triggered     bool          `json:"triggered"`
	Action        ScalingAction `json:"action"`
	CurrentValue  float64       `json:"current_value"`
	Threshold     float64       `json:"threshold"`
	Timestamp     time.Time     `json:"timestamp"`
}

func (spm *ScalingPolicyManager) StartEvaluationLoop(mm *matchmaker.Matchmaker, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		evaluations := spm.Evaluate(mm)
		for _, eval := range evaluations {
			if eval.Triggered {
				log.Printf("Scaling policy triggered: %s - Action: %s", eval.PolicyName, eval.Action)
			}
		}
	}
}

var policyCounter int

func generatePolicyID() string {
	policyCounter++
	return "pol-" + time.Now().Format("20060102150405") + "-" + string(rune(policyCounter))
}

func (spm *ScalingPolicyManager) MarshalPolicies() ([]byte, error) {
	spm.mu.RLock()
	defer spm.mu.RUnlock()
	return json.Marshal(spm.policies)
}
