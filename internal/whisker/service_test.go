package whisker

import (
	"testing"
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

	if service.blockedFlowAnalyzer == nil {
		t.Error("Expected blockedFlowAnalyzer to be initialized, got nil")
	}

	if service.securityPostureAnalyzer == nil {
		t.Error("Expected securityPostureAnalyzer to be initialized, got nil")
	}

	if service.kubeconfigPath != "/path/to/kubeconfig" {
		t.Errorf("Expected kubeconfigPath to be /path/to/kubeconfig, got %s", service.kubeconfigPath)
	}
}
