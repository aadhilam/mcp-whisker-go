package whisker

import (
	"strings"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNormalizePodName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ReplicaSet pod name",
			input:    "coredns-789465848c-abc123",
			expected: "coredns-789465848c-*",
		},
		{
			name:     "Another ReplicaSet pod",
			input:    "metrics-server-fc9846b48-xyz99",
			expected: "metrics-server-fc9846b48-*",
		},
		{
			name:     "Regular name without pattern",
			input:    "my-service",
			expected: "my-service",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Deployment pod",
			input:    "goldmane-ff655769-abc12",
			expected: "goldmane-ff655769-*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePodName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizePodName(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClassifyNetwork(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		namespace string
		expected  string
	}{
		{
			name:      "Empty name",
			inputName: "",
			namespace: "",
			expected:  "PRIVATE NETWORK",
		},
		{
			name:      "Private IP",
			inputName: "10.0.0.1",
			namespace: "",
			expected:  "PRIVATE NETWORK",
		},
		{
			name:      "Private IP 192.168",
			inputName: "192.168.1.1",
			namespace: "",
			expected:  "PRIVATE NETWORK",
		},
		{
			name:      "Public IP",
			inputName: "8.8.8.8",
			namespace: "",
			expected:  "PUBLIC NETWORK",
		},
		{
			name:      "External domain",
			inputName: "api.example.com",
			namespace: "",
			expected:  "PUBLIC NETWORK",
		},
		{
			name:      "Pod with namespace",
			inputName: "my-pod",
			namespace: "default",
			expected:  "my-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyNetwork(tt.inputName, tt.namespace)
			if result != tt.expected {
				t.Errorf("classifyNetwork(%q, %q) = %q, expected %q",
					tt.inputName, tt.namespace, result, tt.expected)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{name: "10.0.0.0/8", ip: "10.0.0.1", expected: true},
		{name: "172.16.0.0/12", ip: "172.16.0.1", expected: true},
		{name: "192.168.0.0/16", ip: "192.168.1.1", expected: true},
		{name: "localhost", ip: "127.0.0.1", expected: true},
		{name: "public IP", ip: "8.8.8.8", expected: false},
		{name: "not an IP", ip: "not-an-ip", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPrivateIP(tt.ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestCategorizeTraffic(t *testing.T) {
	tests := []struct {
		name          string
		protocol      string
		port          int
		destNamespace string
		expected      string
	}{
		{name: "DNS UDP", protocol: "UDP", port: 53, destNamespace: "", expected: "DNS Queries"},
		{name: "DNS TCP", protocol: "TCP", port: 53, destNamespace: "", expected: "DNS Queries"},
		{name: "HTTPS", protocol: "TCP", port: 443, destNamespace: "", expected: "API/HTTPS"},
		{name: "Kubelet", protocol: "TCP", port: 10250, destNamespace: "", expected: "Metrics Collection"},
		{name: "Calico service", protocol: "TCP", port: 7443, destNamespace: "calico-system", expected: "Calico Services"},
		{name: "Monitoring", protocol: "TCP", port: 9153, destNamespace: "", expected: "Monitoring"},
		{name: "HTTP", protocol: "TCP", port: 80, destNamespace: "", expected: "HTTP"},
		{name: "MySQL", protocol: "TCP", port: 3306, destNamespace: "", expected: "Database"},
		{name: "Other", protocol: "TCP", port: 9999, destNamespace: "", expected: "Other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := categorizeTraffic(tt.protocol, tt.port, tt.destNamespace)
			if result != tt.expected {
				t.Errorf("categorizeTraffic(%q, %d, %q) = %q, expected %q",
					tt.protocol, tt.port, tt.destNamespace, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{name: "bytes", bytes: 500, expected: "500B"},
		{name: "kilobytes", bytes: 1700, expected: "1.7KB"},
		{name: "megabytes", bytes: 3300000, expected: "3.3MB"},
		{name: "gigabytes", bytes: 1300000000, expected: "1.3GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, expected %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatPackets(t *testing.T) {
	tests := []struct {
		name     string
		packets  int64
		expected string
	}{
		{name: "small", packets: 17, expected: "17"},
		{name: "hundreds", packets: 400, expected: "400"},
		{name: "thousands", packets: 2374, expected: "2.4K"},
		{name: "millions", packets: 1385000, expected: "1.4M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPackets(tt.packets)
			if result != tt.expected {
				t.Errorf("formatPackets(%d) = %q, expected %q", tt.packets, result, tt.expected)
			}
		})
	}
}

func TestGetPrimaryPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policies []types.Policy
		expected string
	}{
		{
			name:     "empty policies",
			policies: []types.Policy{},
			expected: "-",
		},
		{
			name: "single policy",
			policies: []types.Policy{
				{Name: "allow-all", Namespace: "default"},
			},
			expected: "default.allow-all",
		},
		{
			name: "multiple same policy",
			policies: []types.Policy{
				{Name: "kns.kube-system", Namespace: ""},
				{Name: "kns.kube-system", Namespace: ""},
				{Name: "other-policy", Namespace: "default"},
			},
			expected: "kns.kube-system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrimaryPolicy(tt.policies)
			if result != tt.expected {
				t.Errorf("getPrimaryPolicy() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestExtractPrimaryActivity(t *testing.T) {
	tests := []struct {
		name     string
		flows    []types.FlowLog
		expected string
	}{
		{
			name:     "empty flows",
			flows:    []types.FlowLog{},
			expected: "",
		},
		{
			name: "DNS flows",
			flows: []types.FlowLog{
				{Protocol: "UDP", DestPort: 53, DestNamespace: "kube-system"},
				{Protocol: "UDP", DestPort: 53, DestNamespace: "kube-system"},
			},
			expected: "dns queries",
		},
		{
			name: "mixed activities",
			flows: []types.FlowLog{
				{Protocol: "UDP", DestPort: 53, DestNamespace: "kube-system"},
				{Protocol: "TCP", DestPort: 443, DestNamespace: "default"},
				{Protocol: "TCP", DestPort: 443, DestNamespace: "default"},
			},
			// Both activities are significant (threshold is 20% of max)
			// dns queries: 1, api/https: 2, so both appear
			expected: "dns queries, api/https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPrimaryActivity(tt.flows)
			// For cases with multiple activities, the order may vary due to map iteration
			// Just check that the result contains the expected activities
			if tt.name == "mixed activities" {
				if !(strings.Contains(result, "dns queries") && strings.Contains(result, "api/https")) {
					t.Errorf("extractPrimaryActivity() = %q, expected to contain both 'dns queries' and 'api/https'", result)
				}
			} else if result != tt.expected {
				t.Errorf("extractPrimaryActivity() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
