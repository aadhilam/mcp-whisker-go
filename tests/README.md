# MCP Whisker Go - Test Suite

This directory contains comprehensive testing scripts for the MCP Whisker Go server.

## Test Scripts Overview

### 🚀 **Quick Tests**

- **`quick_test.py`** - Fast connectivity test (10 seconds)
  ```bash
  python3 quick_test.py
  ```

- **`test_tool.py`** - Interactive single tool testing
  ```bash
  python3 test_tool.py check_whisker_service
  python3 test_tool.py analyze_namespace_flows '{"namespace": "kube-system"}'
  ```

### 🧪 **Comprehensive Tests**

- **`run_all_tests.py`** - Complete test suite for all MCP tools
  ```bash
  python3 run_all_tests.py
  ```

- **`test_mcp_client_with_timeout.py`** - Full MCP client test with timeout handling
  ```bash
  python3 test_mcp_client_with_timeout.py
  ```

### 📊 **Performance Tests**

- **`benchmark.py`** - Performance benchmarking of MCP tools
  ```bash
  python3 benchmark.py
  ```

### 🔧 **Debug Scripts**

- **`debug_mcp.py`** - Minimal debug test for troubleshooting
- **`test_mcp_client.py`** - Original comprehensive test client

## Available MCP Tools

The following tools can be tested:

1. **`check_whisker_service`** - Check if Calico Whisker service is available
2. **`setup_port_forward`** - Setup port-forward to Calico Whisker service  
3. **`get_flow_logs`** - Retrieve flow logs from Calico Whisker
4. **`analyze_namespace_flows`** - Analyze flow logs for a specific namespace
5. **`analyze_blocked_flows`** - Analyze blocked flows and identify blocking policies

## Usage Examples

### Test Individual Tools
```bash
# Check service availability
python3 test_tool.py check_whisker_service

# Analyze flows for kube-system namespace
python3 test_tool.py analyze_namespace_flows '{"namespace": "kube-system"}'

# Find blocked flows in production namespace
python3 test_tool.py analyze_blocked_flows '{"namespace": "production"}'

# Setup port forward
python3 test_tool.py setup_port_forward '{"namespace": "calico-system"}'

# Get raw flow logs (assumes port forward is setup)
python3 test_tool.py get_flow_logs '{"setup_port_forward": false}'
```

### Run Complete Test Suite
```bash
# Quick connectivity check
python3 quick_test.py

# Full test suite
python3 run_all_tests.py

# Performance benchmark
python3 benchmark.py
```

### Manual Testing with Command Line
```bash
# From the parent directory
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"check_whisker_service","arguments":{}}}' | ./mcp-whisker server --debug
```

## Prerequisites

1. **MCP Server Binary**: Ensure `mcp-whisker` binary exists in parent directory
   ```bash
   cd .. && go build -o mcp-whisker ./cmd/server
   ```

2. **Kubernetes Access**: Valid kubeconfig with access to cluster containing Calico Whisker
   ```bash
   kubectl get services -n calico-system whisker
   ```

3. **Python 3**: All test scripts require Python 3

## Troubleshooting

- **"MCP server binary not found"**: Build the server first with `go build -o mcp-whisker ./cmd/server`
- **Port forward timeouts**: Ensure you have network access to the Kubernetes cluster
- **Service not available**: Verify Calico Whisker is deployed and running in your cluster
- **Permission errors**: Check that your kubeconfig has proper RBAC permissions

## Test Output

All tests provide detailed output including:
- ✅ Success indicators
- ❌ Failure indicators  
- 📊 Performance metrics
- 🔧 Debug information
- 📈 Success rates and timing statistics

## Integration Testing

For integration with MCP clients (Claude Desktop, VS Code, etc.), refer to the main `MCP_CLIENT_SETUP.md` file in the parent directory.