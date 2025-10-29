package whisker

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// normalizePodName detects pod patterns and adds wildcards
// Example: coredns-789465848c-abc123 -> coredns-789465848c-*
func normalizePodName(name string) string {
	if name == "" {
		return name
	}

	// Match patterns like: coredns-789465848c-abc123 (ReplicaSet pods)
	// This matches: name-hash-podid where hash is 8-10 chars and podid is 5 chars
	replicaSetPattern := regexp.MustCompile(`^(.+-[a-z0-9]{8,10})-[a-z0-9]{5,6}$`)
	if matches := replicaSetPattern.FindStringSubmatch(name); len(matches) > 1 {
		return matches[1] + "-*"
	}

	// Match patterns like: coredns-abc123 (Deployment pods without ReplicaSet hash)
	deploymentPattern := regexp.MustCompile(`^(.+)-[a-z0-9]{5}$`)
	if matches := deploymentPattern.FindStringSubmatch(name); len(matches) > 1 {
		// Only apply if the name looks like a pod (contains a dash)
		if strings.Contains(matches[1], "-") {
			return matches[1] + "-*"
		}
	}

	return name
}

// classifyNetwork determines if a name represents a private network, public network, or specific entity
func classifyNetwork(name, namespace string) string {
	if name == "" {
		return "PRIVATE NETWORK"
	}

	// If namespace is empty and name looks like an IP or is empty, it's likely private network
	if namespace == "" {
		if isPrivateIP(name) || name == "" {
			return "PRIVATE NETWORK"
		}
		if isPublicIP(name) || isExternalDomain(name) {
			return "PUBLIC NETWORK"
		}
	}

	// Check if it's a public IP or external domain
	if isPublicIP(name) || isExternalDomain(name) {
		return "PUBLIC NETWORK"
	}

	return name
}

// isPrivateIP checks if the given string is a private IP address (RFC1918)
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for private IP ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8", // localhost
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// isPublicIP checks if the given string is a public IP address
func isPublicIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// If it's an IP and not private, it's public
	return !isPrivateIP(ipStr)
}

// isExternalDomain checks if the name looks like an external domain
func isExternalDomain(name string) bool {
	// Simple check: contains dots and looks like a domain
	return strings.Contains(name, ".") && !strings.HasPrefix(name, "10.") && !strings.HasPrefix(name, "192.168.")
}

// categorizeTraffic categorizes a flow based on its characteristics
func categorizeTraffic(protocol string, port int, destNamespace string) string {
	// DNS Queries
	if port == 53 {
		return "DNS Queries"
	}

	// API/HTTPS
	if port == 443 && protocol == "TCP" {
		return "API/HTTPS"
	}

	// Metrics Collection
	if port == 10250 || port == 4443 {
		return "Metrics Collection"
	}

	// Calico Services
	if destNamespace == "calico-system" || destNamespace == "calico-apiserver" {
		return "Calico Services"
	}

	// Monitoring
	if port == 9153 {
		return "Monitoring"
	}

	// HTTP
	if port == 80 || port == 8080 {
		return "HTTP"
	}

	// Database
	if port == 3306 || port == 5432 || port == 27017 || port == 6379 {
		return "Database"
	}

	// Default
	return "Other"
}

// formatBytes converts bytes to human-readable format (KB, MB, GB)
func formatBytes(bytes int64) string {
	const unit = 1000
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	// Calculate with one decimal place
	value := float64(bytes) / float64(div)

	switch exp {
	case 0:
		return fmt.Sprintf("%.1fKB", value)
	case 1:
		return fmt.Sprintf("%.1fMB", value)
	case 2:
		return fmt.Sprintf("%.1fGB", value)
	default:
		return fmt.Sprintf("%.1fTB", value)
	}
}

// formatPackets formats packet counts with K/M suffixes if needed
func formatPackets(packets int64) string {
	if packets < 1000 {
		return fmt.Sprintf("%d", packets)
	}
	if packets < 1000000 {
		return fmt.Sprintf("%.1fK", float64(packets)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(packets)/1000000)
}

// extractPrimaryActivity determines the primary activity for an entity based on its flows
func extractPrimaryActivity(flows []types.FlowLog) string {
	if len(flows) == 0 {
		return ""
	}

	// Count different activity types
	activityCounts := make(map[string]int)

	for _, flow := range flows {
		category := categorizeTraffic(flow.Protocol, flow.DestPort, flow.DestNamespace)
		activityCounts[category]++
	}

	// Find the most common activities
	activities := []string{}
	maxCount := 0

	for _, count := range activityCounts {
		if count > maxCount {
			maxCount = count
		}
	}

	// Collect activities that are significant (at least 20% of max)
	threshold := maxCount / 5
	for activity, count := range activityCounts {
		if count >= threshold && activity != "Other" {
			activities = append(activities, strings.ToLower(activity))
		}
	}

	if len(activities) == 0 {
		return "Various activities"
	}

	// Build activity string
	if len(activities) > 3 {
		return strings.Join(activities[:3], ", ")
	}

	return strings.Join(activities, ", ")
}

// normalizeEntityName normalizes both pod name and network classification
func normalizeEntityName(name, namespace string) string {
	// First check if it should be classified as a network
	classified := classifyNetwork(name, namespace)
	if classified == "PRIVATE NETWORK" || classified == "PUBLIC NETWORK" {
		return classified
	}

	// Otherwise normalize the pod name
	return normalizePodName(name)
}

// getPrimaryPolicy extracts the most commonly applied policy from a list of policies
func getPrimaryPolicy(policies []types.Policy) string {
	if len(policies) == 0 {
		return "-"
	}

	// Count policy occurrences
	policyCounts := make(map[string]int)
	for _, policy := range policies {
		policyName := policy.Name
		if policy.Namespace != "" {
			policyName = policy.Namespace + "." + policyName
		}
		policyCounts[policyName]++
	}

	// Find the most common policy
	maxCount := 0
	primaryPolicy := ""
	for policy, count := range policyCounts {
		if count > maxCount {
			maxCount = count
			primaryPolicy = policy
		}
	}

	if primaryPolicy == "" {
		return "-"
	}

	return primaryPolicy
}
