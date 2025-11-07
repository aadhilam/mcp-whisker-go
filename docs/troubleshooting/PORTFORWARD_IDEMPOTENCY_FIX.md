# Port-Forward Fixes for MCP Tools

## Problem

The MCP tools `get_flow_logs` and `get_aggregated_flow_logs` were failing when called via the LLM client with multiple errors:

1. `"failed to setup port-forward: port-forward already running"`
2. `"Tool execution failed"` with stdout contamination
3. Tilde (~) path expansion issues

## Root Causes

The `Manager.Setup()` method in `internal/portforward/manager.go` was **not idempotent**. If a port-forward was already running, it would return an error instead of reusing the existing connection:

```go
// BEFORE (Non-idempotent)
func (m *Manager) Setup(ctx context.Context) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    if m.cmd != nil && m.cmd.Process != nil {
        return fmt.Errorf("port-forward already running")  // ❌ ERROR!
    }
    // ...
}
```

### Why This Was a Problem

1. **Default Behavior**: Both `get_flow_logs` and `get_aggregated_flow_logs` default to `setup_port_forward: true`
2. **Consecutive Calls**: If the LLM calls these tools multiple times in a session, the second call would fail
3. **Long-Running Server**: The MCP server process stays running, so the Manager instance persists across tool calls

### Call Sequence That Failed

```
1. LLM calls get_flow_logs → Setup() succeeds, port-forward starts
2. LLM calls get_aggregated_flow_logs → Setup() fails! "port-forward already running"
```

## The Fixes

### Fix 1: Idempotent Setup()

Made `Setup()` **idempotent** - if port-forward is already running, return success immediately:

```go
// AFTER (Idempotent)
func (m *Manager) Setup(ctx context.Context) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    // If port-forward is already running, reuse it (idempotent behavior)
    if m.cmd != nil && m.cmd.Process != nil {
        fmt.Fprintf(os.Stderr, "✅ Port-forward already running, reusing existing connection\n")
        return nil  // ✅ SUCCESS!
    }
    
    // ... rest of setup logic ...
}
```

### Fix 2: Port-Forward stdout → stderr

Port-forward output was going to stdout, corrupting the MCP JSON-RPC protocol:

```go
// BEFORE (Broken)
m.cmd.Stderr = os.Stderr
m.cmd.Stdout = os.Stdout  // ❌ Corrupts JSON-RPC on stdout!

// AFTER (Fixed)
m.cmd.Stderr = os.Stderr
m.cmd.Stdout = os.Stderr  // ✅ All output to stderr
```

**Impact**: kubectl port-forward prints "Forwarding from..." messages which were breaking the protocol.

### Fix 3: Tilde Path Expansion

The `~/.kube/config` path was not being expanded:

```go
// BEFORE (Broken)
func getKubeconfigPath() string {
    if kubeconfigPath != "" {
        return kubeconfigPath  // ❌ Returns "~/.kube/config" literally
    }
    // ...
}

// AFTER (Fixed)
func getKubeconfigPath() string {
    if kubeconfigPath != "" {
        // Expand tilde (~) to home directory
        if strings.HasPrefix(kubeconfigPath, "~/") {
            if home, err := os.UserHomeDir(); err == nil {
                return filepath.Join(home, kubeconfigPath[2:])  // ✅ Expands to full path
            }
        }
        return kubeconfigPath
    }
    // ...
}
```

**Impact**: kubectl couldn't find the kubeconfig file, causing "no such file or directory" errors.

### Benefits of All Fixes

✅ **Idempotent**: Calling `Setup()` multiple times is safe  
✅ **LLM-Friendly**: Tools can be called in any order without failing  
✅ **Efficient**: Reuses existing connections instead of killing and restarting  
✅ **Backward Compatible**: Existing behavior unchanged when port-forward isn't running  
✅ **Clean Protocol**: No stdout contamination  
✅ **Path Compatibility**: Works with ~ paths and absolute paths  

## Additional Improvements

Added debug logging to help diagnose issues:

```go
func (s *MCPServer) getFlowLogs(ctx context.Context, args map[string]interface{}) (string, error) {
    if setupPortForward {
        log.Printf("[get_flow_logs] Setting up port-forward...")  // Debug info
        if err := s.manager.Setup(ctx); err != nil {
            return "", fmt.Errorf("failed to setup port-forward: %w", err)
        }
    }
    // ...
}
```

Logs go to **stderr** so they don't corrupt JSON-RPC on stdout.

## Testing

### Test 1: Consecutive Tool Calls
```bash
$ python3 test_flow_logs_mcp.py
✅ get_flow_logs SUCCESS
✅ get_aggregated_flow_logs SUCCESS
```

### Test 2: Multiple Calls in Same Session
```python
# Call 1
response = mcp_client.call_tool("get_flow_logs", {"setup_port_forward": True})
# ✅ Port-forward established

# Call 2 (immediately after)
response = mcp_client.call_tool("get_aggregated_flow_logs", {"setup_port_forward": True})
# ✅ Reuses existing port-forward (no error!)
```

## Files Modified

1. **internal/portforward/manager.go**
   - Changed `Setup()` to return early if port-forward already running (idempotency)
   - Changed `m.cmd.Stdout` from `os.Stdout` to `os.Stderr` (protocol cleanliness)
   - Added friendly log message

2. **internal/mcp/server.go**
   - Added debug logging for port-forward setup in `getFlowLogs()`
   - Added debug logging for port-forward setup in `getAggregatedFlowLogs()`

3. **cmd/server/main.go**
   - Added tilde (~) path expansion in `getKubeconfigPath()`
   - Added `strings` import

## Best Practices Applied

### Idempotency Pattern

An operation is **idempotent** if calling it multiple times has the same effect as calling it once:

```go
// Non-idempotent (BAD)
func Connect() error {
    if connected {
        return fmt.Errorf("already connected")  // ❌ Error on second call
    }
    // ...
}

// Idempotent (GOOD)
func Connect() error {
    if connected {
        return nil  // ✅ Safe to call multiple times
    }
    // ...
}
```

**Why It Matters for MCP:**
- Tools can be called in any order
- LLM might retry failed operations
- Long-running server means state persists

## Troubleshooting

If you still see port-forward errors, check:

1. **Logs** - Look in stderr for debug messages
2. **Port availability** - Ensure port 8081 isn't blocked
3. **kubectl access** - Verify kubectl can access the cluster
4. **Whisker service** - Check if calico-system/whisker exists

Run with `--debug` flag to see more details:

```bash
./mcp-whisker-go --debug --kubeconfig ~/.kube/config
```

## Related Fixes

This builds on the previous stdout contamination fix (see `MCP_STDOUT_FIX.md`), ensuring the MCP protocol remains clean and robust.

---

**Date Fixed:** October 23, 2025  
**Issue:** get_flow_logs and get_aggregated_flow_logs failing  
**Root Cause:** Non-idempotent port-forward setup  
**Status:** ✅ Resolved
