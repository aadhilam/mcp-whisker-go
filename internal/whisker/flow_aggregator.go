package whisker

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// FlowAggregator handles flow aggregation and summary generation
type FlowAggregator struct {
	policyAnalyzer *PolicyAnalyzer
}

// NewFlowAggregator creates a new FlowAggregator
func NewFlowAggregator(policyAnalyzer *PolicyAnalyzer) *FlowAggregator {
	return &FlowAggregator{
		policyAnalyzer: policyAnalyzer,
	}
}

// aggregatedFlow is an internal type for tracking aggregated flow data
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

// GenerateFlowSummary generates a comprehensive namespace flow summary
func (fa *FlowAggregator) GenerateFlowSummary(namespace string, logs []types.FlowLog) *types.NamespaceFlowSummary {
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
			fa.aggregatePolicies(existing, &log)
			fa.updateActions(existing, &log)
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

			fa.aggregatePolicies(flow, &log)
			fa.updateActions(flow, &log)
			flowMap[flowKey] = flow
		}
	}

	// Convert to FlowSummary slice
	flows := make([]types.FlowSummary, 0, len(flowMap))
	totalPackets := int64(0)
	totalBytes := int64(0)
	blockedCount := 0

	for _, flow := range flowMap {
		summary := fa.convertToFlowSummary(flow)
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

// convertToFlowSummary converts an aggregatedFlow to a FlowSummary
func (fa *FlowAggregator) convertToFlowSummary(flow *aggregatedFlow) types.FlowSummary {
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
			Action:          formatAction(flow.sourceAction),
			Policies:        sourcePolicies,
			PendingPolicies: pendingPolicyNames,
		},
		Destination: types.FlowEndpoint{
			Name:            flow.destination,
			Namespace:       flow.destNamespace,
			Action:          formatAction(flow.destAction),
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

// AggregateFlows groups and aggregates flow logs by connection
func (fa *FlowAggregator) AggregateFlows(logs []types.FlowLog) []types.AggregatedFlowEntry {
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

// aggregatePolicies aggregates policies from a log into a flow
func (fa *FlowAggregator) aggregatePolicies(flow *aggregatedFlow, log *types.FlowLog) {
	fa.policyAnalyzer.AggregatePolicies(
		&flow.enforcedPolicies,
		&flow.pendingPolicies,
		flow.sourcePolicies,
		flow.destPolicies,
		log,
	)
}

// updateActions updates the source and destination actions based on reporter
func (fa *FlowAggregator) updateActions(flow *aggregatedFlow, log *types.FlowLog) {
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

// formatAction formats an action with an emoji
func formatAction(action string) string {
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
