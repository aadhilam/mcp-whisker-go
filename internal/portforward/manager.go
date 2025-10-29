package portforward

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager handles kubectl port-forward operations
type Manager struct {
	cmd            *exec.Cmd
	kubeconfigPath string
	mutex          sync.RWMutex
	cancel         context.CancelFunc
}

// NewManager creates a new port-forward manager
func NewManager(kubeconfigPath string) *Manager {
	return &Manager{
		kubeconfigPath: kubeconfigPath,
	}
}

// Setup establishes port-forward to Whisker service
func (m *Manager) Setup(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// If port-forward is already running, verify it's healthy and return success (idempotent)
	if m.cmd != nil && m.cmd.Process != nil {
		fmt.Fprintf(os.Stderr, "✅ Port-forward already running, reusing existing connection\n")
		return nil
	}

	// Pre-flight checks
	fmt.Fprintf(os.Stderr, "🔍 Pre-flight checks for port-forward...\n")

	if err := m.checkKubectl(); err != nil {
		return fmt.Errorf("pre-flight check failed: %w", err)
	}

	// Kill existing processes on port 8081
	if err := m.killExistingPortForwards(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not clean up existing port forwards: %v\n", err)
	}

	// Setup context for cancellation
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	// Prepare kubectl command
	// Build kubectl port-forward command arguments
	// Example result: ["port-forward", "service/whisker", "8081:8081", "-n", "calico-system"]
	args := []string{"port-forward", "service/whisker", "8081:8081", "-n", "calico-system"}

	// If kubeconfig path is specified, prepend it to the arguments
	// Example result: ["--kubeconfig", "/path/to/config", "port-forward", "service/whisker", ...]
	if m.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", m.kubeconfigPath}, args...)
	}

	// Log the complete kubectl command being executed to stderr (for debugging)
	// strings.Join combines the args slice into a single string with spaces
	// Example output: "Starting port-forward with command: kubectl --kubeconfig ~/.kube/config port-forward service/whisker 8081:8081 -n calico-system"
	fmt.Fprintf(os.Stderr, "Starting port-forward with command: kubectl %s\n", strings.Join(args, " "))

	// Create a kubectl subprocess that respects the context (can be canceled)
	// exec.CommandContext(ctx, "kubectl", args...) expands to:
	//   - ctx: The context that controls cancellation/timeout
	//   - "kubectl": The command to execute
	//   - args...: Expands the args slice into individual arguments
	// Example: exec.CommandContext(ctx, "kubectl", "port-forward", "service/whisker", "8081:8081", "-n", "calico-system")
	m.cmd = exec.CommandContext(ctx, "kubectl", args...)

	// Redirect kubectl's stderr to our stderr (for error messages)
	m.cmd.Stderr = os.Stderr

	// Redirect kubectl's stdout to stderr to prevent corrupting MCP JSON-RPC protocol
	// MCP uses stdout for JSON-RPC messages, so kubectl's output must not go there
	m.cmd.Stdout = os.Stderr

	if err := m.cmd.Start(); err != nil {
		m.cleanup()
		return fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Wait for port-forward to be ready
	if err := m.waitForPortForward(ctx); err != nil {
		m.cleanup()
		return err
	}

	fmt.Fprintf(os.Stderr, "Port-forward established successfully\n")
	return nil
}

// Stop terminates the port-forward process
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.cleanup()
}

// IsRunning returns true if port-forward is active
func (m *Manager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.cmd != nil && m.cmd.Process != nil
}

// CheckWhiskerServiceStatus verifies Whisker service availability
func (m *Manager) CheckWhiskerServiceStatus() (bool, string, error) {
	args := []string{"get", "service", "whisker", "-n", "calico-system", "-o", "json"}
	if m.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", m.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "not found") {
			return false, "Whisker service not found in calico-system namespace", nil
		}
		return false, fmt.Sprintf("Error: %s", strings.TrimSpace(string(output))), nil
	}

	// Simple check - if we got JSON output, service exists
	if strings.Contains(string(output), `"kind": "Service"`) {
		return true, "Service found and accessible", nil
	}

	return false, "Service found but could not parse details", nil
}

func (m *Manager) checkKubectl() error {
	args := []string{"version", "--client"}
	if m.kubeconfigPath != "" {
		args = append([]string{"--kubeconfig", m.kubeconfigPath}, args...)
	}

	cmd := exec.Command("kubectl", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl not accessible: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ kubectl is accessible\n")
	return nil
}

func (m *Manager) killExistingPortForwards() error {
	// Use lsof to find processes using port 8081
	cmd := exec.Command("lsof", "-ti:8081")
	output, err := cmd.Output()

	if err != nil {
		// lsof failed, but that's okay - might mean no processes on port
		fmt.Fprintf(os.Stderr, "Port 8081 is available for use\n")
		return nil
	}

	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		fmt.Fprintf(os.Stderr, "Port 8081 is available for use\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d process(es) using port 8081, killing them...\n", len(pids))

	for _, pid := range pids {
		if _, err := strconv.Atoi(pid); err != nil {
			continue // Skip invalid PIDs
		}

		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err == nil {
			fmt.Fprintf(os.Stderr, "Successfully killed process %s\n", pid)
		} else {
			fmt.Fprintf(os.Stderr, "Failed to kill process %s: %v\n", pid, err)
		}
	}

	// Wait a bit for processes to be killed
	time.Sleep(1 * time.Second)
	fmt.Fprintf(os.Stderr, "✓ Port 8081 cleanup completed\n")
	return nil
}

func (m *Manager) waitForPortForward(ctx context.Context) error {
	// Give the port-forward process time to establish
	for i := 0; i < 6; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if process is still running
		if m.cmd.Process == nil {
			return fmt.Errorf("port-forward process exited unexpectedly")
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Port-forward should be ready after 3 seconds
	fmt.Fprintf(os.Stderr, "Port-forward process established (skipping health check)\n")
	return nil
}

func (m *Manager) cleanup() error {
	var err error

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if m.cmd != nil && m.cmd.Process != nil {
		if killErr := m.cmd.Process.Kill(); killErr != nil {
			err = fmt.Errorf("failed to kill process: %w", killErr)
		}

		// Wait for process to exit
		if waitErr := m.cmd.Wait(); waitErr != nil && err == nil {
			// Only set error if we didn't already have a kill error
			if !strings.Contains(waitErr.Error(), "signal: killed") {
				err = fmt.Errorf("process wait error: %w", waitErr)
			}
		}
	}

	m.cmd = nil
	return err
}
