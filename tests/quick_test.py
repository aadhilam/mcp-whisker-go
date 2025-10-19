#!/usr/bin/env python3
"""
Quick MCP Server Connectivity Test
Usage: python3 quick_test.py
"""

import json
import subprocess
import sys
import os

def quick_connectivity_test():
    """Quick test to verify MCP server is working"""
    
    print("🔍 Quick MCP Server Connectivity Test")
    print("=" * 40)
    
    # Check if binary exists
    server_binary = os.path.join(os.path.dirname(os.path.dirname(__file__)), "mcp-whisker")
    if not os.path.exists(server_binary):
        print("❌ MCP server binary not found")
        print("   Build it with: go build -o mcp-whisker ./cmd/server")
        return False
    
    # Test initialization
    print("\n1. Testing initialization...")
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "quick-test", "version": "1.0"}
        }
    }
    
    try:
        process = subprocess.Popen(
            [server_binary, "server"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        stdout, stderr = process.communicate(input=json.dumps(request) + '\n', timeout=10)
        
        if stdout.strip():
            response = json.loads(stdout.strip())
            if 'result' in response:
                server_info = response['result'].get('serverInfo', {})
                print(f"✅ Server: {server_info.get('name')} v{server_info.get('version')}")
            else:
                print("❌ Initialization failed")
                return False
        else:
            print("❌ No response from server")
            return False
            
    except Exception as e:
        print(f"❌ Connection failed: {e}")
        return False
    
    print("\n🎉 Quick test passed! MCP server is responding correctly.")
    print("\n💡 Run full test suite with: python3 run_all_tests.py")
    return True

if __name__ == "__main__":
    success = quick_connectivity_test()
    sys.exit(0 if success else 1)