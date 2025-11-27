package whisker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// PolicyAnalyzer handles policy analysis, conversion, and kubectl interactions
type PolicyAnalyzer struct {
	kubeconfigPath string
}

// NewPolicyAnalyzer creates a new policy analyzer
func NewPolicyAnalyzer(kubeconfigPath string) *PolicyAnalyzer {
	return &PolicyAnalyzer{
		kubeconfigPath: kubeconfigPath,
	}
}

// ConvertPolicyToDetail converts a Policy to PolicyDetail, preserving trigger chains
func (p *PolicyAnalyzer) ConvertPolicyToDetail(policy *types.Policy) types.PolicyDetail {
	detail := types.PolicyDetail{
		Name:        policy.Name,
		Namespace:   policy.Namespace,
		Kind:        policy.Kind,
		Tier:        policy.Tier,
		Action:      policy.Action,
		PolicyIndex: policy.PolicyIndex,
		RuleIndex:   policy.RuleIndex,
	}

	// Recursively convert trigger if present
	if policy.Trigger != nil {
		triggerDetail := p.ConvertPolicyToDetail(policy.Trigger)
		detail.Trigger = &triggerDetail
	}

	return detail
}

// AggregatePolicies processes and aggregates enforced and pending policies from a flow log
func (p *PolicyAnalyzer) AggregatePolicies(
	enforcedPolicies *[]types.PolicyDetail,
	pendingPolicies *[]types.PolicyDetail,
	sourcePolicies map[string]bool,
	destPolicies map[string]bool,
	log *types.FlowLog,
) {
	// Process enforced policies
	for _, policy := range log.Policies.Enforced {
		policyDetail := p.ConvertPolicyToDetail(&policy)
		*enforcedPolicies = append(*enforcedPolicies, policyDetail)

		policyName := fmt.Sprintf("%s (%s)", policy.Name, policy.Namespace)
		if log.Reporter == "Src" {
			sourcePolicies[policyName] = true
		} else if log.Reporter == "Dst" {
			destPolicies[policyName] = true
		}
	}

	// Process pending policies
	for _, policy := range log.Policies.Pending {
		policyDetail := p.ConvertPolicyToDetail(&policy)
		*pendingPolicies = append(*pendingPolicies, policyDetail)
	}
}

// ExtractBlockingPolicies identifies and extracts blocking policies from a flow log
// Returns both the main policy and the trigger policy responsible for the block
// Also retrieves the actual policy YAML from the cluster for both policies
func (p *PolicyAnalyzer) ExtractBlockingPolicies(ctx context.Context, log *types.FlowLog) []types.BlockingPolicy {
	blockingPolicies := []types.BlockingPolicy{}

	// Check pending policies first (staged policies that would block)
	for _, policy := range log.Policies.Pending {
		if strings.EqualFold(policy.Action, "Deny") {
			// Direct deny in pending policy
			policyDetail := p.ConvertPolicyToDetail(&policy)

			blockingPolicy := types.BlockingPolicy{
				Policy:         &policyDetail,
				BlockingReason: p.GetBlockingReason(policy.Action),
			}

			// Check if there's a trigger - even deny policies can have triggers
			if policy.Trigger != nil {
				triggerDetail := p.ConvertPolicyToDetail(policy.Trigger)
				blockingPolicy.TriggerPolicy = &triggerDetail

				// Retrieve YAML for the trigger policy
				if triggerYaml := p.RetrievePolicyDetails(ctx, policy.Trigger); triggerYaml != nil {
					blockingPolicy.TriggerPolicyYAML = triggerYaml
				}
			}

			// Retrieve YAML for the main policy from the cluster
			if yamlDetails := p.RetrievePolicyDetails(ctx, &policy); yamlDetails != nil {
				blockingPolicy.PolicyYAML = yamlDetails
			}

			blockingPolicies = append(blockingPolicies, blockingPolicy)
		} else if policy.Trigger != nil && strings.EqualFold(policy.Trigger.Action, "Deny") {
			// Trigger causes the deny
			policyDetail := p.ConvertPolicyToDetail(&policy)
			triggerDetail := p.ConvertPolicyToDetail(policy.Trigger)

			blockingPolicy := types.BlockingPolicy{
				Policy:         &policyDetail,
				TriggerPolicy:  &triggerDetail,
				BlockingReason: p.GetBlockingReason(policy.Trigger.Action),
			}

			// Retrieve YAML for both the main policy and the trigger policy
			if mainYaml := p.RetrievePolicyDetails(ctx, &policy); mainYaml != nil {
				blockingPolicy.PolicyYAML = mainYaml
			}
			if triggerYaml := p.RetrievePolicyDetails(ctx, policy.Trigger); triggerYaml != nil {
				blockingPolicy.TriggerPolicyYAML = triggerYaml
			}

			blockingPolicies = append(blockingPolicies, blockingPolicy)
		}
	}

	// Check enforced policies
	for _, policy := range log.Policies.Enforced {
		if strings.EqualFold(policy.Action, "Deny") {
			// Direct deny in enforced policy
			policyDetail := p.ConvertPolicyToDetail(&policy)

			blockingPolicy := types.BlockingPolicy{
				Policy:         &policyDetail,
				BlockingReason: p.GetBlockingReason(policy.Action),
			}

			// Check if there's a trigger - even deny policies can have triggers
			if policy.Trigger != nil {
				triggerDetail := p.ConvertPolicyToDetail(policy.Trigger)
				blockingPolicy.TriggerPolicy = &triggerDetail

				// Retrieve YAML for the trigger policy
				if triggerYaml := p.RetrievePolicyDetails(ctx, policy.Trigger); triggerYaml != nil {
					blockingPolicy.TriggerPolicyYAML = triggerYaml
				}
			}

			// Retrieve YAML for the main policy from the cluster
			if yamlDetails := p.RetrievePolicyDetails(ctx, &policy); yamlDetails != nil {
				blockingPolicy.PolicyYAML = yamlDetails
			}

			blockingPolicies = append(blockingPolicies, blockingPolicy)
		} else if policy.Trigger != nil && strings.EqualFold(policy.Trigger.Action, "Deny") {
			// Trigger causes the deny
			policyDetail := p.ConvertPolicyToDetail(&policy)
			triggerDetail := p.ConvertPolicyToDetail(policy.Trigger)

			blockingPolicy := types.BlockingPolicy{
				Policy:         &policyDetail,
				TriggerPolicy:  &triggerDetail,
				BlockingReason: p.GetBlockingReason(policy.Trigger.Action),
			}

			// Retrieve YAML for both the main policy and the trigger policy
			if mainYaml := p.RetrievePolicyDetails(ctx, &policy); mainYaml != nil {
				blockingPolicy.PolicyYAML = mainYaml
			}
			if triggerYaml := p.RetrievePolicyDetails(ctx, policy.Trigger); triggerYaml != nil {
				blockingPolicy.TriggerPolicyYAML = triggerYaml
			}

			blockingPolicies = append(blockingPolicies, blockingPolicy)
		}
	}

	return blockingPolicies
}

// RetrievePolicyDetails fetches policy YAML details using kubectl
func (p *PolicyAnalyzer) RetrievePolicyDetails(ctx context.Context, policy *types.Policy) *string {
	if policy == nil {
		return nil
	}

	// Special case: EndOfTier is not a real policy object in the cluster
	if policy.Kind == "EndOfTier" || policy.Kind == "Profile" {
		// These are system-generated markers, not actual policy objects
		explanation := fmt.Sprintf("# %s Policy\n# This is a system-generated %s marker, not a retrievable cluster resource.\n", policy.Kind, policy.Kind)
		if policy.Kind == "EndOfTier" {
			explanation += fmt.Sprintf("# Tier: %s\n# Action: %s\n# This represents the default action at the end of tier '%s'.\n", policy.Tier, policy.Action, policy.Tier)
		} else if policy.Kind == "Profile" {
			explanation += fmt.Sprintf("# Name: %s\n# This represents a Calico profile for the workload.\n", policy.Name)
		}
		return &explanation
	}

	resourceType := p.MapPolicyKindToResource(policy.Kind)
	if resourceType == "" {
		errMsg := fmt.Sprintf("# Unknown policy kind: %s\n# Unable to retrieve policy details from cluster.\n", policy.Kind)
		return &errMsg
	}

	args := []string{"get", resourceType, policy.Name, "-o", "yaml"}

	// Add namespace if specified and not a global policy
	if policy.Namespace != "" && policy.Kind != "GlobalNetworkPolicy" {
		args = append(args, "-n", policy.Namespace)
	}

	// Add kubeconfig if specified
	if p.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", p.kubeconfigPath}, args...)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.Output()
	if err != nil {
		// Return error message instead of nil
		errMsg := fmt.Sprintf("# Error retrieving policy %s/%s\n# kubectl error: %v\n", policy.Namespace, policy.Name, err)
		return &errMsg
	}

	result := strings.TrimSpace(string(output))
	return &result
}

// MapPolicyKindToResource maps policy kind to kubectl resource type
func (p *PolicyAnalyzer) MapPolicyKindToResource(kind string) string {
	switch kind {
	case "CalicoNetworkPolicy":
		return "caliconetworkpolicy"
	case "NetworkPolicy":
		return "networkpolicy"
	case "GlobalNetworkPolicy":
		return "globalnetworkpolicy"
	case "StagedNetworkPolicy":
		return "stagednetworkpolicy"
	case "StagedGlobalNetworkPolicy":
		return "stagedglobalnetworkpolicy"
	case "StagedKubernetesNetworkPolicy":
		return "stagedkubernetesnetworkpolicy"
	default:
		return ""
	}
}

// GetBlockingReason returns a human-readable reason for why traffic was blocked
func (p *PolicyAnalyzer) GetBlockingReason(action string) string {
	if strings.EqualFold(action, "Deny") {
		return "Explicit deny rule"
	}
	return "End of tier default deny"
}

// GenerateRecommendation generates a recommendation based on blocking policies
func (p *PolicyAnalyzer) GenerateRecommendation(blockingPolicies []types.BlockingPolicy) string {
	if len(blockingPolicies) > 0 {
		return "Review the identified policies to understand why traffic is being blocked. Consider modifying the policy rules if this traffic should be allowed."
	}
	return "No specific blocking policies identified. This may be due to default deny behavior or policy ordering."
}
