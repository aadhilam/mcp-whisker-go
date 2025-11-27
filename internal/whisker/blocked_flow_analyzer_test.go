package whisker

import (
	"context"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewBlockedFlowAnalyzer(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	if analyzer == nil {
		t.Fatal("Expected non-nil BlockedFlowAnalyzer")
	}

	if analyzer.policyAnalyzer == nil {
		t.Error("Expected policyAnalyzer to be set")
	}
}

func TestAnalyzeBlockedFlows_Basic(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	blockedLogs := []types.FlowLog{
		{
			SourceName:      "app-1",
			SourceNamespace: "default",
			DestName:        "db-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        5432,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       10,
			PacketsOut:      0,
			BytesIn:         1024,
			BytesOut:        0,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{
						Name: "block-db-access",
						Tier: "security",
						Kind: "NetworkPolicy",
					},
				},
			},
		},
	}

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", blockedLogs)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", result.Namespace)
	}

	if result.Analysis.TotalBlockedFlows != 1 {
		t.Errorf("Expected 1 blocked flow, got %d", result.Analysis.TotalBlockedFlows)
	}

	if result.Analysis.UniqueBlockedConnections != 1 {
		t.Errorf("Expected 1 unique connection, got %d", result.Analysis.UniqueBlockedConnections)
	}

	if len(result.BlockedFlows) != 1 {
		t.Fatalf("Expected 1 blocked flow detail, got %d", len(result.BlockedFlows))
	}

	detail := result.BlockedFlows[0]
	if detail.Flow.Source != "app-1 (default)" {
		t.Errorf("Expected source 'app-1 (default)', got '%s'", detail.Flow.Source)
	}

	if detail.Flow.Destination != "db-1 (production)" {
		t.Errorf("Expected destination 'db-1 (production)', got '%s'", detail.Flow.Destination)
	}

	if detail.Flow.Protocol != "TCP" {
		t.Errorf("Expected protocol 'TCP', got '%s'", detail.Flow.Protocol)
	}

	if detail.Flow.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", detail.Flow.Port)
	}

	if detail.Traffic.Packets.In != 10 {
		t.Errorf("Expected 10 packets in, got %d", detail.Traffic.Packets.In)
	}

	if detail.Traffic.Packets.Total != 10 {
		t.Errorf("Expected 10 total packets, got %d", detail.Traffic.Packets.Total)
	}

	if detail.Traffic.Bytes.In != 1024 {
		t.Errorf("Expected 1024 bytes in, got %d", detail.Traffic.Bytes.In)
	}

	if detail.Traffic.Bytes.Total != 1024 {
		t.Errorf("Expected 1024 total bytes, got %d", detail.Traffic.Bytes.Total)
	}
}

func TestAnalyzeBlockedFlows_MultipleFlows(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	blockedLogs := []types.FlowLog{
		{
			SourceName:      "app-1",
			SourceNamespace: "default",
			DestName:        "db-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        5432,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       10,
			PacketsOut:      0,
			BytesIn:         1024,
			BytesOut:        0,
		},
		{
			SourceName:      "app-2",
			SourceNamespace: "default",
			DestName:        "api-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        443,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       5,
			PacketsOut:      0,
			BytesIn:         512,
			BytesOut:        0,
		},
		{
			SourceName:      "app-1",
			SourceNamespace: "default",
			DestName:        "db-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        5432,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:10Z",
			EndTime:         "2024-11-07T10:00:15Z",
			PacketsIn:       8,
			PacketsOut:      0,
			BytesIn:         800,
			BytesOut:        0,
		},
	}

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", blockedLogs)

	if result.Analysis.TotalBlockedFlows != 3 {
		t.Errorf("Expected 3 blocked flows, got %d", result.Analysis.TotalBlockedFlows)
	}

	// Should have 2 unique connections: app-1→db-1:5432 and app-2→api-1:443
	if result.Analysis.UniqueBlockedConnections != 2 {
		t.Errorf("Expected 2 unique connections, got %d", result.Analysis.UniqueBlockedConnections)
	}

	if len(result.BlockedFlows) != 3 {
		t.Errorf("Expected 3 blocked flow details, got %d", len(result.BlockedFlows))
	}
}

func TestAnalyzeBlockedFlows_EmptyLogs(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", []types.FlowLog{})

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Analysis.TotalBlockedFlows != 0 {
		t.Errorf("Expected 0 blocked flows, got %d", result.Analysis.TotalBlockedFlows)
	}

	if result.Analysis.UniqueBlockedConnections != 0 {
		t.Errorf("Expected 0 unique connections, got %d", result.Analysis.UniqueBlockedConnections)
	}

	if len(result.BlockedFlows) != 0 {
		t.Errorf("Expected 0 blocked flow details, got %d", len(result.BlockedFlows))
	}

	if result.SecurityInsights.Message != "🚨 0 blocked flow(s) detected" {
		t.Errorf("Expected message '🚨 0 blocked flow(s) detected', got '%s'", result.SecurityInsights.Message)
	}
}

func TestAnalyzeBlockedFlows_SecurityInsights(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	blockedLogs := []types.FlowLog{
		{
			SourceName:      "suspicious-pod",
			SourceNamespace: "default",
			DestName:        "sensitive-db",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        3306,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       100,
			PacketsOut:      0,
			BytesIn:         10240,
			BytesOut:        0,
		},
	}

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", blockedLogs)

	if result.SecurityInsights.Message != "🚨 1 blocked flow(s) detected" {
		t.Errorf("Expected message '🚨 1 blocked flow(s) detected', got '%s'", result.SecurityInsights.Message)
	}

	if len(result.SecurityInsights.Recommendations) != 4 {
		t.Errorf("Expected 4 recommendations, got %d", len(result.SecurityInsights.Recommendations))
	}

	expectedRecommendations := []string{
		"Review each blocking policy to ensure it aligns with your security requirements",
		"Consider if any blocked flows represent legitimate traffic that should be allowed",
		"Verify that policy ordering and tier configuration are correct",
		"Monitor for patterns that might indicate security threats or misconfigurations",
	}

	for i, expected := range expectedRecommendations {
		if result.SecurityInsights.Recommendations[i] != expected {
			t.Errorf("Recommendation %d mismatch.\nExpected: %s\nGot: %s",
				i, expected, result.SecurityInsights.Recommendations[i])
		}
	}
}

func TestAnalyzeBlockedFlows_WithBlockingPolicies(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	blockedLogs := []types.FlowLog{
		{
			SourceName:      "app-1",
			SourceNamespace: "default",
			DestName:        "db-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        5432,
			Action:          "deny",
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       10,
			PacketsOut:      0,
			BytesIn:         1024,
			BytesOut:        0,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{
						Name:   "block-db-access",
						Tier:   "security",
						Kind:   "NetworkPolicy",
						Action: "Deny",
					},
					{
						Name:   "default-deny",
						Tier:   "default",
						Kind:   "GlobalNetworkPolicy",
						Action: "Deny",
					},
				},
			},
		},
	}

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", blockedLogs)

	if len(result.BlockedFlows) != 1 {
		t.Fatalf("Expected 1 blocked flow detail, got %d", len(result.BlockedFlows))
	}

	detail := result.BlockedFlows[0]

	// Should have blocking policies
	if len(detail.BlockingPolicies) == 0 {
		t.Error("Expected blocking policies to be extracted")
	}

	// Should have analysis with total blocking policies count
	if detail.Analysis.TotalBlockingPolicies != len(detail.BlockingPolicies) {
		t.Errorf("Expected total blocking policies to match count. Got %d, expected %d",
			detail.Analysis.TotalBlockingPolicies, len(detail.BlockingPolicies))
	}

	// Should have recommendation
	if detail.Analysis.Recommendation == "" {
		t.Error("Expected non-empty recommendation")
	}
}

// TestAnalyzeBlockedFlows_WithStagedPolicies tests handling of flows with staged deny policies
func TestAnalyzeBlockedFlows_WithStagedPolicies(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	analyzer := NewBlockedFlowAnalyzer(policyAnalyzer)

	blockedLogs := []types.FlowLog{
		{
			SourceName:      "app-1",
			SourceNamespace: "default",
			DestName:        "db-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        5432,
			Action:          "Deny", // Currently blocked
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       10,
			PacketsOut:      0,
			BytesIn:         1024,
			BytesOut:        0,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{
						Name:   "block-db-access",
						Tier:   "security",
						Kind:   "NetworkPolicy",
						Action: "Deny",
					},
				},
			},
		},
		{
			SourceName:      "app-2",
			SourceNamespace: "default",
			DestName:        "api-1",
			DestNamespace:   "production",
			Protocol:        "TCP",
			DestPort:        443,
			Action:          "Allow", // Currently allowed but would be blocked
			Reporter:        "dst",
			StartTime:       "2024-11-07T10:00:00Z",
			EndTime:         "2024-11-07T10:00:05Z",
			PacketsIn:       100,
			PacketsOut:      50,
			BytesIn:         10240,
			BytesOut:        5120,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{
						Name:   "allow-egress",
						Tier:   "default",
						Kind:   "NetworkPolicy",
						Action: "Allow",
					},
				},
				Pending: []types.Policy{
					{
						Name:      "staged-deny-egress",
						Tier:      "security",
						Kind:      "CalicoNetworkPolicy",
						Namespace: "production",
						Action:    "Deny",
					},
				},
			},
		},
	}

	result := analyzer.AnalyzeBlockedFlows(context.Background(), "default", blockedLogs)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Analysis.TotalBlockedFlows != 2 {
		t.Errorf("Expected 2 flows analyzed, got %d", result.Analysis.TotalBlockedFlows)
	}

	if len(result.BlockedFlows) != 2 {
		t.Fatalf("Expected 2 blocked flow details, got %d", len(result.BlockedFlows))
	}

	// Check first flow (currently blocked)
	flow1 := result.BlockedFlows[0]
	if flow1.Flow.BlockStatus != "currently_blocked" {
		t.Errorf("Expected first flow to have BlockStatus 'currently_blocked', got '%s'", flow1.Flow.BlockStatus)
	}
	if flow1.Flow.Action != "Deny" {
		t.Errorf("Expected first flow action 'Deny', got '%s'", flow1.Flow.Action)
	}

	// Check second flow (staged block)
	flow2 := result.BlockedFlows[1]
	if flow2.Flow.BlockStatus != "staged_block" {
		t.Errorf("Expected second flow to have BlockStatus 'staged_block', got '%s'", flow2.Flow.BlockStatus)
	}
	if flow2.Flow.Action != "Allow" {
		t.Errorf("Expected second flow action 'Allow', got '%s'", flow2.Flow.Action)
	}

	// Check security insights message mentions both types
	message := result.SecurityInsights.Message
	if !containsSubstring(message, "currently blocked") {
		t.Errorf("Expected security insights to mention 'currently blocked', got: %s", message)
	}
	if !containsSubstring(message, "staged policies") {
		t.Errorf("Expected security insights to mention 'staged policies', got: %s", message)
	}

	// Check for staged policy specific recommendation
	hasStageRecommendation := false
	for _, rec := range result.SecurityInsights.Recommendations {
		if containsSubstring(rec, "STAGED POLICIES DETECTED") {
			hasStageRecommendation = true
			break
		}
	}
	if !hasStageRecommendation {
		t.Error("Expected recommendations to include staged policy warning")
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstring(s, substr) >= 0
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
