# Kubernetes Integration for MCP Whisker Go

This document describes the Kubernetes integration added to the MCP Whisker Go server, providing similar functionality to the TypeScript version.

## Overview

The Kubernetes integration provides tools for:
- Managing Kubernetes contexts and connections
- Checking cluster accessibility 
- Verifying Calico Whisker installation
- Validating kubeconfig files

## New MCP Tools

### 1. `k8s_connect`
**Description**: Connect to a Kubernetes cluster and set context

**Parameters**:
- `context` (optional): Kubernetes context name to use
- `kubeconfig_path` (optional): Path to kubeconfig file

**Example**:
```json
{
  "name": "k8s_connect",
  "arguments": {
    "context": "aks-calico-demo",
    "kubeconfig_path": "/path/to/kubeconfig"
  }
}
```

### 2. `k8s_get_contexts`
**Description**: Get all available Kubernetes contexts

**Parameters**:
- `kubeconfig_path` (optional): Path to kubeconfig file

**Returns**: List of all contexts with cluster, user, and namespace information

**Example Response**:
```json
{
  "contexts": [
    {
      "name": "aks-calico-demo",
      "cluster": "aks-calico-demo", 
      "user": "clusterUser_rg-calico-demo_aks-calico-demo",
      "isCurrent": true
    }
  ],
  "total": 5
}
```

### 3. `k8s_get_current_context`
**Description**: Get information about the current Kubernetes context

**Parameters**:
- `kubeconfig_path` (optional): Path to kubeconfig file

**Returns**: Details of the currently active context

### 4. `k8s_check_cluster_access`
**Description**: Check if Kubernetes cluster is accessible

**Parameters**:
- `context` (optional): Kubernetes context name to check

**Returns**: Accessibility status and any errors

**Example Response**:
```json
{
  "accessible": true,
  "status": "✅ Accessible"
}
```

### 5. `k8s_check_whisker_installation`
**Description**: Check if Calico Whisker is installed in the cluster

**Returns**: Comprehensive installation status including namespace and service checks

**Example Response**:
```json
{
  "calico_system_namespace": true,
  "whisker_service": {
    "available": true,
    "details": "Service found with 1 port(s). Whisker port (8081): Available"
  },
  "overall_status": "✅ Fully Installed"
}
```

### 6. `k8s_check_kubeconfig`
**Description**: Check if kubeconfig file exists and get default path

**Parameters**:
- `kubeconfig_path` (optional): Path to kubeconfig file to check

**Returns**: File existence status and paths

**Example Response**:
```json
{
  "default_path": "/Users/username/.kube/config",
  "checked_path": "/Users/username/.kube/config", 
  "exists": true,
  "status": "✅ Found"
}
```

## CLI Commands

All tools are also available as CLI commands:

### Context Management
```bash
# List all contexts
./server k8s-contexts

# Show current context
./server k8s-current-context

# Connect to a specific context
./server k8s-connect --context aks-calico-demo
```

### Cluster Operations
```bash
# Check cluster accessibility
./server k8s-check-cluster --context aks-calico-demo

# Check Whisker installation
./server k8s-check-whisker

# Check kubeconfig file
./server k8s-check-kubeconfig
```

## Implementation Details

### Architecture
- **`internal/kubernetes/service.go`**: Core Kubernetes service providing all functionality
- **`internal/mcp/server.go`**: MCP tool integration 
- **`cmd/server/main.go`**: CLI command implementations

### Key Features
- **Kubeconfig Parsing**: Full YAML parsing with proper error handling
- **Context Management**: Switch between multiple Kubernetes clusters
- **Service Discovery**: Detect Calico Whisker service and port availability
- **Accessibility Checks**: Verify cluster connectivity before operations
- **Path Resolution**: Automatic default kubeconfig path detection

### Dependencies
- `gopkg.in/yaml.v2`: YAML parsing for kubeconfig files
- `kubectl`: External dependency for cluster operations

### Error Handling
- Graceful handling of missing kubeconfig files
- Proper error reporting for cluster connectivity issues
- Validation of context names and cluster accessibility

## Comparison with TypeScript Version

This Go implementation provides equivalent functionality to the original TypeScript version:

| TypeScript Method | Go Implementation | Description |
|-------------------|-------------------|-------------|
| `connect()` | `k8s_connect` | Connect to cluster with context |
| `getAvailableContexts()` | `k8s_get_contexts` | List all contexts |
| `getCurrentContextInfo()` | `k8s_get_current_context` | Get current context |
| `checkServerAccessibility()` | `k8s_check_cluster_access` | Check cluster access |
| `checkWhiskerService()` | `k8s_check_whisker_installation` | Check Whisker installation |
| `kubeconfigExists()` | `k8s_check_kubeconfig` | Verify kubeconfig file |

## Usage Examples

### Through MCP Protocol
```python
# Test the tools via MCP
import json
import subprocess

# Initialize MCP connection
init_request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {"protocolVersion": "2024-11-05", "capabilities": {}}
}

# Call a Kubernetes tool
tool_request = {
    "jsonrpc": "2.0", 
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "k8s_get_contexts",
        "arguments": {}
    }
}
```

### Through CLI
```bash
# Check what clusters are available
./server k8s-contexts

# Connect to a specific cluster
./server k8s-connect --context production-cluster

# Verify Whisker is installed
./server k8s-check-whisker
```

## Testing

Test scripts are provided for validation:
- `test_kubernetes_tools.py`: Test MCP integration
- `test_whisker_check.py`: Test Whisker installation check

All tools have been tested with real Kubernetes clusters and return properly formatted JSON responses.