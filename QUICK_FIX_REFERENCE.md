# Quick Fix Reference: analyze_blocked_flows Tool

## Problem
❌ MCP tool failing with "Tool execution failed"  
✅ CLI command working perfectly

## Root Cause
**stdout contamination** - Non-JSON text corrupting MCP JSON-RPC protocol

## Solution Applied

### 1. Made MCP Server Default (cmd/server/main.go)
```go
rootCmd := &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        log.SetOutput(os.Stderr)  // ← Critical!
        return server.Run(ctx)
    },
    SilenceUsage: true,
}
```

### 2. Fixed Logging (internal/mcp/server.go)
```go
func (s *MCPServer) Run(ctx context.Context) error {
    log.SetOutput(os.Stderr)  // ← All logs to stderr
    // ...
}
```

### 3. Bonus: Fixed Circular References (pkg/types/types.go)
```go
type BlockingPolicy struct {
    TriggerPolicy *PolicyDetail `json:"triggerPolicy,omitempty"`  // ← Was *Policy
    // ...
}
```

## Testing

```bash
# Build
go build -o mcp-whisker-go ./cmd/server

# Test CLI
./mcp-whisker-go analyze-blocked --namespace yaobank

# Test MCP (Python)
python3 test_analyze_blocked_mcp.py

# Run all tests
go test ./...
```

## MCP Golden Rules

1. **stdout = JSON-RPC ONLY** 📤
2. **stderr = Everything Else** 📋
3. **No Help Text on Default Run** 🚫
4. **Silent by Default** 🤐

## Status
✅ Fixed  
✅ Tested  
✅ Documented

## Quick Diagnosis

If MCP tool fails but CLI works:
1. Check for stdout contamination
2. Verify `log.SetOutput(os.Stderr)`
3. Ensure default command doesn't print help
4. Test with `python3 test_analyze_blocked_mcp.py`

---
**Fixed:** October 23, 2025
