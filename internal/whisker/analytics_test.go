package whisker

import (
	"strings"
	"testing"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

func TestNewAnalytics(t *testing.T) {
	analytics := NewAnalytics()
	if analytics == nil {
		t.Fatal("Expected NewAnalytics to return non-nil Analytics")
	}
}

func TestDetermineTimeRange(t *testing.T) {
	tests := []struct {
		name     string
		logs     []types.FlowLog
		expected string
	}{
		{
			name:     "Empty logs",
			logs:     []types.FlowLog{},
			expected: "Unknown",
		},
		{
			name: "Single log",
			logs: []types.FlowLog{
				{StartTime: "2024-01-01T12:00:00Z", EndTime: "2024-01-01T12:05:00Z"},
			},
			expected: "2024-01-01T12:00:00Z to 2024-01-01T12:05:00Z",
		},
		{
			name: "Multiple logs with different times",
			logs: []types.FlowLog{
				{StartTime: "2024-01-01T13:00:00Z", EndTime: "2024-01-01T13:05:00Z"},
				{StartTime: "2024-01-01T12:00:00Z", EndTime: "2024-01-01T12:05:00Z"},
				{StartTime: "2024-01-01T14:00:00Z", EndTime: "2024-01-01T14:05:00Z"},
			},
			expected: "2024-01-01T12:00:00Z to 2024-01-01T14:05:00Z",
		},
	}

	analytics := NewAnalytics()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analytics.DetermineTimeRange(tt.logs)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCalculateTopSources(t *testing.T) {
	tests := []struct {
		name          string
		logs          []types.FlowLog
		expectedCount int
		expectedTop   string
	}{
		{
			name:          "Empty logs",
			logs:          []types.FlowLog{},
			expectedCount: 0,
		},
		{
			name: "Single source",
			logs: []types.FlowLog{
				{SourceName: "pod-1", SourceNamespace: "default"},
			},
			expectedCount: 1,
			expectedTop:   "pod-1", // Normalized name without namespace
		},
		{
			name: "Multiple sources - count flows",
			logs: []types.FlowLog{
				{SourceName: "pod-1", SourceNamespace: "default"},
				{SourceName: "pod-2", SourceNamespace: "default"},
				{SourceName: "pod-1", SourceNamespace: "default"},
				{SourceName: "pod-1", SourceNamespace: "default"},
			},
			expectedCount: 2,
			expectedTop:   "pod-1", // 3 flows
		},
		{
			name: "More than 10 sources",
			logs: func() []types.FlowLog {
				logs := []types.FlowLog{}
				for i := 0; i < 15; i++ {
					logs = append(logs, types.FlowLog{
						SourceName:      "pod-" + string(rune('a'+i)),
						SourceNamespace: "default",
					})
				}
				return logs
			}(),
			expectedCount: 10, // Should cap at 10
		},
	}

	analytics := NewAnalytics()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analytics.CalculateTopSources(tt.logs)
			
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d sources, got %d", tt.expectedCount, len(result))
			}

			if tt.expectedTop != "" && len(result) > 0 {
				if result[0].Name != tt.expectedTop {
					t.Errorf("Expected top source %s, got %s", tt.expectedTop, result[0].Name)
				}
			}
		})
	}
}

func TestCalculateTopDestinations(t *testing.T) {
	tests := []struct {
		name          string
		logs          []types.FlowLog
		expectedCount int
		expectedTop   string
	}{
		{
			name:          "Empty logs",
			logs:          []types.FlowLog{},
			expectedCount: 0,
		},
		{
			name: "Single destination",
			logs: []types.FlowLog{
				{DestName: "svc-1", DestNamespace: "default"},
			},
			expectedCount: 1,
			expectedTop:   "svc-1", // Normalized name without namespace
		},
		{
			name: "Multiple destinations - count flows",
			logs: []types.FlowLog{
				{DestName: "svc-1", DestNamespace: "default"},
				{DestName: "svc-2", DestNamespace: "default"},
				{DestName: "svc-1", DestNamespace: "default"},
				{DestName: "svc-1", DestNamespace: "default"},
			},
			expectedCount: 2,
			expectedTop:   "svc-1", // 3 flows
		},
	}

	analytics := NewAnalytics()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analytics.CalculateTopDestinations(tt.logs)
			
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d destinations, got %d", tt.expectedCount, len(result))
			}

			if tt.expectedTop != "" && len(result) > 0 {
				if result[0].Name != tt.expectedTop {
					t.Errorf("Expected top destination %s, got %s", tt.expectedTop, result[0].Name)
				}
			}
		})
	}
}

func TestAnalyzeNamespaceActivity(t *testing.T) {
	tests := []struct {
		name          string
		logs          []types.FlowLog
		expectedCount int
		expectedTop   string
	}{
		{
			name:          "Empty logs",
			logs:          []types.FlowLog{},
			expectedCount: 0,
		},
		{
			name: "Single namespace egress",
			logs: []types.FlowLog{
				{SourceNamespace: "default", BytesOut: 100},
			},
			expectedCount: 1,
			expectedTop:   "default",
		},
		{
			name: "Multiple namespaces with ingress and egress",
			logs: []types.FlowLog{
				{SourceNamespace: "app", BytesOut: 100},
				{DestNamespace: "app", BytesIn: 200},
				{SourceNamespace: "db", BytesOut: 300},
			},
			expectedCount: 2,
			expectedTop:   "app", // 2 flows (1 egress + 1 ingress)
		},
		{
			name: "Namespace as both source and destination",
			logs: []types.FlowLog{
				{SourceNamespace: "app", BytesOut: 100},
				{SourceNamespace: "app", BytesOut: 200},
				{DestNamespace: "app", BytesIn: 150},
				{SourceNamespace: "db", BytesOut: 50},
			},
			expectedCount: 2,
			expectedTop:   "app", // 3 flows total
		},
	}

	analytics := NewAnalytics()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analytics.AnalyzeNamespaceActivity(tt.logs)
			
			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d namespaces, got %d", tt.expectedCount, len(result))
			}

			if tt.expectedTop != "" && len(result) > 0 {
				if result[0].Namespace != tt.expectedTop {
					t.Errorf("Expected top namespace %s, got %s", tt.expectedTop, result[0].Namespace)
				}
			}

			// Verify traffic volume is formatted
			for _, activity := range result {
				if activity.TotalTrafficVolume == "" {
					t.Errorf("Expected traffic volume to be formatted, got empty string")
				}
				// Should contain "in" and "out"
				if !strings.Contains(activity.TotalTrafficVolume, "in") ||
					!strings.Contains(activity.TotalTrafficVolume, "out") {
					t.Errorf("Expected traffic volume format '* in / * out', got %s", activity.TotalTrafficVolume)
				}
			}
		})
	}
}

func TestCategorizeFlows(t *testing.T) {
	tests := []struct {
		name          string
		logs          []types.FlowLog
		expectedCats  []string
		minCategories int
	}{
		{
			name:          "Empty logs",
			logs:          []types.FlowLog{},
			expectedCats:  []string{},
			minCategories: 0,
		},
		{
			name: "HTTP and HTTPS traffic",
			logs: []types.FlowLog{
				{DestPort: 80, Protocol: "TCP", DestNamespace: "app"},
				{DestPort: 443, Protocol: "TCP", DestNamespace: "app"},
				{DestPort: 8080, Protocol: "TCP", DestNamespace: "app"},
			},
			expectedCats:  []string{"HTTP", "API/HTTPS"},
			minCategories: 2,
		},
		{
			name: "Database traffic",
			logs: []types.FlowLog{
				{DestPort: 3306, Protocol: "TCP", DestNamespace: "db"}, // MySQL
				{DestPort: 5432, Protocol: "TCP", DestNamespace: "db"}, // PostgreSQL
				{DestPort: 27017, Protocol: "TCP", DestNamespace: "db"}, // MongoDB
			},
			expectedCats:  []string{"Database"},
			minCategories: 1,
		},
		{
			name: "DNS traffic",
			logs: []types.FlowLog{
				{DestPort: 53, Protocol: "UDP", DestNamespace: "kube-system"},
				{DestPort: 53, Protocol: "TCP", DestNamespace: "kube-system"},
			},
			expectedCats:  []string{"DNS Queries"},
			minCategories: 1,
		},
		{
			name: "Mixed traffic types",
			logs: []types.FlowLog{
				{DestPort: 53, Protocol: "UDP", DestNamespace: "kube-system"},
				{DestPort: 443, Protocol: "TCP", DestNamespace: "app"},
				{DestPort: 3306, Protocol: "TCP", DestNamespace: "db"},
				{DestPort: 9999, Protocol: "TCP", DestNamespace: "app"},
			},
			expectedCats:  []string{"DNS Queries", "API/HTTPS", "Database", "Other"},
			minCategories: 4,
		},
	}

	analytics := NewAnalytics()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analytics.CategorizeFlows(tt.logs)
			
			if len(result) < tt.minCategories {
				t.Errorf("Expected at least %d categories, got %d", tt.minCategories, len(result))
			}

			// Check that expected categories are present
			categoryMap := make(map[string]bool)
			for _, cat := range result {
				categoryMap[cat.Category] = true
				// Verify each category has a description
				if cat.Description == "" {
					t.Errorf("Category %s has empty description", cat.Category)
				}
			}

			for _, expectedCat := range tt.expectedCats {
				if !categoryMap[expectedCat] {
					t.Errorf("Expected category %s not found in result", expectedCat)
				}
			}

			// Verify categories are sorted by count (descending)
			for i := 1; i < len(result); i++ {
				if result[i-1].Count < result[i].Count {
					t.Errorf("Categories not sorted by count: %s (%d) before %s (%d)",
						result[i-1].Category, result[i-1].Count,
						result[i].Category, result[i].Count)
				}
			}
		})
	}
}
