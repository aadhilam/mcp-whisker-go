package whisker

import (
	"fmt"
	"sort"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// Analytics handles metrics calculation, traffic analysis, and statistics
type Analytics struct{}

// NewAnalytics creates a new analytics instance
func NewAnalytics() *Analytics {
	return &Analytics{}
}

// DetermineTimeRange extracts the time range from flow logs
func (a *Analytics) DetermineTimeRange(logs []types.FlowLog) string {
	if len(logs) == 0 {
		return "Unknown"
	}

	earliest := logs[0].StartTime
	latest := logs[0].EndTime

	for _, log := range logs {
		if log.StartTime < earliest {
			earliest = log.StartTime
		}
		if log.EndTime > latest {
			latest = log.EndTime
		}
	}

	return fmt.Sprintf("%s to %s", earliest, latest)
}

// CalculateTopSources identifies and ranks top traffic sources
func (a *Analytics) CalculateTopSources(logs []types.FlowLog) []types.TopTrafficEntity {
	sourceFlows := make(map[string][]types.FlowLog)

	for _, log := range logs {
		normalizedSource := normalizeEntityName(log.SourceName, log.SourceNamespace)
		sourceFlows[normalizedSource] = append(sourceFlows[normalizedSource], log)
	}

	// Convert to slice
	entities := []types.TopTrafficEntity{}
	for source, flows := range sourceFlows {
		entity := types.TopTrafficEntity{
			Name:            source,
			TotalFlows:      len(flows),
			PrimaryActivity: extractPrimaryActivity(flows),
		}
		entities = append(entities, entity)
	}

	// Sort by flow count (descending)
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].TotalFlows > entities[j].TotalFlows
	})

	// Return top 10
	if len(entities) > 10 {
		return entities[:10]
	}
	return entities
}

// CalculateTopDestinations identifies and ranks top traffic destinations
func (a *Analytics) CalculateTopDestinations(logs []types.FlowLog) []types.TopTrafficEntity {
	destFlows := make(map[string][]types.FlowLog)

	for _, log := range logs {
		normalizedDest := normalizeEntityName(log.DestName, log.DestNamespace)
		destFlows[normalizedDest] = append(destFlows[normalizedDest], log)
	}

	// Convert to slice
	entities := []types.TopTrafficEntity{}
	for dest, flows := range destFlows {
		entity := types.TopTrafficEntity{
			Name:            dest,
			TotalFlows:      len(flows),
			PrimaryActivity: extractPrimaryActivity(flows),
		}
		entities = append(entities, entity)
	}

	// Sort by flow count (descending)
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].TotalFlows > entities[j].TotalFlows
	})

	// Return top 10
	if len(entities) > 10 {
		return entities[:10]
	}
	return entities
}

// AnalyzeNamespaceActivity analyzes traffic by namespace
func (a *Analytics) AnalyzeNamespaceActivity(logs []types.FlowLog) []types.NamespaceActivityInfo {
	namespaceData := make(map[string]*types.NamespaceActivityInfo)

	for _, log := range logs {
		// Track source namespace (egress)
		if log.SourceNamespace != "" {
			if _, exists := namespaceData[log.SourceNamespace]; !exists {
				namespaceData[log.SourceNamespace] = &types.NamespaceActivityInfo{
					Namespace: log.SourceNamespace,
				}
			}
			namespaceData[log.SourceNamespace].EgressFlows++
			namespaceData[log.SourceNamespace].BytesOut += log.BytesOut
		}

		// Track destination namespace (ingress)
		if log.DestNamespace != "" {
			if _, exists := namespaceData[log.DestNamespace]; !exists {
				namespaceData[log.DestNamespace] = &types.NamespaceActivityInfo{
					Namespace: log.DestNamespace,
				}
			}
			namespaceData[log.DestNamespace].IngressFlows++
			namespaceData[log.DestNamespace].BytesIn += log.BytesIn
		}
	}

	// Convert to slice and format traffic volume
	activities := []types.NamespaceActivityInfo{}
	for _, data := range namespaceData {
		data.TotalTrafficVolume = fmt.Sprintf("~%s in / %s out",
			formatBytes(data.BytesIn), formatBytes(data.BytesOut))
		activities = append(activities, *data)
	}

	// Sort by total flows (ingress + egress)
	sort.Slice(activities, func(i, j int) bool {
		totalI := activities[i].IngressFlows + activities[i].EgressFlows
		totalJ := activities[j].IngressFlows + activities[j].EgressFlows
		return totalI > totalJ
	})

	return activities
}

// CategorizeFlows categorizes flows and counts them
func (a *Analytics) CategorizeFlows(logs []types.FlowLog) []types.TrafficCategory {
	categoryCounts := make(map[string]int)
	categoryDescriptions := map[string]string{
		"DNS Queries":        "DNS resolution traffic (port 53)",
		"API/HTTPS":          "HTTPS traffic to Kubernetes API and public endpoints (port 443)",
		"Metrics Collection": "Metrics server collecting from nodes (ports 10250, 4443)",
		"Calico Services":    "Traffic to Calico API server and related services",
		"Monitoring":         "Monitoring and metrics scraping (port 9153)",
		"HTTP":               "HTTP web traffic (ports 80, 8080)",
		"Database":           "Database connections (MySQL, PostgreSQL, MongoDB, Redis)",
		"Other":              "Other traffic not matching common categories",
	}

	for _, log := range logs {
		category := categorizeTraffic(log.Protocol, log.DestPort, log.DestNamespace)
		categoryCounts[category]++
	}

	// Convert to slice and sort
	categories := []types.TrafficCategory{}
	for category, count := range categoryCounts {
		description := categoryDescriptions[category]
		if description == "" {
			description = "Uncategorized traffic"
		}
		categories = append(categories, types.TrafficCategory{
			Category:    category,
			Count:       count,
			Description: description,
		})
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Count > categories[j].Count
	})

	return categories
}
