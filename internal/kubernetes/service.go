package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Service provides Kubernetes cluster management functionality
type Service struct {
	kubeconfigPath string
	currentContext string
}

// ContextInfo represents information about a Kubernetes context
type ContextInfo struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`
	User      string `json:"user"`
	Namespace string `json:"namespace,omitempty"`
	IsCurrent bool   `json:"isCurrent"`
}

// ClusterStatus represents the status of cluster connectivity
type ClusterStatus struct {
	Accessible bool   `json:"accessible"`
	Error      string `json:"error,omitempty"`
}

// WhiskerStatus represents the status of Whisker service
type WhiskerStatus struct {
	Available bool   `json:"available"`
	Details   string `json:"details"`
}

// KubeConfig represents the structure of a kubeconfig file
type KubeConfig struct {
	APIVersion     string         `yaml:"apiVersion"`
	Kind           string         `yaml:"kind"`
	CurrentContext string         `yaml:"current-context"`
	Contexts       []ContextEntry `yaml:"contexts"`
	Clusters       []ClusterEntry `yaml:"clusters"`
	Users          []UserEntry    `yaml:"users"`
}

// ContextEntry represents a context entry in kubeconfig
type ContextEntry struct {
	Name    string        `yaml:"name"`
	Context ContextDetail `yaml:"context"`
}

// ContextDetail represents the details of a context
type ContextDetail struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

// ClusterEntry represents a cluster entry in kubeconfig
type ClusterEntry struct {
	Name    string        `yaml:"name"`
	Cluster ClusterDetail `yaml:"cluster"`
}

// ClusterDetail represents the details of a cluster
type ClusterDetail struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

// UserEntry represents a user entry in kubeconfig
type UserEntry struct {
	Name string     `yaml:"name"`
	User UserDetail `yaml:"user"`
}

// UserDetail represents the details of a user
type UserDetail struct {
	ClientCertificate     string                 `yaml:"client-certificate,omitempty"`
	ClientCertificateData string                 `yaml:"client-certificate-data,omitempty"`
	ClientKey             string                 `yaml:"client-key,omitempty"`
	ClientKeyData         string                 `yaml:"client-key-data,omitempty"`
	Token                 string                 `yaml:"token,omitempty"`
	Username              string                 `yaml:"username,omitempty"`
	Password              string                 `yaml:"password,omitempty"`
	Exec                  map[string]interface{} `yaml:"exec,omitempty"`
}

// NewService creates a new Kubernetes service
func NewService(kubeconfigPath string) *Service {
	if kubeconfigPath == "" {
		homeDir, _ := os.UserHomeDir()
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	return &Service{
		kubeconfigPath: kubeconfigPath,
	}
}

// Connect establishes connection to a Kubernetes cluster
func (s *Service) Connect(ctx context.Context, contextName string, kubeconfigPath string) error {
	// Set kubeconfig path if provided
	if kubeconfigPath != "" {
		s.kubeconfigPath = kubeconfigPath
	}

	// Set context if provided
	if contextName != "" {
		if err := s.SetContext(ctx, contextName); err != nil {
			return fmt.Errorf("failed to set context: %w", err)
		}
	}

	// Verify connection
	return s.VerifyConnection(ctx)
}

// SetContext sets the current Kubernetes context
func (s *Service) SetContext(ctx context.Context, contextName string) error {
	args := []string{"config", "use-context", contextName}
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set context to %s: %w", contextName, err)
	}

	s.currentContext = contextName
	return nil
}

// VerifyConnection verifies connectivity to the Kubernetes cluster
func (s *Service) VerifyConnection(ctx context.Context) error {
	args := []string{"cluster-info"}
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to Kubernetes cluster: %s", string(output))
	}

	return nil
}

// GetAvailableContexts returns all available Kubernetes contexts
func (s *Service) GetAvailableContexts(kubeconfigPath string) ([]ContextInfo, error) {
	configPath := kubeconfigPath
	if configPath == "" {
		configPath = s.kubeconfigPath
	}

	kubeconfig, err := s.parseKubeConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	contexts := make([]ContextInfo, 0, len(kubeconfig.Contexts))
	for _, ctx := range kubeconfig.Contexts {
		contexts = append(contexts, ContextInfo{
			Name:      ctx.Name,
			Cluster:   ctx.Context.Cluster,
			User:      ctx.Context.User,
			Namespace: ctx.Context.Namespace,
			IsCurrent: ctx.Name == kubeconfig.CurrentContext,
		})
	}

	return contexts, nil
}

// GetCurrentContextInfo returns information about the current context
func (s *Service) GetCurrentContextInfo(kubeconfigPath string) (*ContextInfo, error) {
	configPath := kubeconfigPath
	if configPath == "" {
		configPath = s.kubeconfigPath
	}

	kubeconfig, err := s.parseKubeConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	if kubeconfig.CurrentContext == "" {
		return nil, nil
	}

	for _, ctx := range kubeconfig.Contexts {
		if ctx.Name == kubeconfig.CurrentContext {
			return &ContextInfo{
				Name:      ctx.Name,
				Cluster:   ctx.Context.Cluster,
				User:      ctx.Context.User,
				Namespace: ctx.Context.Namespace,
				IsCurrent: true,
			}, nil
		}
	}

	return nil, nil
}

// GetDefaultKubeconfigPath returns the default kubeconfig path
func (s *Service) GetDefaultKubeconfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", "config")
}

// KubeconfigExists checks if the kubeconfig file exists
func (s *Service) KubeconfigExists(kubeconfigPath string) bool {
	configPath := kubeconfigPath
	if configPath == "" {
		configPath = s.GetDefaultKubeconfigPath()
	}

	_, err := os.Stat(configPath)
	return err == nil
}

// CheckServerAccessibility checks if the Kubernetes server is accessible
func (s *Service) CheckServerAccessibility(ctx context.Context, contextInfo *ContextInfo) ClusterStatus {
	args := []string{"cluster-info"}
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}
	if contextInfo != nil && contextInfo.Name != "" {
		args = append(args, "--context", contextInfo.Name)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ClusterStatus{
			Accessible: false,
			Error:      string(output),
		}
	}

	return ClusterStatus{
		Accessible: true,
	}
}

// CheckWhiskerService checks if Calico Whisker service is available
func (s *Service) CheckWhiskerService(ctx context.Context) WhiskerStatus {
	args := []string{"get", "service", "whisker", "-n", "calico-system", "-o", "json"}
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "not found") {
			return WhiskerStatus{
				Available: false,
				Details:   "Whisker service not found in calico-system namespace",
			}
		}
		return WhiskerStatus{
			Available: false,
			Details:   fmt.Sprintf("Error checking service: %s", string(output)),
		}
	}

	// Parse service details
	var service map[string]interface{}
	if err := json.Unmarshal(output, &service); err != nil {
		return WhiskerStatus{
			Available: true,
			Details:   "Service found but could not parse details",
		}
	}

	// Check for whisker port
	spec, ok := service["spec"].(map[string]interface{})
	if !ok {
		return WhiskerStatus{
			Available: true,
			Details:   "Service found but spec not accessible",
		}
	}

	ports, ok := spec["ports"].([]interface{})
	if !ok {
		return WhiskerStatus{
			Available: true,
			Details:   "Service found but ports not accessible",
		}
	}

	whiskerPortFound := false
	for _, port := range ports {
		portMap, ok := port.(map[string]interface{})
		if !ok {
			continue
		}

		if portVal, exists := portMap["port"]; exists {
			if portFloat, ok := portVal.(float64); ok && int(portFloat) == 8081 {
				whiskerPortFound = true
				break
			}
		}

		if targetPortVal, exists := portMap["targetPort"]; exists {
			if targetPortFloat, ok := targetPortVal.(float64); ok && int(targetPortFloat) == 8081 {
				whiskerPortFound = true
				break
			}
		}
	}

	portStatus := "Not found"
	if whiskerPortFound {
		portStatus = "Available"
	}

	return WhiskerStatus{
		Available: true,
		Details:   fmt.Sprintf("Service found with %d port(s). Whisker port (8081): %s", len(ports), portStatus),
	}
}

// CheckCalicoWhiskerInstalled checks if Calico Whisker is installed
func (s *Service) CheckCalicoWhiskerInstalled(ctx context.Context) bool {
	args := []string{"get", "namespace", "calico-system"}
	if s.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", s.kubeconfigPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)
	err := cmd.Run()
	return err == nil
}

// GetCurrentContext returns the current context name
func (s *Service) GetCurrentContext() string {
	return s.currentContext
}

// GetKubeconfigPath returns the current kubeconfig path
func (s *Service) GetKubeconfigPath() string {
	return s.kubeconfigPath
}

// parseKubeConfig parses a kubeconfig file
func (s *Service) parseKubeConfig(kubeconfigPath string) (*KubeConfig, error) {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file not found at: %s", kubeconfigPath)
	}

	data, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file: %w", err)
	}

	var kubeconfig KubeConfig
	if err := yaml.Unmarshal(data, &kubeconfig); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig YAML: %w", err)
	}

	if len(kubeconfig.Contexts) == 0 {
		return nil, fmt.Errorf("invalid kubeconfig format: no contexts found")
	}

	return &kubeconfig, nil
}
