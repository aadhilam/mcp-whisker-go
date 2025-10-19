package whisker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

const (
	defaultWhiskerURL      = "http://127.0.0.1:8081"
	defaultWhiskerEndpoint = "/whisker-backend/flows"
)

// Service provides access to Calico Whisker flow logs
type Service struct {
	baseURL        string
	endpoint       string
	client         *http.Client
	kubeconfigPath string
}

// NewService creates a new Whisker service client
func NewService(kubeconfigPath string) *Service {
	return &Service{
		baseURL:        defaultWhiskerURL,
		endpoint:       defaultWhiskerEndpoint,
		kubeconfigPath: kubeconfigPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetFlowLogs retrieves flow logs from Whisker service
func (s *Service) GetFlowLogs(ctx context.Context) ([]types.FlowLog, error) {
	url := s.baseURL + s.endpoint
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Calico Whisker. Please ensure port-forward is running: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("whisker service returned status %d", resp.StatusCode)
	}

	var response types.FlowLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Items, nil
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
				source:          log.SourceName,
				sourceNamespace: log.SourceNamespace,
				destination:     log.DestName,
				destNamespace:   log.DestNamespace,
				protocol:        log.Protocol,
				port:            log.DestPort,
				sourceAction:    "N/A",
				destAction:      "N/A",
				packetsIn:       log.PacketsIn,
				packetsOut:      log.PacketsOut,
				bytesIn:         log.BytesIn,
				bytesOut:        log.BytesOut,
				startTime:       log.StartTime,
				endTime:         log.EndTime,
				sourcePolicies:  make(map[string]bool),
				destPolicies:    make(map[string]bool),
				enforcedPolicies: []types.PolicyDetail{},
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
				Recommendation: s.generateRecommendation(blockingPolicies),
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
	blockingPolicies := make([]types.BlockingPolicy, 0)

	// Check pending policies for triggers
	for _, pendingPolicy := range log.Policies.Pending {
		if pendingPolicy.Trigger != nil && pendingPolicy.Trigger.Name != "" {
			policyYAML := s.retrievePolicyDetails(ctx, pendingPolicy.Trigger)
			
			blockingPolicy := types.BlockingPolicy{
				TriggerPolicy: pendingPolicy.Trigger,
				PolicyYAML:    policyYAML,
				BlockingReason: s.getBlockingReason(pendingPolicy.Action),
			}
			
			blockingPolicies = append(blockingPolicies, blockingPolicy)
		}
	}

	// Check enforced policies
	for _, enforcedPolicy := range log.Policies.Enforced {
		if enforcedPolicy.Action == "Deny" && enforcedPolicy.Trigger != nil && enforcedPolicy.Trigger.Name != "" {
			policyYAML := s.retrievePolicyDetails(ctx, enforcedPolicy.Trigger)
			
			blockingPolicy := types.BlockingPolicy{
				TriggerPolicy: enforcedPolicy.Trigger,
				PolicyYAML:    policyYAML,
				BlockingReason: "Enforced deny rule",
			}
			
			blockingPolicies = append(blockingPolicies, blockingPolicy)
		}
	}

	return blockingPolicies
}

func (s *Service) retrievePolicyDetails(ctx context.Context, policy *types.Policy) *string {
	if policy == nil {
		return nil
	}

	resourceType := s.mapPolicyKindToResource(policy.Kind)
	if resourceType == "" {
		return nil
	}

	args := []string{"get", resourceType, policy.Name, "-o", "yaml"}
	
	// Add namespace if specified and not a global policy
	if policy.Namespace != "" && policy.Kind != "GlobalNetworkPolicy" {
		args = append(args, "-n", policy.Namespace)
	}
	
	// Add kubeconfig if specified
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := strings.TrimSpace(string(output))
	return &result
}

func (s *Service) mapPolicyKindToResource(kind string) string {
	switch kind {
	case "CalicoNetworkPolicy":
		return "caliconetworkpolicy"
	case "NetworkPolicy":
		return "networkpolicy"
	case "GlobalNetworkPolicy":
		return "globalnetworkpolicy"
	default:
		return ""
	}
}

func (s *Service) getBlockingReason(action string) string {
	if action == "Deny" {
		return "Explicit deny rule"
	}
	return "End of tier default deny"
}

func (s *Service) generateRecommendation(blockingPolicies []types.BlockingPolicy) string {
	if len(blockingPolicies) > 0 {
		return "Review the identified policies to understand why traffic is being blocked. Consider modifying the policy rules if this traffic should be allowed."
	}
	return "No specific blocking policies identified. This may be due to default deny behavior or policy ordering."
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
}

func (s *Service) aggregatePolicies(flow *aggregatedFlow, log *types.FlowLog) {
	for _, policy := range log.Policies.Enforced {
		policyDetail := types.PolicyDetail{
			Name:        policy.Name,
			Namespace:   policy.Namespace,
			Kind:        policy.Kind,
			Tier:        policy.Tier,
			Action:      policy.Action,
			PolicyIndex: policy.PolicyIndex,
			RuleIndex:   policy.RuleIndex,
		}
		flow.enforcedPolicies = append(flow.enforcedPolicies, policyDetail)

		policyName := fmt.Sprintf("%s (%s)", policy.Name, policy.Namespace)
		if log.Reporter == "Src" {
			flow.sourcePolicies[policyName] = true
		} else if log.Reporter == "Dst" {
			flow.destPolicies[policyName] = true
		}
	}
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

	status := "✅ ALLOWED"
	if flow.sourceAction == "Deny" || flow.destAction == "Deny" {
		status = "🚨 BLOCKED"
	}

	startTime, _ := time.Parse(time.RFC3339, flow.startTime)
	endTime, _ := time.Parse(time.RFC3339, flow.endTime)
	duration := endTime.Sub(startTime)

	return types.FlowSummary{
		Source: types.FlowEndpoint{
			Name:      flow.source,
			Namespace: flow.sourceNamespace,
			Action:    s.formatAction(flow.sourceAction),
			Policies:  sourcePolicies,
		},
		Destination: types.FlowEndpoint{
			Name:      flow.destination,
			Namespace: flow.destNamespace,
			Action:    s.formatAction(flow.destAction),
			Policies:  destPolicies,
		},
		Connection: types.ConnectionInfo{
			Protocol: flow.protocol,
			Port:     flow.port,
		},
		Enforcement: types.EnforcementInfo{
			TotalPolicies:  len(flow.enforcedPolicies),
			UniquePolicies: uniquePolicySlice,
			PolicyDetails:  flow.enforcedPolicies,
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