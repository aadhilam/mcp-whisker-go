package whisker

import (
	"context"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewPolicyAnalyzer(t *testing.T) {
	analyzer := NewPolicyAnalyzer("/path/to/kubeconfig")

	if analyzer == nil {
		t.Fatal("Expected PolicyAnalyzer to be created, got nil")
	}

	if analyzer.kubeconfigPath != "/path/to/kubeconfig" {
		t.Errorf("Expected kubeconfigPath to be /path/to/kubeconfig, got %s", analyzer.kubeconfigPath)
	}
}

func TestPolicyAnalyzer_MapPolicyKindToResource(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")

	tests := []struct {
		input    string
		expected string
	}{
		{"CalicoNetworkPolicy", "caliconetworkpolicy"},
		{"NetworkPolicy", "networkpolicy"},
		{"GlobalNetworkPolicy", "globalnetworkpolicy"},
		{"UnknownPolicy", ""},
	}

	for _, test := range tests {
		result := analyzer.MapPolicyKindToResource(test.input)
		if result != test.expected {
			t.Errorf("MapPolicyKindToResource(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestPolicyAnalyzer_GetBlockingReason(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")

	tests := []struct {
		input    string
		expected string
	}{
		{"Deny", "Explicit deny rule"},
		{"Allow", "End of tier default deny"},
		{"Other", "End of tier default deny"},
	}

	for _, test := range tests {
		result := analyzer.GetBlockingReason(test.input)
		if result != test.expected {
			t.Errorf("GetBlockingReason(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestPolicyAnalyzer_GenerateRecommendation(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")

	// Test with blocking policies
	blockingPolicies := []types.BlockingPolicy{
		{BlockingReason: "Test reason"},
	}

	result := analyzer.GenerateRecommendation(blockingPolicies)
	expected := "Review the identified policies to understand why traffic is being blocked. Consider modifying the policy rules if this traffic should be allowed."

	if result != expected {
		t.Errorf("GenerateRecommendation with policies = %s, expected %s", result, expected)
	}

	// Test without blocking policies
	emptyPolicies := []types.BlockingPolicy{}
	result = analyzer.GenerateRecommendation(emptyPolicies)
	expected = "No specific blocking policies identified. This may be due to default deny behavior or policy ordering."

	if result != expected {
		t.Errorf("GenerateRecommendation without policies = %s, expected %s", result, expected)
	}
}

func TestPolicyAnalyzer_ConvertPolicyToDetail(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")

	// Test simple policy without trigger
	policy := &types.Policy{
		Name:        "test-policy",
		Namespace:   "test-ns",
		Kind:        "CalicoNetworkPolicy",
		Tier:        "default",
		Action:      "Allow",
		PolicyIndex: 1,
		RuleIndex:   0,
	}

	detail := analyzer.ConvertPolicyToDetail(policy)

	if detail.Name != "test-policy" {
		t.Errorf("Expected name to be test-policy, got %s", detail.Name)
	}

	if detail.Trigger != nil {
		t.Error("Expected trigger to be nil for policy without trigger")
	}

	// Test policy with trigger
	policyWithTrigger := &types.Policy{
		Name:        "staged-policy",
		Namespace:   "test-ns",
		Kind:        "CalicoNetworkPolicy",
		Tier:        "security",
		Action:      "Pass",
		PolicyIndex: 0,
		RuleIndex:   0,
		Trigger: &types.Policy{
			Name:        "trigger-policy",
			Namespace:   "test-ns",
			Kind:        "CalicoNetworkPolicy",
			Tier:        "default",
			Action:      "Deny",
			PolicyIndex: 1,
			RuleIndex:   0,
		},
	}

	detailWithTrigger := analyzer.ConvertPolicyToDetail(policyWithTrigger)

	if detailWithTrigger.Trigger == nil {
		t.Fatal("Expected trigger to be set")
	}

	if detailWithTrigger.Trigger.Name != "trigger-policy" {
		t.Errorf("Expected trigger name to be trigger-policy, got %s", detailWithTrigger.Trigger.Name)
	}
}

func TestPolicyAnalyzer_AggregatePolicies(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")

	enforcedPolicies := []types.PolicyDetail{}
	pendingPolicies := []types.PolicyDetail{}
	sourcePolicies := make(map[string]bool)
	destPolicies := make(map[string]bool)

	log := &types.FlowLog{
		Reporter: "Src",
		Policies: types.Policies{
			Enforced: []types.Policy{
				{
					Name:      "allow-dns",
					Namespace: "default",
					Kind:      "CalicoNetworkPolicy",
					Action:    "Allow",
				},
			},
			Pending: []types.Policy{
				{
					Name:      "staged-deny",
					Namespace: "security",
					Kind:      "CalicoNetworkPolicy",
					Action:    "Deny",
				},
			},
		},
	}

	analyzer.AggregatePolicies(&enforcedPolicies, &pendingPolicies, sourcePolicies, destPolicies, log)

	if len(enforcedPolicies) != 1 {
		t.Errorf("Expected 1 enforced policy, got %d", len(enforcedPolicies))
	}

	if len(pendingPolicies) != 1 {
		t.Errorf("Expected 1 pending policy, got %d", len(pendingPolicies))
	}

	if len(sourcePolicies) != 1 {
		t.Errorf("Expected 1 source policy, got %d", len(sourcePolicies))
	}

	expectedPolicyName := "allow-dns (default)"
	if !sourcePolicies[expectedPolicyName] {
		t.Errorf("Expected source policy %s to be set", expectedPolicyName)
	}
}

func TestPolicyAnalyzer_ExtractBlockingPolicies(t *testing.T) {
	analyzer := NewPolicyAnalyzer("")
	ctx := context.Background()

	// Test with pending policy that has deny action
	log := &types.FlowLog{
		Policies: types.Policies{
			Pending: []types.Policy{
				{
					Name:      "staged-deny",
					Namespace: "security",
					Kind:      "CalicoNetworkPolicy",
					Action:    "Deny",
				},
			},
		},
	}

	blockingPolicies := analyzer.ExtractBlockingPolicies(ctx, log)

	if len(blockingPolicies) != 1 {
		t.Errorf("Expected 1 blocking policy, got %d", len(blockingPolicies))
	}

	if blockingPolicies[0].BlockingReason != "Explicit deny rule" {
		t.Errorf("Expected blocking reason 'Explicit deny rule', got %s", blockingPolicies[0].BlockingReason)
	}
}
