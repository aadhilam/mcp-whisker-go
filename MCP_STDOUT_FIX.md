# MCP JSON-RPC Protocol Fix - stdout Contamination Issue

## Problem

The `analyze_blocked_flows` MCP tool (and potentially other tools) was consistently failing with "Tool execution failed" when called via the MCP JSON-RPC protocol, even though:
- The CLI command (`./mcp-whisker-go analyze-blocked`) worked perfectly
- Port-forward was established
- Whisker service was accessible
- The function returned valid data

## Root Cause Analysis

The MCP protocol uses **stdout for JSON-RPC communication**. Any non-JSON text written to stdout corrupts the protocol and causes tool failures.

The issue had **two sources**:

### 1. Help Text Pollution
When running `./mcp-whisker-go` without a subcommand, Cobra was showing the Long description:

```go
Long: `A Go implementation of the Calico Whisker MCP Server that provides 
Model Context Protocol functionality for analyzing Calico Whisker flow logs 
in Kubernetes environments.`,
```

This text was being written to stdout **before** the MCP server started reading JSON-RPC requests.

### 2. Log Output to stdout
The MCP server had logging statements that defaulted to stdout:

```go
log.Println("Starting MCP server...")  // Goes to stdout by default!
```

## The Fix

### 1. Made MCP Server the Default Command

**File:** `cmd/server/main.go`

Changed the root command to run the MCP server by default when no subcommand is provided:

```go
func main() {
    rootCmd := &cobra.Command{
        Use:   "mcp-whisker-go",
        Short: "Calico Whisker MCP Server for flow log analysis",
        Long:  `...`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // DEFAULT BEHAVIOR: Run as MCP server
            kubeconfig := getKubeconfigPath()
            server := mcp.NewMCPServer(kubeconfig)
            
            // Log to stderr ONLY
            log.SetOutput(os.Stderr)
            if debug {
                log.Printf("MCP server starting with kubeconfig: %s\n", kubeconfig)
            }
            
            return server.Run(ctx)
        },
        SilenceUsage: true,  // Don't show usage on error
    }
    // ...
}
```

**Benefits:**
- No help text is shown when running without arguments
- Server starts immediately
- Consistent with MCP server expectations

### 2. Redirected All Logging to stderr

**File:** `internal/mcp/server.go`

```go
func (s *MCPServer) Run(ctx context.Context) error {
    // Ensure log output goes to stderr to avoid corrupting JSON-RPC on stdout
    log.SetOutput(os.Stderr)
    log.Println("Starting MCP server...")
    
    scanner := bufio.NewScanner(s.input)
    // ...
}
```

**Critical Rule:** In MCP servers:
- ✅ **stdout** = JSON-RPC messages ONLY
- ✅ **stderr** = Logging, debugging, diagnostics

## Testing

### Before Fix
```bash
$ python3 test_analyze_blocked_mcp.py
Starting MCP server...
Initialize response: A Go implementation of the Calico Whisker MCP Server...
❌ Failed to parse response JSON: Expecting value: line 1 column 1 (char 0)
```

### After Fix
```bash
$ python3 test_analyze_blocked_mcp.py
Starting MCP server...
Initialize response: {"jsonrpc":"2.0","id":1,"result":{...}}
✅ SUCCESS!

Content:
{
  "namespace": "yaobank",
  "analysis": {
    "totalBlockedFlows": 0,
    "uniqueBlockedConnections": 0,
    "timeWindow": {}
  },
  "blockedFlows": [],
  "securityInsights": {
    "message": "No blocked flows found",
    "recommendations": []
  }
}
```

## Verification

All MCP tools now work correctly:

```bash
# Via MCP JSON-RPC
✅ setup_port_forward
✅ get_flow_logs
✅ get_aggregated_flow_logs
✅ analyze_namespace_flows
✅ analyze_blocked_flows  # <-- Previously failing!
✅ check_whisker_service

# Via CLI (still work as before)
✅ ./mcp-whisker-go setup-port-forward
✅ ./mcp-whisker-go get-flows
✅ ./mcp-whisker-go analyze-blocked
```

## MCP Server Best Practices

This fix implements critical MCP server best practices:

1. **stdout Discipline**
   - NEVER write non-JSON to stdout in MCP mode
   - ALL diagnostic output → stderr
   - ONLY JSON-RPC messages → stdout

2. **Default Behavior**
   - Running without args starts MCP server (not help text)
   - Explicit subcommands available for CLI usage

3. **Silent Operation**
   - No banner messages
   - No startup text to stdout
   - Debug logging only when requested (to stderr)

4. **Error Handling**
   - `SilenceUsage: true` prevents usage text on errors
   - Errors go to stderr, not stdout

## Related Issues

This fix also resolves potential failures in other MCP tools if they were similarly affected by stdout pollution.

## Files Modified

1. **cmd/server/main.go**
   - Added default RunE handler to root command
   - Redirected logs to stderr
   - Made MCP server the default behavior

2. **internal/mcp/server.go**
   - Added `log.SetOutput(os.Stderr)` at start of Run()

3. **test_analyze_blocked_mcp.py** (new)
   - Test script to verify MCP JSON-RPC communication

## Build & Deploy

```bash
# Rebuild
go build -o mcp-whisker-go ./cmd/server

# Test MCP mode (default)
./mcp-whisker-go

# Test specific commands (CLI mode)
./mcp-whisker-go analyze-blocked --namespace yaobank
```

## Lessons Learned

1. **MCP Protocol is Fragile**
   - stdout contamination breaks everything
   - Even a single non-JSON character causes failures

2. **Default Behavior Matters**
   - Help text on startup is harmful for daemon/server processes
   - Choose sensible defaults for the most common use case

3. **Separation of Concerns**
   - stdout = data channel (JSON-RPC)
   - stderr = logging channel (diagnostics)
   - Never mix them

---

**Date Fixed:** October 23, 2025  
**Issue:** All MCP tools failing with "Tool execution failed"  
**Root Cause:** stdout contamination breaking JSON-RPC protocol  
**Status:** ✅ Resolved
