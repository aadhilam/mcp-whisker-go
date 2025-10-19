#!/usr/bin/env python3
"""
Claude Desktop MCP Simulation Test
This simulates the exact JSON-RPC flow that Claude Desktop uses
"""

import json
import subprocess
import sys
import os

def test_claude_simulation():
    """Simulate Claude Desktop's MCP interaction"""
    
    server_path = "/Users/aadhilamajeed/Library/CloudStorage/OneDrive-Personal/k8/mcp-whisker-go/mcp-whisker"
    
    print("🧪 Simulating Claude Desktop MCP Interaction")
    print("=" * 50)
    
    # Test sequence that mimics Claude Desktop
    test_requests = [
        {
            "name": "Initialize",
            "request": {
                "jsonrpc": "2.0",
                "id": 0,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2025-06-18",
                    "capabilities": {},
                    "clientInfo": {"name": "claude-ai", "version": "0.1.0"}
                }
            }
        },
        {
            "name": "List Tools", 
            "request": {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "tools/list"
            }
        },
        {
            "name": "Call Tool - Check Service",
            "request": {
                "jsonrpc": "2.0",
                "id": 2,
                "method": "tools/call",
                "params": {
                    "name": "check_whisker_service",
                    "arguments": {}
                }
            }
        }
    ]
    
    for test in test_requests:
        print(f"\n📤 {test['name']}...")
        
        try:
            process = subprocess.Popen(
                [server_path, "server"],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            request_json = json.dumps(test['request'])
            print(f"   Request: {request_json}")
            
            stdout, stderr = process.communicate(input=request_json + '\n', timeout=15)
            
            if stderr:
                print(f"   Server logs: {stderr.strip()}")
            
            if stdout.strip():
                try:
                    response = json.loads(stdout.strip())
                    print(f"   ✅ Response ID: {response.get('id')}")
                    print(f"   📦 Response type: {'result' if 'result' in response else 'error'}")
                    
                    # Validate JSON-RPC compliance
                    if 'jsonrpc' not in response or response['jsonrpc'] != '2.0':
                        print("   ❌ Missing or invalid jsonrpc field")
                    
                    if 'id' not in response:
                        print("   ❌ Missing id field")
                    elif response['id'] != test['request']['id']:
                        print(f"   ❌ ID mismatch: expected {test['request']['id']}, got {response['id']}")
                    
                    if 'result' not in response and 'error' not in response:
                        print("   ❌ Missing both result and error fields")
                    
                    print("   ✅ JSON-RPC format valid")
                    
                except json.JSONDecodeError as e:
                    print(f"   ❌ Invalid JSON response: {e}")
                    print(f"   Raw output: {stdout}")
            else:
                print("   ❌ No response received")
                
        except subprocess.TimeoutExpired:
            print("   ❌ Request timed out")
            process.kill()
        except Exception as e:
            print(f"   ❌ Error: {e}")
    
    print(f"\n🎉 Claude Desktop simulation completed!")
    print("If all tests show ✅, the server should work properly with Claude Desktop.")

if __name__ == "__main__":
    test_claude_simulation()