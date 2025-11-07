package whisker

import (
	"context"
	"fmt"
	"sort"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// Service provides access to Calico Whisker flow logs
type Service struct {
	httpClient     *HTTPClient
	policyAnalyzer *PolicyAnalyzer
	analytics      *Analytics
	flowAggregator *FlowAggregator
	kubeconfigPath string
}

// NewService creates a new Whisker service client
func NewService(kubeconfigPath string) *Service {
	policyAnalyzer := NewPolicyAnalyzer(kubeconfigPath)
	return &Service{
		httpClient:     NewHTTPClient(),
		policyAnalyzer: policyAnalyzer,
		analytics:      NewAnalytics(),
		flowAggregator: NewFlowAggregator(policyAnalyzer),
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

// generateFlowSummary generates a comprehensive namespace flow summary (delegates to FlowAggregator)
func (s *Service) generateFlowSummary(namespace string, logs []types.FlowLog) *types.NamespaceFlowSummary {
	return s.flowAggregator.GenerateFlowSummary(namespace, logs)
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

// convertPolicyToDetail converts a Policy to PolicyDetail (delegates to PolicyAnalyzer)
func (s *Service) convertPolicyToDetail(policy *types.Policy) types.PolicyDetail {
	return s.policyAnalyzer.ConvertPolicyToDetail(policy)
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

// determineTimeRange extracts the time range from flow logs (delegates to Analytics)
func (s *Service) determineTimeRange(logs []types.FlowLog) string {
	return s.analytics.DetermineTimeRange(logs)
}

// aggregateFlows groups and aggregates flow logs by connection (delegates to FlowAggregator)
func (s *Service) aggregateFlows(logs []types.FlowLog) []types.AggregatedFlowEntry {
	return s.flowAggregator.AggregateFlows(logs)
}

// categorizeFlows categorizes flows and counts them (delegates to Analytics)
func (s *Service) categorizeFlows(logs []types.FlowLog) []types.TrafficCategory {
	return s.analytics.CategorizeFlows(logs)
}

// calculateTopSources identifies and ranks top traffic sources (delegates to Analytics)
func (s *Service) calculateTopSources(logs []types.FlowLog) []types.TopTrafficEntity {
	return s.analytics.CalculateTopSources(logs)
}

// calculateTopDestinations identifies and ranks top traffic destinations (delegates to Analytics)
func (s *Service) calculateTopDestinations(logs []types.FlowLog) []types.TopTrafficEntity {
	return s.analytics.CalculateTopDestinations(logs)
}

// analyzeNamespaceActivity analyzes traffic by namespace (delegates to Analytics)
func (s *Service) analyzeNamespaceActivity(logs []types.FlowLog) []types.NamespaceActivityInfo {
	return s.analytics.AnalyzeNamespaceActivity(logs)
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
