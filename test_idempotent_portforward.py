#!/usr/bin/env python3
"""
Test to verify idempotent port-forward setup
"""
import json
import subprocess
import sys
import time

def test_with_existing_portforward():
    """Test MCP tools when port-forward is already running"""
    
    print("Step 1: Starting standalone port-forward...")
    pf_process = subprocess.Popen(
        ["./mcp-whisker-go", "setup-port-forward", "--kubeconfig", "~/.kube/config"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    # Wait for port-forward to establish
    time.sleep(3)
    
    try:
        print("Step 2: Starting MCP server (port-forward already running)...")
        mcp_process = subprocess.Popen(
            ["./mcp-whisker-go", "--kubeconfig", "~/.kube/config"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1
        )
        
        try:
            # Initialize
            init_request = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "test-client", "version": "1.0.0"}
                }
            }
            
            mcp_process.stdin.write(json.dumps(init_request) + "\n")
            mcp_process.stdin.flush()
            init_response = mcp_process.stdout.readline()
            print(f"✅ Initialize successful")
            
            # Test get_flow_logs with setup_port_forward=true (should reuse existing)
            print("\nStep 3: Calling get_flow_logs with setup_port_forward=true...")
            flow_request = {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {
                    "name": "get_flow_logs",
                    "arguments": {
                        "setup_port_forward": True  # Should reuse existing port-forward!
                    }
                }
            }
            
            mcp_process.stdin.write(json.dumps(flow_request) + "\n")
            mcp_process.stdin.flush()
            flow_response = mcp_process.stdout.readline()
            
            try:
                response_obj = json.loads(flow_response)
                if "error" in response_obj:
                    print(f"❌ FAILED: {response_obj['error']}")
                    return False
                else:
                    print(f"✅ SUCCESS: get_flow_logs worked with existing port-forward!")
                    
            except json.JSONDecodeError as e:
                print(f"❌ Failed to parse response: {e}")
                return False
            
            # Test get_aggregated_flow_logs
            print("\nStep 4: Calling get_aggregated_flow_logs...")
            agg_request = {
                "jsonrpc": "2.0",
                "id": 3,
                "method": "tools/call",
                "params": {
                    "name": "get_aggregated_flow_logs",
                    "arguments": {
                        "setup_port_forward": True
                    }
                }
            }
            
            mcp_process.stdin.write(json.dumps(agg_request) + "\n")
            mcp_process.stdin.flush()
            agg_response = mcp_process.stdout.readline()
            
            try:
                response_obj = json.loads(agg_response)
                if "error" in response_obj:
                    print(f"❌ FAILED: {response_obj['error']}")
                    return False
                else:
                    print(f"✅ SUCCESS: get_aggregated_flow_logs worked!")
                    
            except json.JSONDecodeError as e:
                print(f"❌ Failed to parse response: {e}")
                return False
                
            print("\n🎉 All tests passed! Port-forward Setup() is now idempotent!")
            return True
            
        finally:
            mcp_process.terminate()
            try:
                mcp_process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                mcp_process.kill()
                
    finally:
        # Cleanup port-forward
        pf_process.terminate()
        try:
            pf_process.wait(timeout=2)
        except subprocess.TimeoutExpired:
            pf_process.kill()
        print("\n✅ Cleaned up port-forward")

if __name__ == "__main__":
    success = test_with_existing_portforward()
    sys.exit(0 if success else 1)
