#!/usr/bin/env python3
"""
Simple MCP client to test the Calico Whisker MCP server
Usage: python3 test_mcp_client.py
"""

import json
import subprocess
import sys
import time

class MCPClient:
    def __init__(self, server_command):
        self.server_command = server_command
        self.request_id = 0
        
    def send_request(self, method, params=None):
        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method
        }
        if params:
            request["params"] = params
            
        # Start server process
        process = subprocess.Popen(
            self.server_command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Send request
        request_json = json.dumps(request)
        stdout, stderr = process.communicate(input=request_json + '\n')
        
        if stderr:
            print(f"Server stderr: {stderr}", file=sys.stderr)
            
        if stdout.strip():
            try:
                return json.loads(stdout.strip())
            except json.JSONDecodeError as e:
                print(f"Failed to parse response: {stdout}", file=sys.stderr)
                return None
        
        return None

def main():
    # Path to your MCP server executable
    server_path = "./mcp-whisker"  # Adjust path as needed
    client = MCPClient([server_path, "server"])
    
    print("🚀 Testing MCP Whisker Server")
    print("=" * 40)
    
    # Test 1: Initialize
    print("\n1. Testing initialization...")
    response = client.send_request("initialize", {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "test-client", "version": "1.0.0"}
    })
    
    if response:
        print("✅ Initialize successful")
        print(f"   Server: {response.get('result', {}).get('serverInfo', {}).get('name')}")
        print(f"   Version: {response.get('result', {}).get('serverInfo', {}).get('version')}")
    else:
        print("❌ Initialize failed")
        return
    
    # Test 2: List tools
    print("\n2. Testing tools list...")
    response = client.send_request("tools/list")
    
    if response and 'result' in response:
        tools = response['result'].get('tools', [])
        print(f"✅ Found {len(tools)} tools:")
        for tool in tools:
            print(f"   - {tool['name']}: {tool['description']}")
    else:
        print("❌ Failed to list tools")
        return
    
    # Test 3: Check Whisker service
    print("\n3. Testing check_whisker_service...")
    response = client.send_request("tools/call", {
        "name": "check_whisker_service",
        "arguments": {}
    })
    
    if response and 'result' in response:
        content = response['result'].get('content', [{}])[0].get('text', '')
        print("✅ Service check successful:")
        print(f"   {content}")
    else:
        error = response.get('error', {}) if response else {}
        print(f"❌ Service check failed: {error.get('message', 'Unknown error')}")
    
    # Test 4: Setup port forward (this might fail if not connected to cluster)
    print("\n4. Testing setup_port_forward...")
    response = client.send_request("tools/call", {
        "name": "setup_port_forward",
        "arguments": {"namespace": "calico-system"}
    })
    
    if response and 'result' in response:
        content = response['result'].get('content', [{}])[0].get('text', '')
        print("✅ Port forward setup successful:")
        print(f"   {content}")
        
        # Test 5: Get flow logs (only if port forward worked)
        print("\n5. Testing get_flow_logs...")
        response = client.send_request("tools/call", {
            "name": "get_flow_logs",
            "arguments": {"setup_port_forward": False}  # Don't setup again
        })
        
        if response and 'result' in response:
            content = response['result'].get('content', [{}])[0].get('text', '')
            print("✅ Flow logs retrieved successfully")
            print("   (Output truncated for readability)")
        else:
            error = response.get('error', {}) if response else {}
            print(f"❌ Flow logs failed: {error.get('message', 'Unknown error')}")
    else:
        error = response.get('error', {}) if response else {}
        print(f"❌ Port forward failed: {error.get('message', 'Unknown error')}")
        print("   This is expected if not connected to a cluster with Calico Whisker")
    
    print("\n🎉 MCP Server test completed!")
    print("\n💡 To use with Claude Desktop, add this to your config:")
    print(f'   "command": "{server_path}"')
    print('   "args": ["server"]')

if __name__ == "__main__":
    main()