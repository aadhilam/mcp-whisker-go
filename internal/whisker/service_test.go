package whisker

import (
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewService(t *testing.T) {
	service := NewService("/path/to/kubeconfig")

	if service == nil {
		t.Fatal("Expected service to be created, got nil")
	}

	if service.httpClient == nil {
		t.Error("Expected httpClient to be initialized, got nil")
	}

	if service.policyAnalyzer == nil {
		t.Error("Expected policyAnalyzer to be initialized, got nil")
	}

	if service.kubeconfigPath != "/path/to/kubeconfig" {
		t.Errorf("Expected kubeconfigPath to be /path/to/kubeconfig, got %s", service.kubeconfigPath)
	}
}

func TestFormatAction(t *testing.T) {
	service := NewService("")

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
		result := service.formatAction(test.input)
		if result != test.expected {
			t.Errorf("formatAction(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestMapPolicyKindToResource(t *testing.T) {
	service := NewService("")

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
		result := service.mapPolicyKindToResource(test.input)
		if result != test.expected {
			t.Errorf("mapPolicyKindToResource(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGetBlockingReason(t *testing.T) {
	service := NewService("")

	tests := []struct {
		input    string
		expected string
	}{
		{"Deny", "Explicit deny rule"},
		{"Allow", "End of tier default deny"},
		{"Other", "End of tier default deny"},
	}

	for _, test := range tests {
		result := service.getBlockingReason(test.input)
		if result != test.expected {
			t.Errorf("getBlockingReason(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGenerateRecommendation(t *testing.T) {
	service := NewService("")

	// Test with blocking policies
	blockingPolicies := []types.BlockingPolicy{
		{BlockingReason: "Test reason"},
	}

	result := service.generateRecommendation(blockingPolicies)
	expected := "Review the identified policies to understand why traffic is being blocked. Consider modifying the policy rules if this traffic should be allowed."

	if result != expected {
		t.Errorf("generateRecommendation with policies = %s, expected %s", result, expected)
	}

	// Test without blocking policies
	emptyPolicies := []types.BlockingPolicy{}
	result = service.generateRecommendation(emptyPolicies)
	expected = "No specific blocking policies identified. This may be due to default deny behavior or policy ordering."

	if result != expected {
		t.Errorf("generateRecommendation without policies = %s, expected %s", result, expected)
	}
}

// Benchmark tests
func BenchmarkFormatAction(b *testing.B) {
	service := NewService("")

	for i := 0; i < b.N; i++ {
		service.formatAction("Allow")
	}
}

// Mock test for flow aggregation
func TestConvertToFlowSummary(t *testing.T) {
	service := NewService("")

	flow := &aggregatedFlow{
		source:          "test-pod",
		sourceNamespace: "test-ns",
		destination:     "dest-pod",
		destNamespace:   "dest-ns",
		protocol:        "TCP",
		port:            8080,
		sourceAction:    "Allow",
		destAction:      "Allow",
		packetsIn:       100,
		packetsOut:      50,
		bytesIn:         1024,
		bytesOut:        512,
		startTime:       "2023-01-01T00:00:00Z",
		endTime:         "2023-01-01T00:01:00Z",
		sourcePolicies:  map[string]bool{"policy1": true},
		destPolicies:    map[string]bool{"policy2": true},
		enforcedPolicies: []types.PolicyDetail{
			{
				Name:      "test-policy",
				Namespace: "test-ns",
				Kind:      "CalicoNetworkPolicy",
				Action:    "Allow",
			},
		},
	}

	summary := service.convertToFlowSummary(flow)

	if summary.Source.Name != "test-pod" {
		t.Errorf("Expected source name to be test-pod, got %s", summary.Source.Name)
	}

	if summary.Status != "✅ ALLOWED" {
		t.Errorf("Expected status to be ✅ ALLOWED, got %s", summary.Status)
	}

	if summary.Traffic.Packets.Total != 150 {
		t.Errorf("Expected total packets to be 150, got %d", summary.Traffic.Packets.Total)
	}

	if summary.Traffic.Bytes.Total != 1536 {
		t.Errorf("Expected total bytes to be 1536, got %d", summary.Traffic.Bytes.Total)
	}
}
