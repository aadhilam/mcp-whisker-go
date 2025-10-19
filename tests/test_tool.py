#!/usr/bin/env python3
"""
Interactive MCP Tool Tester
Usage: python3 test_tool.py <tool_name> [arguments]
"""

import json
import os
import subprocess
import sys

def test_mcp_tool(tool_name, arguments=None):
    """Test a specific MCP tool"""
    
    if arguments is None:
        arguments = {}
    
    # Get server path
    server_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "mcp-whisker")
    
    # MCP request structure
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": tool_name,
            "arguments": arguments
        }
    }
    
    print(f"🧪 Testing MCP tool: {tool_name}")
    print(f"📥 Arguments: {json.dumps(arguments, indent=2)}")
    print("=" * 50)
    
    # Run MCP server
    process = subprocess.Popen(
        [server_path, "server", "--debug"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    request_json = json.dumps(request)
    stdout, stderr = process.communicate(input=request_json + '\n')
    
    # Show server logs
    if stderr:
        print("🔧 Server Debug Output:")
        print(stderr)
        print("=" * 50)
    
    # Parse and display result
    if stdout.strip():
        try:
            response = json.loads(stdout.strip())
            if 'result' in response:
                content = response['result'].get('content', [{}])[0].get('text', '')
                print("✅ Tool executed successfully!")
                print("📤 Result:")
                # Pretty print JSON if possible
                try:
                    parsed_content = json.loads(content)
                    print(json.dumps(parsed_content, indent=2))
                except:
                    print(content)
            elif 'error' in response:
                print(f"❌ Tool failed: {response['error'].get('message', 'Unknown error')}")
            else:
                print(f"🤔 Unexpected response: {response}")
        except json.JSONDecodeError as e:
            print(f"❌ Failed to parse response: {e}")
            print(f"Raw output: {stdout}")
    else:
        print("❌ No output received")

def show_usage():
    """Show available tools and usage examples"""
    tools = {
        "check_whisker_service": "Check if Calico Whisker service is available",
        "setup_port_forward": "Setup port-forward to Calico Whisker service", 
        "get_flow_logs": "Retrieve flow logs from Calico Whisker",
        "analyze_namespace_flows": "Analyze flow logs for a specific namespace",
        "analyze_blocked_flows": "Analyze blocked flows and identify blocking policies"
    }
    
    print("🔧 Available MCP Tools:")
    print("=" * 50)
    for tool, desc in tools.items():
        print(f"• {tool}: {desc}")
    
    print("\n📖 Usage Examples:")
    print("=" * 50)
    print("python3 test_tool.py check_whisker_service")
    print("python3 test_tool.py setup_port_forward")
    print('python3 test_tool.py analyze_namespace_flows \'{"namespace": "kube-system"}\'')
    print('python3 test_tool.py analyze_blocked_flows \'{"namespace": "production"}\'')
    print('python3 test_tool.py get_flow_logs \'{"setup_port_forward": false}\'')

def main():
    if len(sys.argv) < 2:
        show_usage()
        return
    
    tool_name = sys.argv[1]
    arguments = {}
    
    # Parse arguments if provided
    if len(sys.argv) > 2:
        try:
            arguments = json.loads(sys.argv[2])
        except json.JSONDecodeError:
            print(f"❌ Invalid JSON arguments: {sys.argv[2]}")
            return
    
    # Add default arguments for certain tools
    if tool_name == "analyze_namespace_flows" and "namespace" not in arguments:
        arguments["namespace"] = "kube-system"
    
    if tool_name in ["get_flow_logs", "analyze_namespace_flows", "analyze_blocked_flows"]:
        if "setup_port_forward" not in arguments:
            arguments["setup_port_forward"] = True
    
    test_mcp_tool(tool_name, arguments)

if __name__ == "__main__":
    main()