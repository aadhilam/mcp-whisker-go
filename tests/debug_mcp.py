#!/usr/bin/env python3
"""
Minimal MCP test to debug port forward issues
"""

import json
import os
import subprocess
import sys

def test_mcp_server():
    server_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "mcp-whisker")
    server_command = [server_path, "server", "--debug"]
    
    # Test setup_port_forward
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "setup_port_forward",
            "arguments": {"namespace": "calico-system"}
        }
    }
    
    print("🔧 Testing setup_port_forward...")
    print(f"Request: {json.dumps(request, indent=2)}")
    
    process = subprocess.Popen(
        server_command,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    request_json = json.dumps(request)
    stdout, stderr = process.communicate(input=request_json + '\n')
    
    print(f"\n📤 Server stderr:\n{stderr}")
    print(f"\n📥 Server stdout:\n{stdout}")
    
    if stdout.strip():
        try:
            response = json.loads(stdout.strip())
            print(f"\n✅ Response: {json.dumps(response, indent=2)}")
        except json.JSONDecodeError as e:
            print(f"❌ Failed to parse response: {e}")

if __name__ == "__main__":
    test_mcp_server()