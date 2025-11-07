package whisker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// Service provides access to Calico Whisker flow logs
type Service struct {
	httpClient     *HTTPClient
	policyAnalyzer *PolicyAnalyzer
	kubeconfigPath string
}

// NewService creates a new Whisker service client
func NewService(kubeconfigPath string) *Service {
	return &Service{
		httpClient:     NewHTTPClient(),
		policyAnalyzer: NewPolicyAnalyzer(kubeconfigPath),
		kubeconfigPath: kubeconfigPath,
	}
}

// GetFlowLogs retrieves flow logs from Whisker service (delegates to HTTPClient)
func (s *Service) GetFlowLogs(ctx context.Context) ([]types.FlowLog, error) {
	return s.httpClient.GetFlowLogs(ctx)
}

// GetNamespaceFlowSummary generates detailed flow analysis for a specific namespace
func (s *Service) GetNamespaceFlowSummary(ctx context.Context, namespace string) (*types.NamespaceFlowSummary, error) {
	allLogs, err := s.GetFlowLogs(ctx)
	if err != nil {
		return nil, err
	}

	// Filter logs for the specified namespace
	namespaceLogs := make([]types.FlowLog, 0)
	for _, log := range allLogs {
		if log.SourceNamespace == namespace || log.DestNamespace == namespace {
			namespaceLogs = append(namespaceLogs, log)
		}
	}

	if len(namespaceLogs) == 0 {
		return &types.NamespaceFlowSummary{
			Namespace: namespace,
			Analysis: types.AnalysisInfo{
				TotalUniqueFlows: 0,
				TotalLogEntries:  0,
			},
			Flows: []types.FlowSummary{},
		}, nil
	}

	return s.generateFlowSummary(namespace, namespaceLogs), nil
}

// AnalyzeBlockedFlows analyzes blocked flows in the specified namespace
func (s *Service) AnalyzeBlockedFlows(ctx context.Context, namespace string) (*types.BlockedFlowAnalysis, error) {
	allLogs, err := s.GetFlowLogs(ctx)
	if err != nil {
		return nil, err
	}

	// Filter for blocked flows
	blockedLogs := make([]types.FlowLog, 0)
	for _, log := range allLogs {
		if log.Action == "Deny" {
			if namespace == "" || log.SourceNamespace == namespace || log.DestNamespace == namespace {
				blockedLogs = append(blockedLogs, log)
			}
		}
	}

	if len(blockedLogs) == 0 {
		return &types.BlockedFlowAnalysis{
			Namespace: namespace,
			Analysis: types.BlockedFlowAnalysisInfo{
				TotalBlockedFlows:        0,
				UniqueBlockedConnections: 0,
			},
			BlockedFlows: []types.BlockedFlowDetail{},
			SecurityInsights: types.SecurityInsights{
				Message:         "No blocked flows found",
				Recommendations: []string{},
			},
		}, nil
	}

	return s.analyzeBlockedFlows(ctx, namespace, blockedLogs), nil
}

func (s *Service) generateFlowSummary(namespace string, logs []types.FlowLog) *types.NamespaceFlowSummary {
	flowMap := make(map[string]*aggregatedFlow)

	// Process each log and aggregate by flow
	for _, log := range logs {
		flowKey := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
			log.SourceName, log.SourceNamespace,
			log.DestName, log.DestNamespace,
			log.Protocol, log.DestPort, log.Action)

		if existing, exists := flowMap[flowKey]; exists {
			// Aggregate existing flow
			existing.packetsIn += log.PacketsIn
			existing.packetsOut += log.PacketsOut
			existing.bytesIn += log.BytesIn
			existing.bytesOut += log.BytesOut

			// Update time range
			if log.StartTime < existing.startTime {
				existing.startTime = log.StartTime
			}
			if log.EndTime > existing.endTime {
				existing.endTime = log.EndTime
			}

			// Aggregate policies
			s.aggregatePolicies(existing, &log)
			s.updateActions(existing, &log)
		} else {
			// Create new flow entry
			flow := &aggregatedFlow{
				source:           log.SourceName,
				sourceNamespace:  log.SourceNamespace,
				destination:      log.DestName,
				destNamespace:    log.DestNamespace,
				protocol:         log.Protocol,
				port:             log.DestPort,
				sourceAction:     "N/A",
				destAction:       "N/A",
				packetsIn:        log.PacketsIn,
				packetsOut:       log.PacketsOut,
				bytesIn:          log.BytesIn,
				bytesOut:         log.BytesOut,
				startTime:        log.StartTime,
				endTime:          log.EndTime,
				sourcePolicies:   make(map[string]bool),
				destPolicies:     make(map[string]bool),
				enforcedPolicies: []types.PolicyDetail{},
				pendingPolicies:  []types.PolicyDetail{},
			}

			s.aggregatePolicies(flow, &log)
			s.updateActions(flow, &log)
			flowMap[flowKey] = flow
		}
	}

	// Convert to FlowSummary slice
	flows := make([]types.FlowSummary, 0, len(flowMap))
	totalPackets := int64(0)
	totalBytes := int64(0)
	blockedCount := 0

	for _, flow := range flowMap {
		summary := s.convertToFlowSummary(flow)
		flows = append(flows, summary)

		totalPackets += summary.Traffic.Packets.Total
		totalBytes += summary.Traffic.Bytes.Total

		if strings.Contains(summary.Status, "BLOCKED") {
			blockedCount++
		}
	}

	// Sort flows by start time
	sort.Slice(flows, func(i, j int) bool {
		return flows[i].TimeRange.Start < flows[j].TimeRange.Start
	})

	// Calculate statistics
	var earliestTime, latestTime *string
	if len(flows) > 0 {
		earliestTime = &flows[0].TimeRange.Start
		latestTime = &flows[len(flows)-1].TimeRange.End
	}

	// Generate security alerts if there are blocked flows
	var securityAlerts *types.SecurityAlerts
	if blockedCount > 0 {
		blockedFlowNames := make([]string, 0, blockedCount)
		for _, flow := range flows {
			if strings.Contains(flow.Status, "BLOCKED") {
				blockedFlowNames = append(blockedFlowNames,
					fmt.Sprintf("%s → %s:%d", flow.Source.Name, flow.Destination.Name, flow.Connection.Port))
			}
		}

		securityAlerts = &types.SecurityAlerts{
			Message:      fmt.Sprintf("🚨 %d blocked flow(s) detected - immediate attention required!", blockedCount),
			BlockedFlows: blockedFlowNames,
		}
	}

	return &types.NamespaceFlowSummary{
		Namespace: namespace,
		Analysis: types.AnalysisInfo{
			TotalUniqueFlows: len(flowMap),
			TotalLogEntries:  len(logs),
			TimeWindow: types.TimeWindowInfo{
				Start: earliestTime,
				End:   latestTime,
			},
		},
		Statistics: types.StatisticsInfo{
			Flows: types.FlowStats{
				Total:   len(flows),
				Allowed: len(flows) - blockedCount,
				Blocked: blockedCount,
			},
			Traffic: types.TrafficStats{
				TotalPackets: totalPackets,
				TotalBytes:   totalBytes,
			},
		},
		Flows:          flows,
		SecurityAlerts: securityAlerts,
	}
}

func (s *Service) analyzeBlockedFlows(ctx context.Context, namespace string, blockedLogs []types.FlowLog) *types.BlockedFlowAnalysis {
	uniqueConnections := make(map[string]bool)
	blockedFlowDetails := make([]types.BlockedFlowDetail, 0, len(blockedLogs))

	for _, log := range blockedLogs {
		connectionKey := fmt.Sprintf("%s→%s:%d", log.SourceName, log.DestName, log.DestPort)
		uniqueConnections[connectionKey] = true

		blockingPolicies := s.extractBlockingPolicies(ctx, &log)

		detail := types.BlockedFlowDetail{
			Flow: types.BlockedFlowInfo{
				Source:      fmt.Sprintf("%s (%s)", log.SourceName, log.SourceNamespace),
				Destination: fmt.Sprintf("%s (%s)", log.DestName, log.DestNamespace),
				Protocol:    log.Protocol,
				Port:        log.DestPort,
				Action:      log.Action,
				Reporter:    log.Reporter,
				TimeRange:   fmt.Sprintf("%s to %s", log.StartTime, log.EndTime),
			},
			Traffic: types.TrafficInfo{
				Packets: types.TrafficMetric{
					In:    log.PacketsIn,
					Out:   log.PacketsOut,
					Total: log.PacketsIn + log.PacketsOut,
				},
				Bytes: types.TrafficMetric{
					In:    log.BytesIn,
					Out:   log.BytesOut,
					Total: log.BytesIn + log.BytesOut,
				},
			},
			BlockingPolicies: blockingPolicies,
			Analysis: types.FlowAnalysis{
				TotalBlockingPolicies: len(blockingPolicies),
				Recommendation:        s.generateRecommendation(blockingPolicies),
			},
		}

		blockedFlowDetails = append(blockedFlowDetails, detail)
	}

	return &types.BlockedFlowAnalysis{
		Namespace: namespace,
		Analysis: types.BlockedFlowAnalysisInfo{
			TotalBlockedFlows:        len(blockedLogs),
			UniqueBlockedConnections: len(uniqueConnections),
		},
		BlockedFlows: blockedFlowDetails,
		SecurityInsights: types.SecurityInsights{
			Message: fmt.Sprintf("🚨 %d blocked flow(s) detected", len(blockedLogs)),
			Recommendations: []string{
				"Review each blocking policy to ensure it aligns with your security requirements",
				"Consider if any blocked flows represent legitimate traffic that should be allowed",
				"Verify that policy ordering and tier configuration are correct",
				"Monitor for patterns that might indicate security threats or misconfigurations",
			},
		},
	}
}

func (s *Service) extractBlockingPolicies(ctx context.Context, log *types.FlowLog) []types.BlockingPolicy {
	return s.policyAnalyzer.ExtractBlockingPolicies(ctx, log)
}

func (s *Service) retrievePolicyDetails(ctx context.Context, policy *types.Policy) *string {
	return s.policyAnalyzer.RetrievePolicyDetails(ctx, policy)
}

func (s *Service) mapPolicyKindToResource(kind string) string {
	return s.policyAnalyzer.MapPolicyKindToResource(kind)
}

func (s *Service) getBlockingReason(action string) string {
	return s.policyAnalyzer.GetBlockingReason(action)
}

func (s *Service) generateRecommendation(blockingPolicies []types.BlockingPolicy) string {
	return s.policyAnalyzer.GenerateRecommendation(blockingPolicies)
}

// Helper types for aggregation
type aggregatedFlow struct {
	source           string
	sourceNamespace  string
	destination      string
	destNamespace    string
	protocol         string
	port             int
	sourceAction     string
	destAction       string
	packetsIn        int64
	packetsOut       int64
	bytesIn          int64
	bytesOut         int64
	startTime        string
	endTime          string
	sourcePolicies   map[string]bool
	destPolicies     map[string]bool
	enforcedPolicies []types.PolicyDetail
	pendingPolicies  []types.PolicyDetail
}

// convertPolicyToDetail converts a Policy to PolicyDetail (delegates to PolicyAnalyzer)
func (s *Service) convertPolicyToDetail(policy *types.Policy) types.PolicyDetail {
	return s.policyAnalyzer.ConvertPolicyToDetail(policy)
}

func (s *Service) aggregatePolicies(flow *aggregatedFlow, log *types.FlowLog) {
	s.policyAnalyzer.AggregatePolicies(
		&flow.enforcedPolicies,
		&flow.pendingPolicies,
		flow.sourcePolicies,
		flow.destPolicies,
		log,
	)
}

func (s *Service) updateActions(flow *aggregatedFlow, log *types.FlowLog) {
	if log.Reporter == "Src" {
		if flow.sourceAction != "N/A" && flow.sourceAction != log.Action {
			flow.sourceAction = fmt.Sprintf("%s+%s", flow.sourceAction, log.Action)
		} else {
			flow.sourceAction = log.Action
		}
	} else if log.Reporter == "Dst" {
		if flow.destAction != "N/A" && flow.destAction != log.Action {
			flow.destAction = fmt.Sprintf("%s+%s", flow.destAction, log.Action)
		} else {
			flow.destAction = log.Action
		}
	}
}

func (s *Service) convertToFlowSummary(flow *aggregatedFlow) types.FlowSummary {
	// Convert maps to slices and sort
	sourcePolicies := make([]string, 0, len(flow.sourcePolicies))
	for policy := range flow.sourcePolicies {
		sourcePolicies = append(sourcePolicies, policy)
	}
	sort.Strings(sourcePolicies)

	destPolicies := make([]string, 0, len(flow.destPolicies))
	for policy := range flow.destPolicies {
		destPolicies = append(destPolicies, policy)
	}
	sort.Strings(destPolicies)

	uniquePolicies := make(map[string]bool)
	for _, policy := range flow.enforcedPolicies {
		uniquePolicies[fmt.Sprintf("%s (%s)", policy.Name, policy.Namespace)] = true
	}

	uniquePolicySlice := make([]string, 0, len(uniquePolicies))
	for policy := range uniquePolicies {
		uniquePolicySlice = append(uniquePolicySlice, policy)
	}
	sort.Strings(uniquePolicySlice)

	// Process pending policies for display
	pendingPolicyNames := make([]string, 0, len(flow.pendingPolicies))
	for _, policy := range flow.pendingPolicies {
		pendingPolicyNames = append(pendingPolicyNames, fmt.Sprintf("⏳ %s (%s)", policy.Name, policy.Namespace))
	}
	sort.Strings(pendingPolicyNames)

	status := "✅ ALLOWED"
	if flow.sourceAction == "Deny" || flow.destAction == "Deny" {
		status = "🚨 BLOCKED"
	}

	startTime, _ := time.Parse(time.RFC3339, flow.startTime)
	endTime, _ := time.Parse(time.RFC3339, flow.endTime)
	duration := endTime.Sub(startTime)

	return types.FlowSummary{
		Source: types.FlowEndpoint{
			Name:            flow.source,
			Namespace:       flow.sourceNamespace,
			Action:          s.formatAction(flow.sourceAction),
			Policies:        sourcePolicies,
			PendingPolicies: pendingPolicyNames,
		},
		Destination: types.FlowEndpoint{
			Name:            flow.destination,
			Namespace:       flow.destNamespace,
			Action:          s.formatAction(flow.destAction),
			Policies:        destPolicies,
			PendingPolicies: pendingPolicyNames,
		},
		Connection: types.ConnectionInfo{
			Protocol: flow.protocol,
			Port:     flow.port,
		},
		Enforcement: types.EnforcementInfo{
			TotalPolicies:        len(flow.enforcedPolicies),
			UniquePolicies:       uniquePolicySlice,
			PolicyDetails:        flow.enforcedPolicies,
			TotalPendingPolicies: len(flow.pendingPolicies),
			PendingPolicyDetails: flow.pendingPolicies,
		},
		Traffic: types.TrafficInfo{
			Packets: types.TrafficMetric{
				In:    flow.packetsIn,
				Out:   flow.packetsOut,
				Total: flow.packetsIn + flow.packetsOut,
			},
			Bytes: types.TrafficMetric{
				In:    flow.bytesIn,
				Out:   flow.bytesOut,
				Total: flow.bytesIn + flow.bytesOut,
			},
		},
		TimeRange: types.TimeRangeInfo{
			Start:    flow.startTime,
			End:      flow.endTime,
			Duration: duration,
		},
		Status: status,
	}
}

func (s *Service) formatAction(action string) string {
	switch action {
	case "Allow":
		return "✅ Allow"
	case "Deny":
		return "🚨 Deny"
	case "N/A":
		return "❌ N/A"
	default:
		return action
	}
}

// GetAggregatedFlowReport generates a comprehensive aggregated flow analysis report
func (s *Service) GetAggregatedFlowReport(ctx context.Context, startTime, endTime *string) (*types.FlowAggregateReport, error) {
	// Fetch all flow logs
	allLogs, err := s.GetFlowLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch flow logs: %w", err)
	}

	if len(allLogs) == 0 {
		return &types.FlowAggregateReport{
			TimeRange:         "No data available",
			TrafficOverview:   []types.AggregatedFlowEntry{},
			TrafficByCategory: []types.TrafficCategory{},
			TopTrafficSources: []types.TopTrafficEntity{},
			TopTrafficDest:    []types.TopTrafficEntity{},
			NamespaceActivity: []types.NamespaceActivityInfo{},
			SecurityPosture: types.SecurityPostureInfo{
				TotalFlows:        0,
				AllowedFlows:      0,
				DeniedFlows:       0,
				UniquePolicyNames: []string{},
			},
		}, nil
	}

	// Filter by time range if provided (for future enhancement)
	filteredLogs := allLogs
	// TODO: Implement time filtering when needed

	// Determine time range
	timeRange := s.determineTimeRange(filteredLogs)

	// Aggregate flows
	aggregatedEntries := s.aggregateFlows(filteredLogs)

	// Categorize traffic
	trafficByCategory := s.categorizeFlows(filteredLogs)

	// Calculate top sources and destinations
	topSources := s.calculateTopSources(filteredLogs)
	topDestinations := s.calculateTopDestinations(filteredLogs)

	// Analyze namespace activity
	namespaceActivity := s.analyzeNamespaceActivity(filteredLogs)

	// Calculate security posture
	securityPosture := s.calculateSecurityPosture(filteredLogs)

	return &types.FlowAggregateReport{
		TimeRange:         timeRange,
		TrafficOverview:   aggregatedEntries,
		TrafficByCategory: trafficByCategory,
		TopTrafficSources: topSources,
		TopTrafficDest:    topDestinations,
		NamespaceActivity: namespaceActivity,
		SecurityPosture:   securityPosture,
	}, nil
}

// determineTimeRange extracts the time range from flow logs
func (s *Service) determineTimeRange(logs []types.FlowLog) string {
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

// aggregateFlows groups and aggregates flow logs by connection
func (s *Service) aggregateFlows(logs []types.FlowLog) []types.AggregatedFlowEntry {
	// Map to hold aggregated flows: key = source|dest|protocol|port|action
	flowMap := make(map[string]*types.AggregatedFlowEntry)

	for _, log := range logs {
		// Normalize names
		normalizedSource := normalizeEntityName(log.SourceName, log.SourceNamespace)
		normalizedDest := normalizeEntityName(log.DestName, log.DestNamespace)

		sourceNS := log.SourceNamespace
		if normalizedSource == "PRIVATE NETWORK" || normalizedSource == "PUBLIC NETWORK" {
			sourceNS = "-"
		}

		destNS := log.DestNamespace
		if normalizedDest == "PRIVATE NETWORK" || normalizedDest == "PUBLIC NETWORK" {
			destNS = "-"
		}

		// Create flow key
		flowKey := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
			normalizedSource, sourceNS, normalizedDest, destNS,
			log.Protocol, log.DestPort, log.Action)

		if existing, exists := flowMap[flowKey]; exists {
			// Aggregate metrics
			existing.PacketsIn += log.PacketsIn
			existing.PacketsOut += log.PacketsOut
			existing.BytesIn += log.BytesIn
			existing.BytesOut += log.BytesOut
		} else {
			// Create new entry
			entry := &types.AggregatedFlowEntry{
				Source:          normalizedSource,
				SourceNamespace: sourceNS,
				Destination:     normalizedDest,
				DestNamespace:   destNS,
				Protocol:        log.Protocol,
				Port:            log.DestPort,
				Action:          log.Action,
				PacketsIn:       log.PacketsIn,
				PacketsOut:      log.PacketsOut,
				BytesIn:         log.BytesIn,
				BytesOut:        log.BytesOut,
				PrimaryPolicy:   getPrimaryPolicy(log.Policies.Enforced),
			}
			flowMap[flowKey] = entry
		}
	}

	// Convert map to slice and format human-readable values
	entries := make([]types.AggregatedFlowEntry, 0, len(flowMap))
	for _, entry := range flowMap {
		entry.PacketsInStr = formatPackets(entry.PacketsIn)
		entry.PacketsOutStr = formatPackets(entry.PacketsOut)
		entry.BytesInStr = formatBytes(entry.BytesIn)
		entry.BytesOutStr = formatBytes(entry.BytesOut)
		entries = append(entries, *entry)
	}

	return entries
}

// categorizeFlows categorizes flows and counts them
func (s *Service) categorizeFlows(logs []types.FlowLog) []types.TrafficCategory {
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

	// Convert to sorted slice
	categories := []types.TrafficCategory{}
	for category, count := range categoryCounts {
		if count > 0 { // Only include categories with traffic
			description := categoryDescriptions[category]
			if description == "" {
				description = category
			}
			categories = append(categories, types.TrafficCategory{
				Category:    category,
				Count:       count,
				Description: description,
			})
		}
	}

	// Sort by count (descending)
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Count > categories[j].Count
	})

	return categories
}

// calculateTopSources identifies and ranks top traffic sources
func (s *Service) calculateTopSources(logs []types.FlowLog) []types.TopTrafficEntity {
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

// calculateTopDestinations identifies and ranks top traffic destinations
func (s *Service) calculateTopDestinations(logs []types.FlowLog) []types.TopTrafficEntity {
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

// analyzeNamespaceActivity analyzes traffic by namespace
func (s *Service) analyzeNamespaceActivity(logs []types.FlowLog) []types.NamespaceActivityInfo {
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

// calculateSecurityPosture analyzes overall security posture
func (s *Service) calculateSecurityPosture(logs []types.FlowLog) types.SecurityPostureInfo {
	totalFlows := len(logs)
	allowedFlows := 0
	deniedFlows := 0
	uniquePolicies := make(map[string]bool)
	uniquePendingPolicies := make(map[string]bool)

	for _, log := range logs {
		if log.Action == "Allow" {
			allowedFlows++
		} else if log.Action == "Deny" {
			deniedFlows++
		}

		// Collect unique enforced policies
		for _, policy := range log.Policies.Enforced {
			policyName := policy.Name
			if policy.Namespace != "" {
				policyName = fmt.Sprintf("%s.%s", policy.Namespace, policy.Name)
			}
			uniquePolicies[policyName] = true
		}

		// Collect unique pending policies
		for _, policy := range log.Policies.Pending {
			policyName := policy.Name
			if policy.Namespace != "" {
				policyName = fmt.Sprintf("%s.%s", policy.Namespace, policy.Name)
			}
			uniquePendingPolicies[policyName] = true
		}
	}

	// Calculate percentages
	allowedPercentage := 0.0
	deniedPercentage := 0.0
	if totalFlows > 0 {
		allowedPercentage = (float64(allowedFlows) / float64(totalFlows)) * 100
		deniedPercentage = (float64(deniedFlows) / float64(totalFlows)) * 100
	}

	// Convert policy map to sorted slice
	policyNames := []string{}
	for policy := range uniquePolicies {
		policyNames = append(policyNames, policy)
	}
	sort.Strings(policyNames)

	// Convert pending policy map to sorted slice
	pendingPolicyNames := []string{}
	for policy := range uniquePendingPolicies {
		pendingPolicyNames = append(pendingPolicyNames, policy)
	}
	sort.Strings(pendingPolicyNames)

	return types.SecurityPostureInfo{
		TotalFlows:               totalFlows,
		AllowedFlows:             allowedFlows,
		AllowedPercentage:        allowedPercentage,
		DeniedFlows:              deniedFlows,
		DeniedPercentage:         deniedPercentage,
		ActivePolicies:           len(uniquePolicies),
		UniquePolicyNames:        policyNames,
		PendingPolicies:          len(uniquePendingPolicies),
		UniquePendingPolicyNames: pendingPolicyNames,
	}
}
