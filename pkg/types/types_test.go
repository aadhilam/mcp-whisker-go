package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlowLogJSONSerialization(t *testing.T) {
	flowLog := FlowLog{
		StartTime:       "2023-01-01T00:00:00Z",
		EndTime:         "2023-01-01T00:01:00Z",
		Action:          "Allow",
		SourceName:      "test-pod",
		SourceNamespace: "test-ns",
		Protocol:        "TCP",
		DestPort:        8080,
		PacketsIn:       100,
		PacketsOut:      50,
		BytesIn:         1024,
		BytesOut:        512,
	}
	
	// Test serialization
	data, err := json.Marshal(flowLog)
	if err != nil {
		t.Fatalf("Failed to marshal FlowLog: %v", err)
	}
	
	// Test deserialization
	var unmarshaled FlowLog
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal FlowLog: %v", err)
	}
	
	// Verify fields
	if unmarshaled.SourceName != flowLog.SourceName {
		t.Errorf("Expected SourceName %s, got %s", flowLog.SourceName, unmarshaled.SourceName)
	}
	
	if unmarshaled.PacketsIn != flowLog.PacketsIn {
		t.Errorf("Expected PacketsIn %d, got %d", flowLog.PacketsIn, unmarshaled.PacketsIn)
	}
}

func TestTrafficMetricCalculation(t *testing.T) {
	metric := TrafficMetric{
		In:  100,
		Out: 200,
	}
	
	expectedTotal := int64(300)
	metric.Total = metric.In + metric.Out
	
	if metric.Total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, metric.Total)
	}
}

func TestTimeRangeInfoDuration(t *testing.T) {
	timeRange := TimeRangeInfo{
		Start: "2023-01-01T00:00:00Z",
		End:   "2023-01-01T00:01:00Z",
	}
	
	startTime, _ := time.Parse(time.RFC3339, timeRange.Start)
	endTime, _ := time.Parse(time.RFC3339, timeRange.End)
	expectedDuration := endTime.Sub(startTime)
	
	timeRange.Duration = expectedDuration
	
	if timeRange.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, timeRange.Duration)
	}
}

func TestPolicyWithTrigger(t *testing.T) {
	trigger := &Policy{
		Kind:      "CalicoNetworkPolicy",
		Name:      "trigger-policy",
		Namespace: "test-ns",
		Action:    "Allow",
	}
	
	policy := Policy{
		Kind:      "CalicoNetworkPolicy",
		Name:      "main-policy",
		Namespace: "test-ns",
		Action:    "Deny",
		Trigger:   trigger,
	}
	
	if policy.Trigger == nil {
		t.Error("Expected trigger to be set")
	}
	
	if policy.Trigger.Name != "trigger-policy" {
		t.Errorf("Expected trigger name to be trigger-policy, got %s", policy.Trigger.Name)
	}
}

func TestNamespaceFlowSummaryComplete(t *testing.T) {
	summary := NamespaceFlowSummary{
		Namespace: "test-ns",
		Analysis: AnalysisInfo{
			TotalUniqueFlows: 5,
			TotalLogEntries:  10,
		},
		Statistics: StatisticsInfo{
			Flows: FlowStats{
				Total:   5,
				Allowed: 4,
				Blocked: 1,
			},
			Traffic: TrafficStats{
				TotalPackets: 1000,
				TotalBytes:   1024000,
			},
		},
		Flows: []FlowSummary{
			{
				Source: FlowEndpoint{
					Name:      "source-pod",
					Namespace: "test-ns",
					Action:    "Allow",
				},
				Destination: FlowEndpoint{
					Name:      "dest-pod",
					Namespace: "test-ns",
					Action:    "Allow",
				},
				Status: "✅ ALLOWED",
			},
		},
	}
	
	if summary.Namespace != "test-ns" {
		t.Errorf("Expected namespace test-ns, got %s", summary.Namespace)
	}
	
	if summary.Statistics.Flows.Total != 5 {
		t.Errorf("Expected 5 total flows, got %d", summary.Statistics.Flows.Total)
	}
	
	if len(summary.Flows) != 1 {
		t.Errorf("Expected 1 flow summary, got %d", len(summary.Flows))
	}
}

func TestSecurityAlertsCreation(t *testing.T) {
	alerts := &SecurityAlerts{
		Message: "🚨 2 blocked flow(s) detected - immediate attention required!",
		BlockedFlows: []string{
			"pod-a → pod-b:8080",
			"pod-c → pod-d:443",
		},
	}
	
	if len(alerts.BlockedFlows) != 2 {
		t.Errorf("Expected 2 blocked flows, got %d", len(alerts.BlockedFlows))
	}
	
	expectedMessage := "🚨 2 blocked flow(s) detected - immediate attention required!"
	if alerts.Message != expectedMessage {
		t.Errorf("Expected message %s, got %s", expectedMessage, alerts.Message)
	}
}

// Benchmark tests
func BenchmarkFlowLogMarshal(b *testing.B) {
	flowLog := FlowLog{
		StartTime:       "2023-01-01T00:00:00Z",
		EndTime:         "2023-01-01T00:01:00Z",
		Action:          "Allow",
		SourceName:      "test-pod",
		SourceNamespace: "test-ns",
		Protocol:        "TCP",
		DestPort:        8080,
		PacketsIn:       100,
		PacketsOut:      50,
		BytesIn:         1024,
		BytesOut:        512,
	}
	
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(flowLog)
	}
}