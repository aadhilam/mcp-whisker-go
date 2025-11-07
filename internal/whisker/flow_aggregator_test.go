package whisker

import (
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewFlowAggregator(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	if aggregator == nil {
		t.Fatal("Expected NewFlowAggregator to return non-nil FlowAggregator")
	}

	if aggregator.policyAnalyzer == nil {
		t.Error("Expected policyAnalyzer to be initialized")
	}
}

func TestFormatAction(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Allow", "✅ Allow"},
		{"Deny", "🚨 Deny"},
		{"N/A", "❌ N/A"},
		{"Unknown", "Unknown"},
	}

	for _, test := range tests {
		result := formatAction(test.input)
		if result != test.expected {
			t.Errorf("formatAction(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGenerateFlowSummary(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	logs := []types.FlowLog{
		{
			SourceName:      "pod-1",
			SourceNamespace: "default",
			DestName:        "svc-1",
			DestNamespace:   "default",
			Protocol:        "TCP",
			DestPort:        80,
			Action:          "Allow",
			StartTime:       "2024-01-01T12:00:00Z",
			EndTime:         "2024-01-01T12:01:00Z",
			PacketsIn:       100,
			PacketsOut:      50,
			BytesIn:         1024,
			BytesOut:        512,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-http", Namespace: "default", Kind: "CalicoNetworkPolicy"},
				},
			},
		},
	}

	summary := aggregator.GenerateFlowSummary("default", logs)

	if summary == nil {
		t.Fatal("Expected GenerateFlowSummary to return non-nil summary")
	}

	if summary.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got %s", summary.Namespace)
	}

	if summary.Analysis.TotalUniqueFlows != 1 {
		t.Errorf("Expected 1 unique flow, got %d", summary.Analysis.TotalUniqueFlows)
	}

	if summary.Analysis.TotalLogEntries != 1 {
		t.Errorf("Expected 1 log entry, got %d", summary.Analysis.TotalLogEntries)
	}

	if len(summary.Flows) != 1 {
		t.Errorf("Expected 1 flow, got %d", len(summary.Flows))
	}

	if summary.Statistics.Flows.Allowed != 1 {
		t.Errorf("Expected 1 allowed flow, got %d", summary.Statistics.Flows.Allowed)
	}

	if summary.Statistics.Flows.Blocked != 0 {
		t.Errorf("Expected 0 blocked flows, got %d", summary.Statistics.Flows.Blocked)
	}
}

func TestGenerateFlowSummary_WithBlockedFlow(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	logs := []types.FlowLog{
		{
			SourceName:      "pod-1",
			SourceNamespace: "default",
			DestName:        "svc-blocked",
			DestNamespace:   "restricted",
			Protocol:        "TCP",
			DestPort:        443,
			Action:          "Deny",
			Reporter:        "Src",
			StartTime:       "2024-01-01T12:00:00Z",
			EndTime:         "2024-01-01T12:01:00Z",
			PacketsIn:       0,
			PacketsOut:      10,
			BytesIn:         0,
			BytesOut:        512,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "deny-restricted", Namespace: "default", Kind: "CalicoNetworkPolicy", Action: "Deny"},
				},
			},
		},
	}

	summary := aggregator.GenerateFlowSummary("default", logs)

	if summary.Statistics.Flows.Blocked != 1 {
		t.Errorf("Expected 1 blocked flow, got %d", summary.Statistics.Flows.Blocked)
	}

	if summary.Statistics.Flows.Allowed != 0 {
		t.Errorf("Expected 0 allowed flows, got %d", summary.Statistics.Flows.Allowed)
	}

	if summary.SecurityAlerts == nil {
		t.Error("Expected security alerts for blocked flows")
	} else {
		if len(summary.SecurityAlerts.BlockedFlows) != 1 {
			t.Errorf("Expected 1 blocked flow alert, got %d", len(summary.SecurityAlerts.BlockedFlows))
		}
	}

	if len(summary.Flows) > 0 {
		flow := summary.Flows[0]
		if flow.Status != "🚨 BLOCKED" {
			t.Errorf("Expected status '🚨 BLOCKED', got %s", flow.Status)
		}
	}
}

func TestGenerateFlowSummary_AggregateMultipleLogs(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	// Same flow from multiple log entries
	logs := []types.FlowLog{
		{
			SourceName:      "pod-1",
			SourceNamespace: "default",
			DestName:        "svc-1",
			DestNamespace:   "default",
			Protocol:        "TCP",
			DestPort:        80,
			Action:          "Allow",
			StartTime:       "2024-01-01T12:00:00Z",
			EndTime:         "2024-01-01T12:01:00Z",
			PacketsIn:       100,
			PacketsOut:      50,
			BytesIn:         1024,
			BytesOut:        512,
		},
		{
			SourceName:      "pod-1",
			SourceNamespace: "default",
			DestName:        "svc-1",
			DestNamespace:   "default",
			Protocol:        "TCP",
			DestPort:        80,
			Action:          "Allow",
			StartTime:       "2024-01-01T12:01:00Z",
			EndTime:         "2024-01-01T12:02:00Z",
			PacketsIn:       150,
			PacketsOut:      75,
			BytesIn:         2048,
			BytesOut:        1024,
		},
	}

	summary := aggregator.GenerateFlowSummary("default", logs)

	if summary.Analysis.TotalUniqueFlows != 1 {
		t.Errorf("Expected 1 unique flow (aggregated), got %d", summary.Analysis.TotalUniqueFlows)
	}

	if summary.Analysis.TotalLogEntries != 2 {
		t.Errorf("Expected 2 log entries, got %d", summary.Analysis.TotalLogEntries)
	}

	if len(summary.Flows) != 1 {
		t.Fatalf("Expected 1 aggregated flow, got %d", len(summary.Flows))
	}

	flow := summary.Flows[0]
	// Check aggregated metrics
	expectedTotalPackets := int64(250 + 125) // 100+150 in, 50+75 out
	if flow.Traffic.Packets.Total != expectedTotalPackets {
		t.Errorf("Expected %d total packets, got %d", expectedTotalPackets, flow.Traffic.Packets.Total)
	}

	expectedTotalBytes := int64(3072 + 1536) // 1024+2048 in, 512+1024 out
	if flow.Traffic.Bytes.Total != expectedTotalBytes {
		t.Errorf("Expected %d total bytes, got %d", expectedTotalBytes, flow.Traffic.Bytes.Total)
	}
}

func TestAggregateFlows(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	logs := []types.FlowLog{
		{
			SourceName:      "pod-1",
			SourceNamespace: "default",
			DestName:        "svc-1",
			DestNamespace:   "default",
			Protocol:        "TCP",
			DestPort:        80,
			Action:          "Allow",
			PacketsIn:       100,
			PacketsOut:      50,
			BytesIn:         1024,
			BytesOut:        512,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-http", Namespace: "default"},
				},
			},
		},
		{
			SourceName:      "pod-2",
			SourceNamespace: "default",
			DestName:        "svc-2",
			DestNamespace:   "default",
			Protocol:        "TCP",
			DestPort:        443,
			Action:          "Allow",
			PacketsIn:       200,
			PacketsOut:      100,
			BytesIn:         2048,
			BytesOut:        1024,
			Policies: types.Policies{
				Enforced: []types.Policy{
					{Name: "allow-https", Namespace: "default"},
				},
			},
		},
	}

	entries := aggregator.AggregateFlows(logs)

	if len(entries) != 2 {
		t.Errorf("Expected 2 aggregated entries, got %d", len(entries))
	}

	// Check that formatting was applied
	for _, entry := range entries {
		if entry.PacketsInStr == "" {
			t.Error("Expected PacketsInStr to be formatted")
		}
		if entry.BytesInStr == "" {
			t.Error("Expected BytesInStr to be formatted")
		}
	}
}

func TestAggregateFlows_WithNetworkClassification(t *testing.T) {
	policyAnalyzer := NewPolicyAnalyzer("")
	aggregator := NewFlowAggregator(policyAnalyzer)

	logs := []types.FlowLog{
		{
			SourceName:      "10.0.0.5",
			SourceNamespace: "",  // Empty namespace indicates external
			DestName:        "8.8.8.8",
			DestNamespace:   "",
			Protocol:        "UDP",
			DestPort:        53,
			Action:          "Allow",
			PacketsIn:       10,
			PacketsOut:      10,
			BytesIn:         512,
			BytesOut:        512,
		},
	}

	entries := aggregator.AggregateFlows(logs)

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	// With empty namespaces, IPs are classified as networks
	// 10.0.0.5 is a private IP
	if entry.Source != "PRIVATE NETWORK" {
		t.Errorf("Expected source to be 'PRIVATE NETWORK', got %s", entry.Source)
	}

	// 8.8.8.8 is a public IP
	if entry.Destination != "PUBLIC NETWORK" {
		t.Errorf("Expected destination to be 'PUBLIC NETWORK', got %s", entry.Destination)
	}

	// Namespace should be "-" for networks
	if entry.SourceNamespace != "-" {
		t.Errorf("Expected source namespace to be '-', got %s", entry.SourceNamespace)
	}

	if entry.DestNamespace != "-" {
		t.Errorf("Expected dest namespace to be '-', got %s", entry.DestNamespace)
	}
}
