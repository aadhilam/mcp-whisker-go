package whisker

import (
	"context"
	"fmt"
	"strings"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// BlockedFlowAnalyzer analyzes blocked network flows and identifies blocking policies
type BlockedFlowAnalyzer struct {
	policyAnalyzer *PolicyAnalyzer
}

// NewBlockedFlowAnalyzer creates a new BlockedFlowAnalyzer instance
func NewBlockedFlowAnalyzer(policyAnalyzer *PolicyAnalyzer) *BlockedFlowAnalyzer {
	return &BlockedFlowAnalyzer{
		policyAnalyzer: policyAnalyzer,
	}
}

// AnalyzeBlockedFlows performs comprehensive analysis of blocked flows
func (b *BlockedFlowAnalyzer) AnalyzeBlockedFlows(ctx context.Context, namespace string, blockedLogs []types.FlowLog) *types.BlockedFlowAnalysis {
	uniqueConnections := make(map[string]bool)
	blockedFlowDetails := make([]types.BlockedFlowDetail, 0, len(blockedLogs))
	currentlyBlockedCount := 0
	stagedBlockCount := 0

	for _, log := range blockedLogs {
		connectionKey := fmt.Sprintf("%s→%s:%d", log.SourceName, log.DestName, log.DestPort)
		uniqueConnections[connectionKey] = true

		blockingPolicies := b.extractBlockingPolicies(ctx, &log)

		// Determine block status
		blockStatus := "currently_blocked"
		if !strings.EqualFold(log.Action, "Deny") {
			// If action is not Deny but we're analyzing it, it must have pending deny policies
			blockStatus = "staged_block"
			stagedBlockCount++
		} else {
			currentlyBlockedCount++
		}

		detail := types.BlockedFlowDetail{
			Flow: types.BlockedFlowInfo{
				Source:      fmt.Sprintf("%s (%s)", log.SourceName, log.SourceNamespace),
				Destination: fmt.Sprintf("%s (%s)", log.DestName, log.DestNamespace),
				Protocol:    log.Protocol,
				Port:        log.DestPort,
				Action:      log.Action,
				Reporter:    log.Reporter,
				TimeRange:   fmt.Sprintf("%s to %s", log.StartTime, log.EndTime),
				BlockStatus: blockStatus,
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
				Recommendation:        b.generateRecommendation(blockingPolicies),
			},
		}

		blockedFlowDetails = append(blockedFlowDetails, detail)
	}

	// Generate appropriate message based on block status
	message := fmt.Sprintf("🚨 %d blocked flow(s) detected", len(blockedLogs))
	if stagedBlockCount > 0 {
		message = fmt.Sprintf("🚨 %d flow(s) analyzed: %d currently blocked, %d would be blocked by staged policies",
			len(blockedLogs), currentlyBlockedCount, stagedBlockCount)
	}

	recommendations := []string{
		"Review each blocking policy to ensure it aligns with your security requirements",
		"Consider if any blocked flows represent legitimate traffic that should be allowed",
		"Verify that policy ordering and tier configuration are correct",
		"Monitor for patterns that might indicate security threats or misconfigurations",
	}

	// Add staged policy specific recommendation if applicable
	if stagedBlockCount > 0 {
		recommendations = append([]string{
			"⚠️  STAGED POLICIES DETECTED: Some flows are currently allowed but would be blocked if staged policies are enforced",
			"Review staged policies before promoting them to ensure they don't break legitimate traffic",
		}, recommendations...)
	}

	return &types.BlockedFlowAnalysis{
		Namespace: namespace,
		Analysis: types.BlockedFlowAnalysisInfo{
			TotalBlockedFlows:        len(blockedLogs),
			UniqueBlockedConnections: len(uniqueConnections),
		},
		BlockedFlows: blockedFlowDetails,
		SecurityInsights: types.SecurityInsights{
			Message:         message,
			Recommendations: recommendations,
		},
	}
}

// extractBlockingPolicies extracts policies that blocked the flow
func (b *BlockedFlowAnalyzer) extractBlockingPolicies(ctx context.Context, log *types.FlowLog) []types.BlockingPolicy {
	return b.policyAnalyzer.ExtractBlockingPolicies(ctx, log)
}

// generateRecommendation generates recommendations for handling blocked flows
func (b *BlockedFlowAnalyzer) generateRecommendation(blockingPolicies []types.BlockingPolicy) string {
	return b.policyAnalyzer.GenerateRecommendation(blockingPolicies)
}
