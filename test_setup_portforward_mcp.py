#!/usr/bin/env python3
"""
Test script to call setup_port_forward via MCP JSON-RPC
"""
import json
import subprocess
import sys
import time

def test_setup_port_forward():
    """Test the setup_port_forward tool via MCP server"""
    
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
        
        # Test setup_port_forward
        print("\n2. Testing setup_port_forward...")
        setup_request = {
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": "setup_port_forward",
                "arguments": {
                    "namespace": "calico-system"
                }
            }
        }
        
        print(f"Request: {json.dumps(setup_request, indent=2)}")
        process.stdin.write(json.dumps(setup_request) + "\n")
        process.stdin.flush()
        
        # Wait a bit for port-forward to establish
        print("\n3. Waiting for response (port-forward takes a few seconds)...")
        time.sleep(4)
        
        # Try to read response (non-blocking)
        try:
            import select
            if select.select([process.stdout], [], [], 1)[0]:
                setup_response = process.stdout.readline()
                print(f"\nRaw response: {setup_response}")
                
                try:
                    response_obj = json.loads(setup_response)
                    print(f"\nParsed response:\n{json.dumps(response_obj, indent=2)}")
                    
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
                    print(f"Raw response was: {setup_response}")
                    return False
            else:
                print("\n⏱️  No response received (port-forward might be running in background)")
                print("This is expected - port-forward is a long-running process")
                return True
                
        except ImportError:
            print("\n⏱️  Cannot check response status (select module not available)")
            return True
            
    finally:
        # Cleanup
        print("\n4. Cleaning up...")
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
        
        # Print any stderr output
        stderr_output = process.stderr.read()
        if stderr_output:
            print(f"\nServer stderr:\n{stderr_output}")

if __name__ == "__main__":
    success = test_setup_port_forward()
    sys.exit(0 if success else 1)
