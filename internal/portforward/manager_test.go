package portforward

import (
	"context"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	kubeconfig := "/path/to/kubeconfig"
	manager := NewManager(kubeconfig)
	
	if manager == nil {
		t.Fatal("Expected manager to be created, got nil")
	}
	
	if manager.kubeconfigPath != kubeconfig {
		t.Errorf("Expected kubeconfigPath to be %s, got %s", kubeconfig, manager.kubeconfigPath)
	}
}

func TestIsRunning(t *testing.T) {
	manager := NewManager("")
	
	// Initially should not be running
	if manager.IsRunning() {
		t.Error("Expected manager to not be running initially")
	}
}

func TestStop(t *testing.T) {
	manager := NewManager("")
	
	// Should be able to stop even when not running
	if err := manager.Stop(); err != nil {
		t.Errorf("Expected no error when stopping inactive manager, got %v", err)
	}
}

// Integration test that requires kubectl to be available
func TestCheckWhiskerServiceStatusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	manager := NewManager("")
	available, details, err := manager.CheckWhiskerServiceStatus()
	
	// This test will fail if kubectl is not available or service doesn't exist
	// That's expected behavior for this integration test
	if err != nil {
		t.Logf("Service check error (expected in test environment): %v", err)
	}
	
	t.Logf("Service available: %v, Details: %s", available, details)
}

// Benchmark test
func BenchmarkNewManager(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewManager("/test/path")
	}
}

// Test context cancellation behavior
func TestSetupWithCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}
	
	manager := NewManager("")
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// This should fail quickly due to context timeout
	err := manager.Setup(ctx)
	if err == nil {
		t.Error("Expected setup to fail with context timeout")
		manager.Stop() // Clean up if somehow it succeeded
	}
}