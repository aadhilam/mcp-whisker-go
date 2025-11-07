#!/usr/bin/env python3
"""
Test script to call analyze_blocked_flows via MCP JSON-RPC
"""
import json
import subprocess
import sys

def test_analyze_blocked_flows():
    """Test the analyze_blocked_flows tool via MCP server"""
    
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
        print(f"Initialize response: {init_response}")
        
        # Send analyze_blocked_flows request
        analyze_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": "analyze_blocked_flows",
                "arguments": {
                    "namespace": "yaobank",
                    "setup_port_forward": False  # Assume already setup
                }
            }
        }
        
        print("\n2. Sending analyze_blocked_flows request...")
        print(f"Request: {json.dumps(analyze_request, indent=2)}")
        process.stdin.write(json.dumps(analyze_request) + "\n")
        process.stdin.flush()
        
        # Read response
        print("\n3. Waiting for response...")
        analyze_response = process.stdout.readline()
        print(f"\nRaw response: {analyze_response}")
        
        # Parse and pretty print
        try:
            response_obj = json.loads(analyze_response)
            print(f"\nParsed response:\n{json.dumps(response_obj, indent=2)}")
            
            # Check for errors
            if "error" in response_obj:
                print(f"\n❌ ERROR: {response_obj['error']}")
                return False
            elif "result" in response_obj:
                print(f"\n✅ SUCCESS!")
                if "content" in response_obj["result"]:
                    for content_item in response_obj["result"]["content"]:
                        if content_item.get("type") == "text":
                            print(f"\nContent:\n{content_item['text']}")
                return True
            else:
                print("\n⚠️  Unexpected response format")
                return False
                
        except json.JSONDecodeError as e:
            print(f"\n❌ Failed to parse response JSON: {e}")
            return False
            
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
    success = test_analyze_blocked_flows()
    sys.exit(0 if success else 1)
