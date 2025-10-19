package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aadhilam/mcp-whisker-go/internal/kubernetes"
	"github.com/aadhilam/mcp-whisker-go/internal/portforward"
	"github.com/aadhilam/mcp-whisker-go/internal/whisker"
)

// MCPServer represents the Model Context Protocol server
type MCPServer struct {
	input      io.Reader
	output     io.Writer
	manager    *portforward.Manager
	service    *whisker.Service
	k8sService *kubernetes.Service
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in MCP format
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// NewMCPServer creates a new MCP server
func NewMCPServer(kubeconfigPath string) *MCPServer {
	return &MCPServer{
		input:      os.Stdin,
		output:     os.Stdout,
		manager:    portforward.NewManager(kubeconfigPath),
		service:    whisker.NewService(kubeconfigPath),
		k8sService: kubernetes.NewService(kubeconfigPath),
	}
}

// Run starts the MCP server
func (s *MCPServer) Run(ctx context.Context) error {
	log.Println("Starting MCP server...")

	scanner := bufio.NewScanner(s.input)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var request MCPRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			// Try to extract ID from malformed request for proper error response
			var partialReq struct {
				ID interface{} `json:"id"`
			}
			json.Unmarshal([]byte(line), &partialReq)

			// Use extracted ID or generate one if none found
			requestID := partialReq.ID
			if requestID == nil {
				requestID = "unknown"
			}

			s.sendErrorResponse(requestID, -32700, "Parse error")
			continue
		}

		// Ensure request ID is not nil
		if request.ID == nil {
			s.sendErrorResponse("unknown", -32600, "Invalid Request: missing id")
			continue
		}

		response := s.handleRequest(ctx, &request)
		if response != nil {
			s.sendResponse(response)
		}
	}

	return scanner.Err()
}

// handleRequest processes an MCP request
func (s *MCPServer) handleRequest(ctx context.Context, req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32601, Message: "Method not found"},
		}
	}
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(req *MCPRequest) *MCPResponse {
	// Extract client protocol version if available
	clientProtocolVersion := "2024-11-05" // Default
	if params, ok := req.Params.(map[string]interface{}); ok {
		if pv, exists := params["protocolVersion"]; exists {
			if pvStr, ok := pv.(string); ok {
				clientProtocolVersion = pvStr
			}
		}
	}

	result := map[string]interface{}{
		"protocolVersion": clientProtocolVersion, // Echo back the client's version
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mcp-whisker-go",
			"version": "1.0.0",
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleToolsList returns available tools
func (s *MCPServer) handleToolsList(req *MCPRequest) *MCPResponse {
	tools := []Tool{
		{
			Name:        "setup_port_forward",
			Description: "Setup port-forward to Calico Whisker service",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes namespace (default: calico-system)",
						"default":     "calico-system",
					},
				},
			},
		},
		{
			Name:        "get_flow_logs",
			Description: "Retrieve flow logs from Calico Whisker",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"setup_port_forward": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to setup port-forward first (default: true)",
						"default":     true,
					},
				},
			},
		},
		{
			Name:        "analyze_namespace_flows",
			Description: "Analyze flow logs for a specific namespace",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Target namespace to analyze",
					},
					"setup_port_forward": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to setup port-forward first (default: true)",
						"default":     true,
					},
				},
				"required": []string{"namespace"},
			},
		},
		{
			Name:        "analyze_blocked_flows",
			Description: "Analyze blocked flows and identify blocking policies",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"namespace": map[string]interface{}{
						"type":        "string",
						"description": "Optional namespace filter",
					},
					"setup_port_forward": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to setup port-forward first (default: true)",
						"default":     true,
					},
				},
			},
		},
		{
			Name:        "check_whisker_service",
			Description: "Check if Calico Whisker service is available",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "k8s_connect",
			Description: "Connect to a Kubernetes cluster and set context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes context name to use",
					},
					"kubeconfig_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to kubeconfig file (optional)",
					},
				},
			},
		},
		{
			Name:        "k8s_get_contexts",
			Description: "Get all available Kubernetes contexts",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kubeconfig_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to kubeconfig file (optional)",
					},
				},
			},
		},
		{
			Name:        "k8s_get_current_context",
			Description: "Get information about the current Kubernetes context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kubeconfig_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to kubeconfig file (optional)",
					},
				},
			},
		},
		{
			Name:        "k8s_check_cluster_access",
			Description: "Check if Kubernetes cluster is accessible",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Kubernetes context name to check (optional)",
					},
				},
			},
		},
		{
			Name:        "k8s_check_whisker_installation",
			Description: "Check if Calico Whisker is installed in the cluster",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "k8s_check_kubeconfig",
			Description: "Check if kubeconfig file exists and get default path",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kubeconfig_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to kubeconfig file to check (optional)",
					},
				},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleToolsCall executes a tool
func (s *MCPServer) handleToolsCall(ctx context.Context, req *MCPRequest) *MCPResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32602, Message: "Invalid params"},
		}
	}

	name, ok := params["name"].(string)
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32602, Message: "Missing tool name"},
		}
	}

	arguments := make(map[string]interface{})
	if args, ok := params["arguments"].(map[string]interface{}); ok {
		arguments = args
	}

	result, err := s.executeTool(ctx, name, arguments)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &MCPError{Code: -32000, Message: err.Error()},
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// executeTool executes the specified tool
func (s *MCPServer) executeTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	switch name {
	case "setup_port_forward":
		return s.setupPortForward(ctx, args)
	case "get_flow_logs":
		return s.getFlowLogs(ctx, args)
	case "analyze_namespace_flows":
		return s.analyzeNamespaceFlows(ctx, args)
	case "analyze_blocked_flows":
		return s.analyzeBlockedFlows(ctx, args)
	case "check_whisker_service":
		return s.checkWhiskerService(ctx, args)
	case "k8s_connect":
		return s.k8sConnect(ctx, args)
	case "k8s_get_contexts":
		return s.k8sGetContexts(ctx, args)
	case "k8s_get_current_context":
		return s.k8sGetCurrentContext(ctx, args)
	case "k8s_check_cluster_access":
		return s.k8sCheckClusterAccess(ctx, args)
	case "k8s_check_whisker_installation":
		return s.k8sCheckWhiskerInstallation(ctx, args)
	case "k8s_check_kubeconfig":
		return s.k8sCheckKubeconfig(ctx, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// Tool implementations
func (s *MCPServer) setupPortForward(ctx context.Context, args map[string]interface{}) (string, error) {
	namespace := "calico-system"
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	err := s.manager.Setup(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to setup port-forward: %w", err)
	}

	return fmt.Sprintf("✅ Port-forward to Calico Whisker service in namespace '%s' established successfully", namespace), nil
}

func (s *MCPServer) getFlowLogs(ctx context.Context, args map[string]interface{}) (string, error) {
	setupPortForward := true
	if setup, ok := args["setup_port_forward"].(bool); ok {
		setupPortForward = setup
	}

	if setupPortForward {
		if err := s.manager.Setup(ctx); err != nil {
			return "", fmt.Errorf("failed to setup port-forward: %w", err)
		}
	}

	flows, err := s.service.GetFlowLogs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get flow logs: %w", err)
	}

	result, err := json.MarshalIndent(flows, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal flow logs: %w", err)
	}

	return string(result), nil
}

func (s *MCPServer) analyzeNamespaceFlows(ctx context.Context, args map[string]interface{}) (string, error) {
	namespace, ok := args["namespace"].(string)
	if !ok || namespace == "" {
		return "", fmt.Errorf("namespace is required")
	}

	setupPortForward := true
	if setup, ok := args["setup_port_forward"].(bool); ok {
		setupPortForward = setup
	}

	if setupPortForward {
		if err := s.manager.Setup(ctx); err != nil {
			return "", fmt.Errorf("failed to setup port-forward: %w", err)
		}
	}

	summary, err := s.service.GetNamespaceFlowSummary(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to analyze namespace flows: %w", err)
	}

	result, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %w", err)
	}

	return string(result), nil
}

func (s *MCPServer) analyzeBlockedFlows(ctx context.Context, args map[string]interface{}) (string, error) {
	var namespace string
	if ns, ok := args["namespace"].(string); ok {
		namespace = ns
	}

	setupPortForward := true
	if setup, ok := args["setup_port_forward"].(bool); ok {
		setupPortForward = setup
	}

	if setupPortForward {
		if err := s.manager.Setup(ctx); err != nil {
			return "", fmt.Errorf("failed to setup port-forward: %w", err)
		}
	}

	analysis, err := s.service.AnalyzeBlockedFlows(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to analyze blocked flows: %w", err)
	}

	result, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal analysis: %w", err)
	}

	return string(result), nil
}

func (s *MCPServer) checkWhiskerService(ctx context.Context, args map[string]interface{}) (string, error) {
	available, details, err := s.manager.CheckWhiskerServiceStatus()
	if err != nil {
		return "", fmt.Errorf("failed to check service status: %w", err)
	}

	statusText := "❌ Not Available"
	if available {
		statusText = "✅ Available"
	}

	status := map[string]interface{}{
		"available": available,
		"details":   details,
		"status":    statusText,
	}

	result, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal status: %w", err)
	}

	return string(result), nil
}

// sendResponse sends a response to the client
func (s *MCPServer) sendResponse(response *MCPResponse) {
	// Validate response before sending
	if response == nil {
		log.Printf("Warning: Attempted to send nil response")
		return
	}

	// Ensure ID is not nil for proper JSON-RPC compliance
	if response.ID == nil {
		log.Printf("Warning: Response ID is nil, setting to 'unknown'")
		response.ID = "unknown"
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	fmt.Fprintln(s.output, string(data))
}

// sendErrorResponse sends an error response
func (s *MCPServer) sendErrorResponse(id interface{}, code int, message string) {
	response := &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &MCPError{Code: code, Message: message},
	}
	s.sendResponse(response)
}

// Kubernetes tool implementations

func (s *MCPServer) k8sConnect(ctx context.Context, args map[string]interface{}) (string, error) {
	var contextName, kubeconfigPath string

	if context, ok := args["context"].(string); ok {
		contextName = context
	}

	if kubeconfig, ok := args["kubeconfig_path"].(string); ok {
		kubeconfigPath = kubeconfig
	}

	err := s.k8sService.Connect(ctx, contextName, kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Kubernetes cluster: %w", err)
	}

	message := "✅ Successfully connected to Kubernetes cluster"
	if contextName != "" {
		message += fmt.Sprintf(" using context '%s'", contextName)
	}
	if kubeconfigPath != "" {
		message += fmt.Sprintf(" with kubeconfig: %s", kubeconfigPath)
	}

	return message, nil
}

func (s *MCPServer) k8sGetContexts(ctx context.Context, args map[string]interface{}) (string, error) {
	var kubeconfigPath string
	if kubeconfig, ok := args["kubeconfig_path"].(string); ok {
		kubeconfigPath = kubeconfig
	}

	contexts, err := s.k8sService.GetAvailableContexts(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to get available contexts: %w", err)
	}

	result, err := json.MarshalIndent(map[string]interface{}{
		"contexts": contexts,
		"total":    len(contexts),
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal contexts: %w", err)
	}

	return string(result), nil
}

func (s *MCPServer) k8sGetCurrentContext(ctx context.Context, args map[string]interface{}) (string, error) {
	var kubeconfigPath string
	if kubeconfig, ok := args["kubeconfig_path"].(string); ok {
		kubeconfigPath = kubeconfig
	}

	currentContext, err := s.k8sService.GetCurrentContextInfo(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}

	if currentContext == nil {
		return "No current context set", nil
	}

	result, err := json.MarshalIndent(currentContext, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal current context: %w", err)
	}

	return string(result), nil
}

func (s *MCPServer) k8sCheckClusterAccess(ctx context.Context, args map[string]interface{}) (string, error) {
	var contextInfo *kubernetes.ContextInfo

	if contextName, ok := args["context"].(string); ok && contextName != "" {
		contextInfo = &kubernetes.ContextInfo{Name: contextName}
	}

	status := s.k8sService.CheckServerAccessibility(ctx, contextInfo)

	result := map[string]interface{}{
		"accessible": status.Accessible,
		"status":     "✅ Accessible",
	}

	if !status.Accessible {
		result["status"] = "❌ Not Accessible"
		result["error"] = status.Error
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster status: %w", err)
	}

	return string(jsonResult), nil
}

func (s *MCPServer) k8sCheckWhiskerInstallation(ctx context.Context, args map[string]interface{}) (string, error) {
	// Check if calico-system namespace exists
	installed := s.k8sService.CheckCalicoWhiskerInstalled(ctx)

	// Also check the whisker service specifically
	whiskerStatus := s.k8sService.CheckWhiskerService(ctx)

	result := map[string]interface{}{
		"calico_system_namespace": installed,
		"whisker_service":         whiskerStatus,
		"overall_status":          "❌ Not Installed",
	}

	if installed && whiskerStatus.Available {
		result["overall_status"] = "✅ Fully Installed"
	} else if installed {
		result["overall_status"] = "⚠️ Partially Installed (namespace exists but service not found)"
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal installation status: %w", err)
	}

	return string(jsonResult), nil
}

func (s *MCPServer) k8sCheckKubeconfig(ctx context.Context, args map[string]interface{}) (string, error) {
	var kubeconfigPath string
	if kubeconfig, ok := args["kubeconfig_path"].(string); ok {
		kubeconfigPath = kubeconfig
	}

	defaultPath := s.k8sService.GetDefaultKubeconfigPath()
	exists := s.k8sService.KubeconfigExists(kubeconfigPath)

	checkPath := kubeconfigPath
	if checkPath == "" {
		checkPath = defaultPath
	}

	result := map[string]interface{}{
		"default_path": defaultPath,
		"checked_path": checkPath,
		"exists":       exists,
		"status":       "❌ Not Found",
	}

	if exists {
		result["status"] = "✅ Found"
	}

	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal kubeconfig status: %w", err)
	}

	return string(jsonResult), nil
}
