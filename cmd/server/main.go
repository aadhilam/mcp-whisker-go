package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/aadhilam/mcp-whisker-go/internal/kubernetes"
	"github.com/aadhilam/mcp-whisker-go/internal/mcp"
	"github.com/aadhilam/mcp-whisker-go/internal/portforward"
	"github.com/aadhilam/mcp-whisker-go/internal/whisker"
	"github.com/spf13/cobra"
)

var (
	kubeconfigPath string
	namespace      string
	debug          bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mcp-whisker-go",
		Short: "Calico Whisker MCP Server for flow log analysis",
		Long: `A Go implementation of the Calico Whisker MCP Server that provides 
Model Context Protocol functionality for analyzing Calico Whisker flow logs 
in Kubernetes environments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to running as MCP server when no subcommand provided
			kubeconfig := getKubeconfigPath()
			server := mcp.NewMCPServer(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan
				cancel()
			}()

			// Log to stderr only, never to stdout (MCP uses stdout for JSON-RPC)
			log.SetOutput(os.Stderr)
			if debug {
				log.Printf("MCP server starting with kubeconfig: %s\n", kubeconfig)
			}

			return server.Run(ctx)
		},
		SilenceUsage: true, // Don't show usage on error
	}

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "",
		"Path to kubeconfig file (default: $HOME/.kube/config)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Add commands
	rootCmd.AddCommand(setupPortForwardCmd())
	rootCmd.AddCommand(getFlowsCmd())
	rootCmd.AddCommand(getAggregatedFlowsCmd())
	rootCmd.AddCommand(analyzeNamespaceCmd())
	rootCmd.AddCommand(analyzeBlockedCmd())
	rootCmd.AddCommand(checkServiceCmd())
	rootCmd.AddCommand(serverCmd())

	// Add Kubernetes commands
	rootCmd.AddCommand(k8sConnectCmd())
	rootCmd.AddCommand(k8sGetContextsCmd())
	rootCmd.AddCommand(k8sCurrentContextCmd())
	rootCmd.AddCommand(k8sCheckClusterCmd())
	rootCmd.AddCommand(k8sCheckWhiskerCmd())
	rootCmd.AddCommand(k8sCheckKubeconfigCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupPortForwardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup-port-forward",
		Short: "Setup port-forward to Whisker service",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Setup signal handling
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				<-sigChan
				fmt.Println("\nReceived interrupt signal, stopping port-forward...")
				cancel()
			}()

			kubeconfig := getKubeconfigPath()
			manager := portforward.NewManager(kubeconfig)

			fmt.Println("Setting up port-forward to Whisker service...")
			if err := manager.Setup(ctx); err != nil {
				return fmt.Errorf("failed to setup port-forward: %w", err)
			}

			fmt.Println("Port-forward established. Press Ctrl+C to stop.")

			// Wait for context cancellation
			<-ctx.Done()

			fmt.Println("Stopping port-forward...")
			return manager.Stop()
		},
	}
	return cmd
}

func getFlowsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-flows",
		Short: "Retrieve flow logs from Whisker service",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := whisker.NewService(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			flows, err := service.GetFlowLogs(ctx)
			if err != nil {
				return fmt.Errorf("failed to get flow logs: %w", err)
			}

			output, err := json.MarshalIndent(flows, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal flows: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}
	return cmd
}

func getAggregatedFlowsCmd() *cobra.Command {
	var startTime, endTime string
	var markdown bool

	cmd := &cobra.Command{
		Use:   "get-aggregated-flows",
		Short: "Get aggregated and categorized flow logs with traffic analysis",
		Long: `Retrieve flow logs from Whisker service and present them in an aggregated format
with traffic categorization, top sources/destinations, namespace activity, and security posture.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := whisker.NewService(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var startTimePtr, endTimePtr *string
			if startTime != "" {
				startTimePtr = &startTime
			}
			if endTime != "" {
				endTimePtr = &endTime
			}

			report, err := service.GetAggregatedFlowReport(ctx, startTimePtr, endTimePtr)
			if err != nil {
				return fmt.Errorf("failed to get aggregated flow logs: %w", err)
			}

			if markdown {
				// Output as formatted Markdown
				output := service.FormatAggregateReportAsMarkdown(report)
				fmt.Println(output)
			} else {
				// Output as JSON
				output, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal report: %w", err)
				}
				fmt.Println(string(output))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&startTime, "start-time", "", "Start time filter (ISO8601 format)")
	cmd.Flags().StringVar(&endTime, "end-time", "", "End time filter (ISO8601 format)")
	cmd.Flags().BoolVarP(&markdown, "markdown", "m", true, "Output in Markdown format (default: true)")
	return cmd
}

func analyzeNamespaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze-namespace",
		Short: "Analyze flows for a specific namespace",
		RunE: func(cmd *cobra.Command, args []string) error {
			if namespace == "" {
				return fmt.Errorf("namespace is required")
			}

			kubeconfig := getKubeconfigPath()
			service := whisker.NewService(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			summary, err := service.GetNamespaceFlowSummary(ctx, namespace)
			if err != nil {
				return fmt.Errorf("failed to analyze namespace flows: %w", err)
			}

			output, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal summary: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to analyze (required)")
	cmd.MarkFlagRequired("namespace")
	return cmd
}

func analyzeBlockedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze-blocked",
		Short: "Analyze blocked flows",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := whisker.NewService(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			analysis, err := service.AnalyzeBlockedFlows(ctx, namespace)
			if err != nil {
				return fmt.Errorf("failed to analyze blocked flows: %w", err)
			}

			output, err := json.MarshalIndent(analysis, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal analysis: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to analyze (optional, analyzes all if not specified)")
	return cmd
}

func checkServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-service",
		Short: "Check Whisker service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			manager := portforward.NewManager(kubeconfig)

			available, details, err := manager.CheckWhiskerServiceStatus()
			if err != nil {
				return fmt.Errorf("failed to check service status: %w", err)
			}

			status := map[string]interface{}{
				"available": available,
				"details":   details,
			}

			output, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal status: %w", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}
	return cmd
}

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run as MCP server (explicit command, same as default behavior)",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			server := mcp.NewMCPServer(kubeconfig)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle graceful shutdown
			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
				<-sigChan
				cancel()
			}()

			// Log to stderr only
			log.SetOutput(os.Stderr)
			if debug {
				log.Printf("MCP server starting with kubeconfig: %s\n", kubeconfig)
			}

			return server.Run(ctx)
		},
	}
	return cmd
}

func getKubeconfigPath() string {
	if kubeconfigPath != "" {
		// Expand tilde (~) to home directory
		if strings.HasPrefix(kubeconfigPath, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				return filepath.Join(home, kubeconfigPath[2:])
			}
		}
		return kubeconfigPath
	}

	// Try default location
	if home, err := os.UserHomeDir(); err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath
		}
	}

	return ""
}

// Kubernetes CLI commands

func k8sConnectCmd() *cobra.Command {
	var contextName string

	cmd := &cobra.Command{
		Use:   "k8s-connect",
		Short: "Connect to a Kubernetes cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			if err := service.Connect(ctx, contextName, kubeconfig); err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}

			fmt.Printf("✅ Successfully connected to Kubernetes cluster")
			if contextName != "" {
				fmt.Printf(" using context '%s'", contextName)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&contextName, "context", "c", "", "Kubernetes context name")
	return cmd
}

func k8sGetContextsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-contexts",
		Short: "List all available Kubernetes contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			contexts, err := service.GetAvailableContexts(kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to get contexts: %w", err)
			}

			result, _ := json.MarshalIndent(map[string]interface{}{
				"contexts": contexts,
				"total":    len(contexts),
			}, "", "  ")

			fmt.Println(string(result))
			return nil
		},
	}
	return cmd
}

func k8sCurrentContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-current-context",
		Short: "Show current Kubernetes context",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			currentContext, err := service.GetCurrentContextInfo(kubeconfig)
			if err != nil {
				return fmt.Errorf("failed to get current context: %w", err)
			}

			if currentContext == nil {
				fmt.Println("No current context set")
				return nil
			}

			result, _ := json.MarshalIndent(currentContext, "", "  ")
			fmt.Println(string(result))
			return nil
		},
	}
	return cmd
}

func k8sCheckClusterCmd() *cobra.Command {
	var contextName string

	cmd := &cobra.Command{
		Use:   "k8s-check-cluster",
		Short: "Check Kubernetes cluster accessibility",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			var contextInfo *kubernetes.ContextInfo
			if contextName != "" {
				contextInfo = &kubernetes.ContextInfo{Name: contextName}
			}

			status := service.CheckServerAccessibility(ctx, contextInfo)

			result := map[string]interface{}{
				"accessible": status.Accessible,
				"status":     "✅ Accessible",
			}

			if !status.Accessible {
				result["status"] = "❌ Not Accessible"
				result["error"] = status.Error
			}

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonResult))
			return nil
		},
	}

	cmd.Flags().StringVarP(&contextName, "context", "c", "", "Kubernetes context to check")
	return cmd
}

func k8sCheckWhiskerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-check-whisker",
		Short: "Check if Calico Whisker is installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			installed := service.CheckCalicoWhiskerInstalled(ctx)
			whiskerStatus := service.CheckWhiskerService(ctx)

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

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonResult))
			return nil
		},
	}
	return cmd
}

func k8sCheckKubeconfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s-check-kubeconfig",
		Short: "Check kubeconfig file status",
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfig := getKubeconfigPath()
			service := kubernetes.NewService(kubeconfig)

			defaultPath := service.GetDefaultKubeconfigPath()
			exists := service.KubeconfigExists(kubeconfig)

			checkPath := kubeconfig
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

			jsonResult, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonResult))
			return nil
		},
	}
	return cmd
}

func init() {
	if debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}
