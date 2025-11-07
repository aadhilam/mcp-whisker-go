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

	if service.analytics == nil {
		t.Error("Expected analytics to be initialized, got nil")
	}

	if service.flowAggregator == nil {
		t.Error("Expected flowAggregator to be initialized, got nil")
	}

	if service.kubeconfigPath != "/path/to/kubeconfig" {
		t.Errorf("Expected kubeconfigPath to be /path/to/kubeconfig, got %s", service.kubeconfigPath)
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
