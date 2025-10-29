package whisker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// TestExtractBlockingPolicies verifies that blocking policies are properly converted
// to PolicyDetail to avoid circular reference issues during JSON marshaling
func TestExtractBlockingPolicies(t *testing.T) {
	service := NewService("")
	ctx := context.Background()

	tests := []struct {
		name     string
		flowLog  *types.FlowLog
		expected int // expected number of blocking policies
	}{
		{
			name: "Pending policy with trigger",
			flowLog: &types.FlowLog{
				SourceName:      "test-pod",
				SourceNamespace: "default",
				DestName:        "api-server",
				DestNamespace:   "kube-system",
				Protocol:        "TCP",
				DestPort:        443,
				Action:          "Allow",
				Policies: types.Policies{
					Pending: []types.Policy{
						{
							Name:      "staged-policy",
							Namespace: "security",
							Kind:      "CalicoNetworkPolicy",
							Tier:      "security",
							Action:    "Pass",
							Trigger: &types.Policy{
								Name:      "default-deny",
								Namespace: "security",
								Kind:      "CalicoNetworkPolicy",
								Tier:      "security",
								Action:    "Deny",
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Enforced deny policy with trigger",
			flowLog: &types.FlowLog{
				SourceName:      "suspicious-pod",
				SourceNamespace: "untrusted",
				DestName:        "database",
				DestNamespace:   "production",
				Protocol:        "TCP",
				DestPort:        5432,
				Action:          "Deny",
				Policies: types.Policies{
					Enforced: []types.Policy{
						{
							Name:      "block-untrusted",
							Namespace: "production",
							Kind:      "NetworkPolicy",
							Tier:      "security",
							Action:    "Deny",
							Trigger: &types.Policy{
								Name:      "default-allow",
								Namespace: "production",
								Kind:      "NetworkPolicy",
								Tier:      "default",
								Action:    "Allow",
							},
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "Multiple blocking policies",
			flowLog: &types.FlowLog{
				SourceName:      "app-pod",
				SourceNamespace: "staging",
				DestName:        "external-api",
				DestNamespace:   "",
				Protocol:        "TCP",
				DestPort:        443,
				Action:          "Deny",
				Policies: types.Policies{
					Pending: []types.Policy{
						{
							Name:      "egress-staged",
							Namespace: "staging",
							Kind:      "CalicoNetworkPolicy",
							Tier:      "security",
							Action:    "Pass",
							Trigger: &types.Policy{
								Name:   "egress-deny",
								Tier:   "security",
								Action: "Deny",
							},
						},
					},
					Enforced: []types.Policy{
						{
							Name:      "egress-block",
							Namespace: "staging",
							Kind:      "NetworkPolicy",
							Tier:      "security",
							Action:    "Deny",
							Trigger: &types.Policy{
								Name:   "platform-deny",
								Tier:   "platform",
								Action: "Deny",
							},
						},
					},
				},
			},
			expected: 2,
		},
		{
			name: "No trigger policies",
			flowLog: &types.FlowLog{
				SourceName:      "test-pod",
				SourceNamespace: "default",
				DestName:        "service",
				DestNamespace:   "default",
				Protocol:        "TCP",
				DestPort:        80,
				Action:          "Allow",
				Policies: types.Policies{
					Enforced: []types.Policy{
						{
							Name:      "allow-policy",
							Namespace: "default",
							Kind:      "NetworkPolicy",
							Action:    "Allow",
							Trigger:   nil, // No trigger
						},
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockingPolicies := service.extractBlockingPolicies(ctx, tt.flowLog)

			if len(blockingPolicies) != tt.expected {
				t.Errorf("Expected %d blocking policies, got %d", tt.expected, len(blockingPolicies))
			}

			// Verify that each blocking policy can be marshaled to JSON without errors
			for i, bp := range blockingPolicies {
				if bp.TriggerPolicy == nil {
					t.Errorf("BlockingPolicy[%d].TriggerPolicy is nil", i)
					continue
				}

				// Verify the trigger policy is a PolicyDetail (not Policy)
				if bp.TriggerPolicy.Name == "" {
					t.Errorf("BlockingPolicy[%d].TriggerPolicy.Name is empty", i)
				}

				// Test JSON marshaling to ensure no circular references
				jsonData, err := json.Marshal(bp)
				if err != nil {
					t.Errorf("Failed to marshal BlockingPolicy[%d] to JSON: %v", i, err)
					continue
				}

				// Verify we can unmarshal it back
				var unmarshaled types.BlockingPolicy
				if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
					t.Errorf("Failed to unmarshal BlockingPolicy[%d] from JSON: %v", i, err)
					continue
				}

				// Verify trigger is preserved after unmarshal
				if unmarshaled.TriggerPolicy == nil {
					t.Errorf("BlockingPolicy[%d].TriggerPolicy is nil after unmarshal", i)
				}
			}
		})
	}
}

// TestAnalyzeBlockedFlowsJSONMarshaling ensures the full BlockedFlowAnalysis
// response can be marshaled to JSON without errors (no circular references)
func TestAnalyzeBlockedFlowsJSONMarshaling(t *testing.T) {
	analysis := &types.BlockedFlowAnalysis{
		Namespace: "test",
		Analysis: types.BlockedFlowAnalysisInfo{
			TotalBlockedFlows:        2,
			UniqueBlockedConnections: 2,
		},
		BlockedFlows: []types.BlockedFlowDetail{
			{
				Flow: types.BlockedFlowInfo{
					Source:      "pod-a (namespace-a)",
					Destination: "pod-b (namespace-b)",
					Protocol:    "TCP",
					Port:        443,
					Action:      "Deny",
					Reporter:    "src",
					TimeRange:   "2025-01-01T00:00:00Z to 2025-01-01T00:01:00Z",
				},
				Traffic: types.TrafficInfo{
					Packets: types.TrafficMetric{In: 10, Out: 0, Total: 10},
					Bytes:   types.TrafficMetric{In: 1500, Out: 0, Total: 1500},
				},
				BlockingPolicies: []types.BlockingPolicy{
					{
						TriggerPolicy: &types.PolicyDetail{
							Name:      "deny-all",
							Namespace: "security",
							Kind:      "CalicoNetworkPolicy",
							Tier:      "security",
							Action:    "Deny",
							Trigger: &types.PolicyDetail{
								Name:   "default-pass",
								Tier:   "platform",
								Action: "Pass",
							},
						},
						BlockingReason: "Explicit deny rule",
					},
				},
				Analysis: types.FlowAnalysis{
					TotalBlockingPolicies: 1,
					Recommendation:        "Review policy",
				},
			},
		},
		SecurityInsights: types.SecurityInsights{
			Message:         "Test insights",
			Recommendations: []string{"Review policies"},
		},
	}

	// Test marshaling to JSON
	jsonData, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal BlockedFlowAnalysis to JSON: %v", err)
	}

	// Verify it's valid JSON by unmarshaling
	var unmarshaled types.BlockedFlowAnalysis
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal BlockedFlowAnalysis from JSON: %v", err)
	}

	// Verify structure is preserved
	if len(unmarshaled.BlockedFlows) != 1 {
		t.Errorf("Expected 1 blocked flow, got %d", len(unmarshaled.BlockedFlows))
	}

	if len(unmarshaled.BlockedFlows[0].BlockingPolicies) != 1 {
		t.Errorf("Expected 1 blocking policy, got %d", len(unmarshaled.BlockedFlows[0].BlockingPolicies))
	}

	bp := unmarshaled.BlockedFlows[0].BlockingPolicies[0]
	if bp.TriggerPolicy == nil {
		t.Error("TriggerPolicy is nil after unmarshal")
	} else {
		if bp.TriggerPolicy.Name != "deny-all" {
			t.Errorf("Expected trigger policy name 'deny-all', got '%s'", bp.TriggerPolicy.Name)
		}

		// Verify nested trigger is preserved
		if bp.TriggerPolicy.Trigger == nil {
			t.Error("Nested trigger is nil after unmarshal")
		} else if bp.TriggerPolicy.Trigger.Name != "default-pass" {
			t.Errorf("Expected nested trigger name 'default-pass', got '%s'", bp.TriggerPolicy.Trigger.Name)
		}
	}

	t.Logf("Successfully marshaled and unmarshaled BlockedFlowAnalysis:\n%s", string(jsonData))
}
