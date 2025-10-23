package types

import "time"

// FlowLog represents a Calico Whisker flow log entry
type FlowLog struct {
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	Action          string   `json:"action"`
	SourceName      string   `json:"source_name"`
	SourceNamespace string   `json:"source_namespace"`
	SourceLabels    string   `json:"source_labels"`
	DestName        string   `json:"dest_name"`
	DestNamespace   string   `json:"dest_namespace"`
	DestLabels      string   `json:"dest_labels"`
	Protocol        string   `json:"protocol"`
	DestPort        int      `json:"dest_port"`
	Reporter        string   `json:"reporter"`
	Policies        Policies `json:"policies"`
	PacketsIn       int64    `json:"packets_in"`
	PacketsOut      int64    `json:"packets_out"`
	BytesIn         int64    `json:"bytes_in"`
	BytesOut        int64    `json:"bytes_out"`
}

// Policy represents a Calico network policy
type Policy struct {
	Kind        string  `json:"kind"`
	Name        string  `json:"name"`
	Namespace   string  `json:"namespace"`
	Tier        string  `json:"tier"`
	Action      string  `json:"action"`
	PolicyIndex int     `json:"policy_index"`
	RuleIndex   int     `json:"rule_index"`
	Trigger     *Policy `json:"trigger"`
}

// Policies represents the policy enforcement information
type Policies struct {
	Enforced []Policy `json:"enforced"`
	Pending  []Policy `json:"pending"`
}

// FlowLogsResponse represents the API response from Whisker
type FlowLogsResponse struct {
	Items []FlowLog `json:"items"`
}

// FlowSummary represents aggregated flow information
type FlowSummary struct {
	Source      FlowEndpoint    `json:"source"`
	Destination FlowEndpoint    `json:"destination"`
	Connection  ConnectionInfo  `json:"connection"`
	Enforcement EnforcementInfo `json:"enforcement"`
	Traffic     TrafficInfo     `json:"traffic"`
	TimeRange   TimeRangeInfo   `json:"timeRange"`
	Status      string          `json:"status"`
}

// FlowEndpoint represents source or destination information
type FlowEndpoint struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Action    string   `json:"action"`
	Policies  []string `json:"policies"`
}

// ConnectionInfo represents connection details
type ConnectionInfo struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}

// EnforcementInfo represents policy enforcement details
type EnforcementInfo struct {
	TotalPolicies  int            `json:"totalPolicies"`
	UniquePolicies []string       `json:"uniquePolicies"`
	PolicyDetails  []PolicyDetail `json:"policyDetails"`
}

// PolicyDetail represents detailed policy information
type PolicyDetail struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Kind        string `json:"kind"`
	Tier        string `json:"tier"`
	Action      string `json:"action"`
	PolicyIndex int    `json:"policyIndex"`
	RuleIndex   int    `json:"ruleIndex"`
}

// TrafficInfo represents traffic statistics
type TrafficInfo struct {
	Packets TrafficMetric `json:"packets"`
	Bytes   TrafficMetric `json:"bytes"`
}

// TrafficMetric represents in/out/total metrics
type TrafficMetric struct {
	In    int64 `json:"in"`
	Out   int64 `json:"out"`
	Total int64 `json:"total"`
}

// TimeRangeInfo represents time range information
type TimeRangeInfo struct {
	Start    string        `json:"start"`
	End      string        `json:"end"`
	Duration time.Duration `json:"duration"`
}

// NamespaceFlowSummary represents the complete namespace analysis
type NamespaceFlowSummary struct {
	Namespace      string          `json:"namespace"`
	Analysis       AnalysisInfo    `json:"analysis"`
	Statistics     StatisticsInfo  `json:"statistics"`
	Flows          []FlowSummary   `json:"flows"`
	SecurityAlerts *SecurityAlerts `json:"securityAlerts,omitempty"`
}

// AnalysisInfo represents analysis metadata
type AnalysisInfo struct {
	TotalUniqueFlows int            `json:"totalUniqueFlows"`
	TotalLogEntries  int            `json:"totalLogEntries"`
	TimeWindow       TimeWindowInfo `json:"timeWindow"`
}

// TimeWindowInfo represents the analysis time window
type TimeWindowInfo struct {
	Start    *string        `json:"start,omitempty"`
	End      *string        `json:"end,omitempty"`
	Duration *time.Duration `json:"duration,omitempty"`
}

// StatisticsInfo represents flow statistics
type StatisticsInfo struct {
	Flows    FlowStats    `json:"flows"`
	Traffic  TrafficStats `json:"traffic"`
	Policies PolicyStats  `json:"policies"`
}

// FlowStats represents flow count statistics
type FlowStats struct {
	Total   int `json:"total"`
	Allowed int `json:"allowed"`
	Blocked int `json:"blocked"`
}

// TrafficStats represents traffic statistics
type TrafficStats struct {
	TotalPackets int64 `json:"totalPackets"`
	TotalBytes   int64 `json:"totalBytes"`
}

// PolicyStats represents policy statistics
type PolicyStats struct {
	TotalPolicyApplications int      `json:"totalPolicyApplications"`
	UniquePolicies          int      `json:"uniquePolicies"`
	UniquePolicyNames       []string `json:"uniquePolicyNames"`
	Tiers                   []string `json:"tiers"`
	Kinds                   []string `json:"kinds"`
}

// SecurityAlerts represents security alert information
type SecurityAlerts struct {
	Message      string   `json:"message"`
	BlockedFlows []string `json:"blockedFlows"`
}

// ServiceStatus represents Whisker service availability
type ServiceStatus struct {
	Available bool   `json:"available"`
	Details   string `json:"details"`
}

// BlockedFlowAnalysis represents analysis of blocked flows
type BlockedFlowAnalysis struct {
	Namespace        string                  `json:"namespace"`
	Analysis         BlockedFlowAnalysisInfo `json:"analysis"`
	BlockedFlows     []BlockedFlowDetail     `json:"blockedFlows"`
	SecurityInsights SecurityInsights        `json:"securityInsights"`
}

// BlockedFlowAnalysisInfo represents metadata about blocked flow analysis
type BlockedFlowAnalysisInfo struct {
	TotalBlockedFlows        int            `json:"totalBlockedFlows"`
	UniqueBlockedConnections int            `json:"uniqueBlockedConnections"`
	TimeWindow               TimeWindowInfo `json:"timeWindow"`
}

// BlockedFlowDetail represents detailed analysis of a blocked flow
type BlockedFlowDetail struct {
	Flow             BlockedFlowInfo  `json:"flow"`
	Traffic          TrafficInfo      `json:"traffic"`
	BlockingPolicies []BlockingPolicy `json:"blockingPolicies"`
	Analysis         FlowAnalysis     `json:"analysis"`
}

// BlockedFlowInfo represents information about a blocked flow
type BlockedFlowInfo struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
	Action      string `json:"action"`
	Reporter    string `json:"reporter"`
	TimeRange   string `json:"timeRange"`
}

// BlockingPolicy represents a policy that blocked traffic
type BlockingPolicy struct {
	TriggerPolicy  *Policy `json:"triggerPolicy"`
	PolicyYAML     *string `json:"policyYaml"`
	Error          *string `json:"error,omitempty"`
	BlockingReason string  `json:"blockingReason"`
}

// FlowAnalysis represents analysis results for a flow
type FlowAnalysis struct {
	TotalBlockingPolicies int    `json:"totalBlockingPolicies"`
	Recommendation        string `json:"recommendation"`
}

// SecurityInsights represents security insights from blocked flow analysis
type SecurityInsights struct {
	Message         string   `json:"message"`
	Recommendations []string `json:"recommendations"`
}

// FlowAggregateReport represents a comprehensive aggregated flow analysis report
type FlowAggregateReport struct {
	TimeRange         string                  `json:"timeRange"`
	TrafficOverview   []AggregatedFlowEntry   `json:"trafficOverview"`
	TrafficByCategory []TrafficCategory       `json:"trafficByCategory"`
	TopTrafficSources []TopTrafficEntity      `json:"topTrafficSources"`
	TopTrafficDest    []TopTrafficEntity      `json:"topTrafficDestinations"`
	NamespaceActivity []NamespaceActivityInfo `json:"namespaceActivity"`
	SecurityPosture   SecurityPostureInfo     `json:"securityPosture"`
}

// AggregatedFlowEntry represents an aggregated flow entry in the traffic overview
type AggregatedFlowEntry struct {
	Source          string `json:"source"`
	SourceNamespace string `json:"sourceNamespace"`
	Destination     string `json:"destination"`
	DestNamespace   string `json:"destNamespace"`
	Protocol        string `json:"protocol"`
	Port            int    `json:"port"`
	Action          string `json:"action"`
	PacketsIn       int64  `json:"packetsIn"`
	PacketsOut      int64  `json:"packetsOut"`
	BytesIn         int64  `json:"bytesIn"`
	BytesOut        int64  `json:"bytesOut"`
	PacketsInStr    string `json:"packetsInStr"`
	PacketsOutStr   string `json:"packetsOutStr"`
	BytesInStr      string `json:"bytesInStr"`
	BytesOutStr     string `json:"bytesOutStr"`
	PrimaryPolicy   string `json:"primaryPolicy"`
}

// TrafficCategory represents a categorized traffic type
type TrafficCategory struct {
	Category    string `json:"category"`
	Count       int    `json:"count"`
	Description string `json:"description"`
}

// TopTrafficEntity represents a top traffic source or destination
type TopTrafficEntity struct {
	Name            string `json:"name"`
	TotalFlows      int    `json:"totalFlows"`
	PrimaryActivity string `json:"primaryActivity"`
}

// NamespaceActivityInfo represents traffic activity for a namespace
type NamespaceActivityInfo struct {
	Namespace          string `json:"namespace"`
	IngressFlows       int    `json:"ingressFlows"`
	EgressFlows        int    `json:"egressFlows"`
	TotalTrafficVolume string `json:"totalTrafficVolume"`
	BytesIn            int64  `json:"bytesIn"`
	BytesOut           int64  `json:"bytesOut"`
}

// SecurityPostureInfo represents overall security posture
type SecurityPostureInfo struct {
	TotalFlows        int      `json:"totalFlows"`
	AllowedFlows      int      `json:"allowedFlows"`
	AllowedPercentage float64  `json:"allowedPercentage"`
	DeniedFlows       int      `json:"deniedFlows"`
	DeniedPercentage  float64  `json:"deniedPercentage"`
	ActivePolicies    int      `json:"activePolicies"`
	UniquePolicyNames []string `json:"uniquePolicyNames"`
}
