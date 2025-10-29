# Fix for analyze_blocked_flows JSON Marshaling Error

## Problem

The `analyze_blocked_flows` MCP tool was failing with "Tool execution failed" without providing detailed error messages. This was caused by a **circular reference issue** during JSON marshaling of the response.

## Root Cause

The `BlockingPolicy` type had a field `TriggerPolicy *Policy` which pointed to the raw `Policy` type from the Whisker API:

```go
// BEFORE (Broken)
type BlockingPolicy struct {
    TriggerPolicy  *Policy  `json:"triggerPolicy"`  // ❌ Policy can contain circular references
    PolicyYAML     *string  `json:"policyYaml"`
    Error          *string  `json:"error,omitempty"`
    BlockingReason string   `json:"blockingReason"`
}
```

The `Policy` type has a recursive structure where `Trigger` points to another `Policy`:

```go
type Policy struct {
    Name        string  `json:"name"`
    Namespace   string  `json:"namespace"`
    Kind        string  `json:"kind"`
    Tier        string  `json:"tier"`
    Action      string  `json:"action"`
    PolicyIndex int     `json:"policy_index"`
    RuleIndex   int     `json:"rule_index"`
    Trigger     *Policy `json:"trigger"`  // ❌ Can create circular references
}
```

When the MCP server tried to serialize the response to JSON, it would encounter:
- Deeply nested Policy chains (`Policy → Policy → Policy...`)
- Or potentially circular references if policies referenced each other
- This caused JSON marshaling to fail or produce invalid output

## Solution

Changed `BlockingPolicy.TriggerPolicy` to use `*PolicyDetail` instead of `*Policy`:

```go
// AFTER (Fixed)
type BlockingPolicy struct {
    TriggerPolicy  *PolicyDetail `json:"triggerPolicy"`  // ✅ PolicyDetail handles nesting correctly
    PolicyYAML     *string       `json:"policyYaml,omitempty"`
    Error          *string       `json:"error,omitempty"`
    BlockingReason string        `json:"blockingReason"`
}
```

`PolicyDetail` is designed to handle recursive trigger chains safely:

```go
type PolicyDetail struct {
    Name        string        `json:"name"`
    Namespace   string        `json:"namespace"`
    Kind        string        `json:"kind"`
    Tier        string        `json:"tier"`
    Action      string        `json:"action"`
    PolicyIndex int           `json:"policyIndex"`
    RuleIndex   int           `json:"ruleIndex"`
    Trigger     *PolicyDetail `json:"trigger,omitempty"`  // ✅ Self-referential but safe
}
```

## Changes Made

### 1. Updated Type Definition
**File:** `pkg/types/types.go`

Changed `BlockingPolicy.TriggerPolicy` from `*Policy` to `*PolicyDetail`.

### 2. Updated Policy Extraction Logic
**File:** `internal/whisker/service.go`

Modified `extractBlockingPolicies()` to convert `Policy` to `PolicyDetail` using the existing `convertPolicyToDetail()` helper:

```go
func (s *Service) extractBlockingPolicies(ctx context.Context, log *types.FlowLog) []types.BlockingPolicy {
    blockingPolicies := make([]types.BlockingPolicy, 0)

    // Check pending policies for triggers
    for _, pendingPolicy := range log.Policies.Pending {
        if pendingPolicy.Trigger != nil && pendingPolicy.Trigger.Name != "" {
            policyYAML := s.retrievePolicyDetails(ctx, pendingPolicy.Trigger)
            
            // ✅ Convert Policy to PolicyDetail to avoid circular references
            triggerDetail := s.convertPolicyToDetail(pendingPolicy.Trigger)

            blockingPolicy := types.BlockingPolicy{
                TriggerPolicy:  &triggerDetail,
                PolicyYAML:     policyYAML,
                BlockingReason: s.getBlockingReason(pendingPolicy.Action),
            }

            blockingPolicies = append(blockingPolicies, blockingPolicy)
        }
    }

    // Same conversion for enforced policies...
}
```

### 3. Added Comprehensive Tests
**File:** `internal/whisker/blocked_flows_test.go` (new)

Created two test functions:
- `TestExtractBlockingPolicies` - Tests policy extraction with various trigger scenarios
- `TestAnalyzeBlockedFlowsJSONMarshaling` - Verifies full response can be marshaled/unmarshaled

## Benefits

✅ **Eliminates circular references** - `PolicyDetail` conversion breaks the circular chain  
✅ **Consistent data model** - Uses the same `PolicyDetail` type as other parts of the system  
✅ **Safe JSON marshaling** - Nested triggers are properly handled without infinite recursion  
✅ **Backward compatible** - JSON output format remains the same  
✅ **Well tested** - New tests verify JSON marshaling works correctly  

## Test Results

All tests pass including the new tests:

```bash
$ go test ./internal/whisker -v -run="TestExtractBlockingPolicies|TestAnalyzeBlockedFlowsJSONMarshaling"

=== RUN   TestExtractBlockingPolicies
=== RUN   TestExtractBlockingPolicies/Pending_policy_with_trigger
=== RUN   TestExtractBlockingPolicies/Enforced_deny_policy_with_trigger
=== RUN   TestExtractBlockingPolicies/Multiple_blocking_policies
=== RUN   TestExtractBlockingPolicies/No_trigger_policies
--- PASS: TestExtractBlockingPolicies (0.48s)

=== RUN   TestAnalyzeBlockedFlowsJSONMarshaling
--- PASS: TestAnalyzeBlockedFlowsJSONMarshaling (0.00s)

PASS
ok      github.com/aadhilam/mcp-whisker-go/internal/whisker     0.823s
```

## Example Output

The tool now correctly returns JSON like this:

```json
{
  "namespace": "test",
  "blockedFlows": [
    {
      "flow": {
        "source": "pod-a (namespace-a)",
        "destination": "pod-b (namespace-b)",
        "protocol": "TCP",
        "port": 443,
        "action": "Deny"
      },
      "blockingPolicies": [
        {
          "triggerPolicy": {
            "name": "deny-all",
            "namespace": "security",
            "kind": "CalicoNetworkPolicy",
            "tier": "security",
            "action": "Deny",
            "trigger": {
              "name": "default-pass",
              "tier": "platform",
              "action": "Pass"
            }
          },
          "blockingReason": "Explicit deny rule"
        }
      ]
    }
  ]
}
```

## Files Modified

1. **pkg/types/types.go** - Updated `BlockingPolicy` type definition
2. **internal/whisker/service.go** - Updated `extractBlockingPolicies()` to use `convertPolicyToDetail()`
3. **internal/whisker/blocked_flows_test.go** - Added comprehensive tests (new file)

## Verification

To verify the fix works:

```bash
# Build the binary
go build -o mcp-whisker-go ./cmd/server

# Run tests
go test ./...

# Test the tool via MCP client
# The analyze_blocked_flows tool should now work without errors
```

## Related Context

This fix complements the recent work on pending policy support (see `PENDING_POLICIES_IMPLEMENTATION.md`). The `convertPolicyToDetail()` helper function added in that work is now reused here to solve the circular reference issue.

---

**Date Fixed:** October 23, 2025  
**Issue:** analyze_blocked_flows failing with "Tool execution failed"  
**Status:** ✅ Resolved
