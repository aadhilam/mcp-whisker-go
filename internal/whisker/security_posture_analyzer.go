package whisker

import (
	"fmt"
	"sort"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// SecurityPostureAnalyzer analyzes overall security posture from flow logs
type SecurityPostureAnalyzer struct{}

// NewSecurityPostureAnalyzer creates a new SecurityPostureAnalyzer instance
func NewSecurityPostureAnalyzer() *SecurityPostureAnalyzer {
	return &SecurityPostureAnalyzer{}
}

// CalculateSecurityPosture analyzes overall security posture including flow statistics and policy usage
func (sp *SecurityPostureAnalyzer) CalculateSecurityPosture(logs []types.FlowLog) types.SecurityPostureInfo {
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
