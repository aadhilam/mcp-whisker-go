#!/usr/bin/env python3
"""
Test script to call get_flow_logs and get_aggregated_flow_logs via MCP JSON-RPC
"""
import json
import subprocess
import sys

def test_mcp_tools():
    """Test the flow log tools via MCP server"""
    
    # Start the MCP server
    print("Starting MCP server...")
    process = subprocess.Popen(
        ["./mcp-whisker-go", "--kubeconfig", "~/.kube/config"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1
    )
    
    try:
        # Send initialize request
        init_request = {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "test-client",
                    "version": "1.0.0"
                }
            }
        }
        
        print("\n1. Sending initialize request...")
        process.stdin.write(json.dumps(init_request) + "\n")
        process.stdin.flush()
        
        init_response = process.stdout.readline()
        print(f"Initialize response: {init_response[:100]}...")
        
        # Test get_flow_logs
        print("\n2. Testing get_flow_logs...")
        flow_logs_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": "get_flow_logs",
                "arguments": {
                    "setup_port_forward": True  # Changed to True!
                }
            }
        }
        
        process.stdin.write(json.dumps(flow_logs_request) + "\n")
        process.stdin.flush()
        
        flow_logs_response = process.stdout.readline()
        print(f"get_flow_logs response: {flow_logs_response[:200]}...")
        
        try:
            response_obj = json.loads(flow_logs_response)
            if "error" in response_obj:
                print(f"❌ get_flow_logs ERROR: {response_obj['error']}")
            else:
                print(f"✅ get_flow_logs SUCCESS")
        except json.JSONDecodeError as e:
            print(f"❌ Failed to parse get_flow_logs response: {e}")
        
        # Test get_aggregated_flow_logs
        print("\n3. Testing get_aggregated_flow_logs...")
        agg_logs_request = {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "get_aggregated_flow_logs",
                "arguments": {
                    "setup_port_forward": True  # Should reuse port-forward from call 2!
                }
            }
        }
        
        process.stdin.write(json.dumps(agg_logs_request) + "\n")
        process.stdin.flush()
        
        agg_logs_response = process.stdout.readline()
        print(f"get_aggregated_flow_logs response: {agg_logs_response[:200]}...")
        
        try:
            response_obj = json.loads(agg_logs_response)
            if "error" in response_obj:
                print(f"❌ get_aggregated_flow_logs ERROR: {response_obj['error']}")
            else:
                print(f"✅ get_aggregated_flow_logs SUCCESS")
        except json.JSONDecodeError as e:
            print(f"❌ Failed to parse get_aggregated_flow_logs response: {e}")
            
    finally:
        # Cleanup
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
        
        # Print any stderr output
        stderr_output = process.stderr.read()
        if stderr_output:
            print(f"\nServer stderr:\n{stderr_output}")

if __name__ == "__main__":
    test_mcp_tools()
