package whisker

import (
	"strings"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// TestConvertPolicyToDetail tests the conversion of Policy to PolicyDetail with trigger preservation
func TestConvertPolicyToDetail(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name     string
		policy   *types.Policy
		expected types.PolicyDetail
	}{
		{
			name: "Simple policy without trigger",
			policy: &types.Policy{
				Name:        "allow-policy",
				Namespace:   "default",
				Kind:        "NetworkPolicy",
				Tier:        "platform",
				Action:      "Allow",
				PolicyIndex: 0,
				RuleIndex:   0,
			},
			expected: types.PolicyDetail{
				Name:        "allow-policy",
				Namespace:   "default",
				Kind:        "NetworkPolicy",
				Tier:        "platform",
				Action:      "Allow",
				PolicyIndex: 0,
				RuleIndex:   0,
				Trigger:     nil,
			},
		},
		{
			name: "Policy with trigger",
			policy: &types.Policy{
				Name:        "end-of-tier-deny",
				Namespace:   "default",
				Kind:        "EndOfTier",
				Tier:        "default",
				Action:      "Deny",
				PolicyIndex: 0,
				RuleIndex:   -1,
				Trigger: &types.Policy{
					Name:        "summary-policy",
					Namespace:   "yaobank",
					Kind:        "StagedNetworkPolicy",
					Tier:        "default",
					Action:      "ActionUnspecified",
					PolicyIndex: 0,
					RuleIndex:   0,
				},
			},
			expected: types.PolicyDetail{
				Name:        "end-of-tier-deny",
				Namespace:   "default",
				Kind:        "EndOfTier",
				Tier:        "default",
				Action:      "Deny",
				PolicyIndex: 0,
				RuleIndex:   -1,
				Trigger: &types.PolicyDetail{
					Name:        "summary-policy",
					Namespace:   "yaobank",
					Kind:        "StagedNetworkPolicy",
					Tier:        "default",
					Action:      "ActionUnspecified",
					PolicyIndex: 0,
					RuleIndex:   0,
					Trigger:     nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.convertPolicyToDetail(tt.policy)

			// Compare main fields
			if result.Name != tt.expected.Name {
				t.Errorf("Name = %v, want %v", result.Name, tt.expected.Name)
			}
			if result.Namespace != tt.expected.Namespace {
				t.Errorf("Namespace = %v, want %v", result.Namespace, tt.expected.Namespace)
			}
			if result.Action != tt.expected.Action {
				t.Errorf("Action = %v, want %v", result.Action, tt.expected.Action)
			}

			// Compare trigger
			if (result.Trigger == nil) != (tt.expected.Trigger == nil) {
				t.Errorf("Trigger presence mismatch: got %v, want %v", result.Trigger != nil, tt.expected.Trigger != nil)
			}

			if result.Trigger != nil && tt.expected.Trigger != nil {
				if result.Trigger.Name != tt.expected.Trigger.Name {
					t.Errorf("Trigger.Name = %v, want %v", result.Trigger.Name, tt.expected.Trigger.Name)
				}
				if result.Trigger.Kind != tt.expected.Trigger.Kind {
					t.Errorf("Trigger.Kind = %v, want %v", result.Trigger.Kind, tt.expected.Trigger.Kind)
				}
			}
		})
	}
}

// TestAggregatePolicies_WithPendingPolicies tests that pending policies are correctly aggregated
func TestAggregatePolicies_WithPendingPolicies(t *testing.T) {
	service := &Service{}

	flow := &aggregatedFlow{
		sourcePolicies:   make(map[string]bool),
		destPolicies:     make(map[string]bool),
		enforcedPolicies: []types.PolicyDetail{},
		pendingPolicies:  []types.PolicyDetail{},
	}

	log := &types.FlowLog{
		Reporter: "Src",
		Policies: types.Policies{
			Enforced: []types.Policy{
				{
					Name:      "allow-policy",
					Namespace: "default",
					Kind:      "NetworkPolicy",
					Action:    "Allow",
				},
			},
			Pending: []types.Policy{
				{
					Name:      "pending-pass-policy",
					Namespace: "platform",
					Kind:      "GlobalNetworkPolicy",
					Action:    "Pass",
				},
				{
					Name:      "another-pending",
					Namespace: "security",
					Kind:      "CalicoNetworkPolicy",
					Action:    "Pass",
				},
			},
		},
	}

	service.aggregatePolicies(flow, log)

	// Verify enforced policies were added
	if len(flow.enforcedPolicies) != 1 {
		t.Errorf("Expected 1 enforced policy, got %d", len(flow.enforcedPolicies))
	}

	if flow.enforcedPolicies[0].Name != "allow-policy" {
		t.Errorf("Expected enforced policy name 'allow-policy', got '%s'", flow.enforcedPolicies[0].Name)
	}

	// Verify pending policies were added
	if len(flow.pendingPolicies) != 2 {
		t.Errorf("Expected 2 pending policies, got %d", len(flow.pendingPolicies))
	}

	expectedPendingNames := map[string]bool{
		"pending-pass-policy": false,
		"another-pending":     false,
	}

	for _, policy := range flow.pendingPolicies {
		if _, exists := expectedPendingNames[policy.Name]; exists {
			expectedPendingNames[policy.Name] = true
		}
	}

	for name, found := range expectedPendingNames {
		if !found {
			t.Errorf("Expected pending policy '%s' not found", name)
		}
	}
}

// TestConvertToFlowSummary_WithPendingPolicies tests that pending policies are included in flow summary
func TestConvertToFlowSummary_WithPendingPolicies(t *testing.T) {
	service := &Service{}

	flow := &aggregatedFlow{
		source:          "pod-a",
		sourceNamespace: "default",
		destination:     "pod-b",
		destNamespace:   "default",
		protocol:        "TCP",
		port:            443,
		sourceAction:    "Allow",
		destAction:      "Allow",
		packetsIn:       100,
		packetsOut:      50,
		bytesIn:         10000,
		bytesOut:        5000,
		startTime:       "2025-10-23T10:00:00Z",
		endTime:         "2025-10-23T10:01:00Z",
		sourcePolicies:  make(map[string]bool),
		destPolicies:    make(map[string]bool),
		enforcedPolicies: []types.PolicyDetail{
			{Name: "enforced-policy", Namespace: "default", Action: "Allow"},
		},
		pendingPolicies: []types.PolicyDetail{
			{Name: "pending-policy-1", Namespace: "platform", Action: "Pass"},
			{Name: "pending-policy-2", Namespace: "security", Action: "Pass"},
		},
	}

	summary := service.convertToFlowSummary(flow)

	// Verify enforced policies count
	if summary.Enforcement.TotalPolicies != 1 {
		t.Errorf("Expected 1 enforced policy, got %d", summary.Enforcement.TotalPolicies)
	}

	// Verify pending policies count
	if summary.Enforcement.TotalPendingPolicies != 2 {
		t.Errorf("Expected 2 pending policies, got %d", summary.Enforcement.TotalPendingPolicies)
	}

	// Verify pending policy details
	if len(summary.Enforcement.PendingPolicyDetails) != 2 {
		t.Errorf("Expected 2 pending policy details, got %d", len(summary.Enforcement.PendingPolicyDetails))
	}

	// Verify pending policies are marked with emoji
	if len(summary.Source.PendingPolicies) != 2 {
		t.Errorf("Expected 2 pending policies in source endpoint, got %d", len(summary.Source.PendingPolicies))
	}

	// Check that pending policies have the hourglass emoji
	for _, pendingPolicyName := range summary.Source.PendingPolicies {
		if !strings.HasPrefix(pendingPolicyName, "⏳") {
			t.Errorf("Expected pending policy to start with ⏳ emoji, got: %s", pendingPolicyName)
		}
	}
}
