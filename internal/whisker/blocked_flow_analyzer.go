package whisker

import (
	"context"
	"fmt"

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

	for _, log := range blockedLogs {
		connectionKey := fmt.Sprintf("%s→%s:%d", log.SourceName, log.DestName, log.DestPort)
		uniqueConnections[connectionKey] = true

		blockingPolicies := b.extractBlockingPolicies(ctx, &log)

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
				Recommendation:        b.generateRecommendation(blockingPolicies),
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

// extractBlockingPolicies extracts policies that blocked the flow
func (b *BlockedFlowAnalyzer) extractBlockingPolicies(ctx context.Context, log *types.FlowLog) []types.BlockingPolicy {
	return b.policyAnalyzer.ExtractBlockingPolicies(ctx, log)
}

// generateRecommendation generates recommendations for handling blocked flows
func (b *BlockedFlowAnalyzer) generateRecommendation(blockingPolicies []types.BlockingPolicy) string {
	return b.policyAnalyzer.GenerateRecommendation(blockingPolicies)
}
