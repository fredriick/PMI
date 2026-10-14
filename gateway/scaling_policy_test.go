package gateway

import (
	"testing"
	"time"
)

func TestScalingPolicyManager_New(t *testing.T) {
	spm := NewScalingPolicyManager()
	if spm == nil {
		t.Fatal("NewScalingPolicyManager returned nil")
	}
	if spm.policies == nil {
		t.Error("expected policies map to be initialized")
	}

	policies := spm.ListPolicies()
	if len(policies) != 2 {
		t.Errorf("expected 2 default policies, got %d", len(policies))
	}
}

func TestScalingPolicyManager_AddPolicy_SP(t *testing.T) {
	spm := NewScalingPolicyManager()

	policy := &ScalingPolicy{
		Name:       "Test Policy",
		Enabled:    true,
		Metric:     "critical_nodes",
		Threshold:  5,
		Comparison: ">",
		Action:     ActionScaleUp,
		Cooldown:   1 * time.Minute,
	}

	spm.AddPolicy(policy)

	if policy.ID == "" {
		t.Error("expected policy ID to be generated")
	}

	got := spm.GetPolicy(policy.ID)
	if got == nil {
		t.Fatal("expected to retrieve added policy")
	}
	if got.Name != "Test Policy" {
		t.Errorf("expected policy name Test Policy, got %q", got.Name)
	}
}

func TestScalingPolicyManager_AddPolicy_EmptyID(t *testing.T) {
	spm := NewScalingPolicyManager()

	policy := &ScalingPolicy{
		ID:         "",
		Name:       "No ID Policy",
		Enabled:    true,
		Metric:     "critical_nodes",
		Threshold:  0,
		Comparison: ">",
		Action:     ActionScaleUp,
		Cooldown:   1 * time.Minute,
	}

	spm.AddPolicy(policy)

	if policy.ID == "" {
		t.Error("expected policy ID to be auto-generated")
	}
}

func TestScalingPolicyManager_GetPolicy_Nonexistent(t *testing.T) {
	spm := NewScalingPolicyManager()

	if spm.GetPolicy("nonexistent") != nil {
		t.Error("expected nil for nonexistent policy")
	}
}

func TestScalingPolicyManager_ListPolicies_SP(t *testing.T) {
	spm := NewScalingPolicyManager()

	policy1 := &ScalingPolicy{Name: "Policy 1", Enabled: true, Metric: "critical_nodes", Threshold: 0, Comparison: ">", Action: ActionScaleUp, Cooldown: 1 * time.Minute}
	policy2 := &ScalingPolicy{Name: "Policy 2", Enabled: true, Metric: "avg_utilization", Threshold: 80, Comparison: ">", Action: ActionScaleUp, Cooldown: 10 * time.Minute}

	spm.AddPolicy(policy1)
	spm.AddPolicy(policy2)

	policies := spm.ListPolicies()
	if len(policies) != 4 {
		t.Errorf("expected 4 policies (2 default + 2 added), got %d", len(policies))
	}
}

func TestScalingPolicy_Disabled(t *testing.T) {
	spm := NewScalingPolicyManager()

	// Disable all policies
	for _, p := range spm.ListPolicies() {
		p.Enabled = false
	}

	// Verify all policies are disabled
	for _, p := range spm.ListPolicies() {
		if p.Enabled {
			t.Error("expected all policies to be disabled")
		}
	}
}

func TestScalingEvaluation_Default(t *testing.T) {
	eval := ScalingEvaluation{
		PolicyID:      "pol-1",
		PolicyName:    "Test Policy",
		Triggered:     true,
		Action:        ActionScaleUp,
		CurrentValue:  5.0,
		Threshold:     3.0,
		Timestamp:     time.Now(),
	}

	if eval.PolicyID != "pol-1" {
		t.Errorf("expected PolicyID pol-1, got %q", eval.PolicyID)
	}
	if !eval.Triggered {
		t.Error("expected Triggered true")
	}
	if eval.Action != ActionScaleUp {
		t.Errorf("expected Action scale_up, got %s", eval.Action)
	}
}

func TestScalingAction_Constants(t *testing.T) {
	if ActionScaleUp != "scale_up" {
		t.Errorf("expected ActionScaleUp to be scale_up, got %s", ActionScaleUp)
	}
	if ActionScaleDown != "scale_down" {
		t.Errorf("expected ActionScaleDown to be scale_down, got %s", ActionScaleDown)
	}
	if ActionNone != "none" {
		t.Errorf("expected ActionNone to be none, got %s", ActionNone)
	}
}

func TestGeneratePolicyID(t *testing.T) {
	id1 := generatePolicyID()
	id2 := generatePolicyID()

	if id1 == id2 {
		t.Error("expected different policy IDs")
	}
	if len(id1) == 0 {
		t.Error("expected non-empty policy ID")
	}
}
