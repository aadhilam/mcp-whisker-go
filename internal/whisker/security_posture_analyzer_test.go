package whisker

import (
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewSecurityPostureAnalyzer(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	if analyzer == nil {
		t.Fatal("Expected non-nil SecurityPostureAnalyzer")
	}
}

func TestCalculateSecurityPosture_EmptyLogs(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	result := analyzer.CalculateSecurityPosture([]types.FlowLog{})

	if result.TotalFlows != 0 {
		t.Errorf("Expected 0 total flows, got %d", result.TotalFlows)
	}

	if result.AllowedFlows != 0 {
		t.Errorf("Expected 0 allowed flows, got %d", result.AllowedFlows)
	}

	if result.DeniedFlows != 0 {
		t.Errorf("Expected 0 denied flows, got %d", result.DeniedFlows)
	}

	if result.AllowedPercentage != 0.0 {
		t.Errorf("Expected 0.0 allowed percentage, got %f", result.AllowedPercentage)
	}

	if result.DeniedPercentage != 0.0 {
		t.Errorf("Expected 0.0 denied percentage, got %f", result.DeniedPercentage)
	}

	if result.ActivePolicies != 0 {
		t.Errorf("Expected 0 active policies, got %d", result.ActivePolicies)
	}

	if result.PendingPolicies != 0 {
		t.Errorf("Expected 0 pending policies, got %d", result.PendingPolicies)
	}
}

func TestCalculateSecurityPosture_Basic(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	logs := []types.FlowLog{
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-egress", Namespace: "default"},
				},
			},
		},
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-egress", Namespace: "default"},
				},
			},
		},
		{
			Action: "Deny",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "deny-ingress", Namespace: "production"},
				},
			},
		},
	}

	result := analyzer.CalculateSecurityPosture(logs)

	if result.TotalFlows != 3 {
		t.Errorf("Expected 3 total flows, got %d", result.TotalFlows)
	}

	if result.AllowedFlows != 2 {
		t.Errorf("Expected 2 allowed flows, got %d", result.AllowedFlows)
	}

	if result.DeniedFlows != 1 {
		t.Errorf("Expected 1 denied flow, got %d", result.DeniedFlows)
	}

	expectedAllowedPercent := (2.0 / 3.0) * 100
	// Use tolerance for floating point comparison
	tolerance := 0.01
	if result.AllowedPercentage < expectedAllowedPercent-tolerance || result.AllowedPercentage > expectedAllowedPercent+tolerance {
		t.Errorf("Expected %.2f%% allowed (±%.2f), got %.2f%%", expectedAllowedPercent, tolerance, result.AllowedPercentage)
	}

	expectedDeniedPercent := (1.0 / 3.0) * 100
	if result.DeniedPercentage < expectedDeniedPercent-tolerance || result.DeniedPercentage > expectedDeniedPercent+tolerance {
		t.Errorf("Expected %.2f%% denied (±%.2f), got %.2f%%", expectedDeniedPercent, tolerance, result.DeniedPercentage)
	}

	if result.ActivePolicies != 2 {
		t.Errorf("Expected 2 active policies, got %d", result.ActivePolicies)
	}

	if len(result.UniquePolicyNames) != 2 {
		t.Errorf("Expected 2 unique policy names, got %d", len(result.UniquePolicyNames))
	}

	// Verify policy names are sorted and formatted correctly
	expectedPolicies := []string{"default.allow-egress", "production.deny-ingress"}
	for i, expected := range expectedPolicies {
		if result.UniquePolicyNames[i] != expected {
			t.Errorf("Policy name %d = %s, want %s", i, result.UniquePolicyNames[i], expected)
		}
	}
}

func TestCalculateSecurityPosture_WithPendingPolicies(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	logs := []types.FlowLog{
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "active-policy", Namespace: "default"},
				},
				Pending: []types.Policy{
					{Name: "staged-policy-1", Namespace: "default"},
					{Name: "staged-policy-2", Namespace: "production"},
				},
			},
		},
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "active-policy", Namespace: "default"},
				},
				Pending: []types.Policy{
					{Name: "staged-policy-1", Namespace: "default"},
				},
			},
		},
	}

	result := analyzer.CalculateSecurityPosture(logs)

	if result.TotalFlows != 2 {
		t.Errorf("Expected 2 total flows, got %d", result.TotalFlows)
	}

	if result.ActivePolicies != 1 {
		t.Errorf("Expected 1 active policy, got %d", result.ActivePolicies)
	}

	if result.PendingPolicies != 2 {
		t.Errorf("Expected 2 pending policies, got %d", result.PendingPolicies)
	}

	if len(result.UniquePendingPolicyNames) != 2 {
		t.Errorf("Expected 2 unique pending policy names, got %d", len(result.UniquePendingPolicyNames))
	}

	// Verify pending policy names are sorted
	expectedPending := []string{"default.staged-policy-1", "production.staged-policy-2"}
	for i, expected := range expectedPending {
		if result.UniquePendingPolicyNames[i] != expected {
			t.Errorf("Pending policy name %d = %s, want %s", i, result.UniquePendingPolicyNames[i], expected)
		}
	}
}

func TestCalculateSecurityPosture_GlobalPolicies(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	logs := []types.FlowLog{
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "global-allow", Namespace: ""}, // No namespace = global policy
					{Name: "namespace-policy", Namespace: "default"},
				},
			},
		},
	}

	result := analyzer.CalculateSecurityPosture(logs)

	if result.ActivePolicies != 2 {
		t.Errorf("Expected 2 active policies, got %d", result.ActivePolicies)
	}

	// Global policies should not have namespace prefix
	if result.UniquePolicyNames[0] != "default.namespace-policy" {
		t.Errorf("Expected 'default.namespace-policy', got '%s'", result.UniquePolicyNames[0])
	}

	if result.UniquePolicyNames[1] != "global-allow" {
		t.Errorf("Expected 'global-allow', got '%s'", result.UniquePolicyNames[1])
	}
}

func TestCalculateSecurityPosture_DuplicatePolicies(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	logs := []types.FlowLog{
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-policy", Namespace: "default"},
					{Name: "allow-policy", Namespace: "default"}, // Duplicate
				},
			},
		},
		{
			Action: "Allow",
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-policy", Namespace: "default"}, // Duplicate across logs
				},
			},
		},
	}

	result := analyzer.CalculateSecurityPosture(logs)

	// Should only count unique policies
	if result.ActivePolicies != 1 {
		t.Errorf("Expected 1 unique policy, got %d", result.ActivePolicies)
	}

	if len(result.UniquePolicyNames) != 1 {
		t.Errorf("Expected 1 unique policy name, got %d", len(result.UniquePolicyNames))
	}
}

func TestCalculateSecurityPosture_PercentageCalculation(t *testing.T) {
	analyzer := NewSecurityPostureAnalyzer()

	logs := []types.FlowLog{
		{Action: "Allow"},
		{Action: "Allow"},
		{Action: "Allow"},
		{Action: "Deny"},
	}

	result := analyzer.CalculateSecurityPosture(logs)

	expectedAllowed := 75.0
	if result.AllowedPercentage != expectedAllowed {
		t.Errorf("Expected %.2f%% allowed, got %.2f%%", expectedAllowed, result.AllowedPercentage)
	}

	expectedDenied := 25.0
	if result.DeniedPercentage != expectedDenied {
		t.Errorf("Expected %.2f%% denied, got %.2f%%", expectedDenied, result.DeniedPercentage)
	}

	// Percentages should add up to 100%
	total := result.AllowedPercentage + result.DeniedPercentage
	if total != 100.0 {
		t.Errorf("Percentages should add up to 100%%, got %.2f%%", total)
	}
}
