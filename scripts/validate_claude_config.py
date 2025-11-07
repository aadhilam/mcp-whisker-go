#!/usr/bin/env python3
"""
Claude Desktop MCP Configuration Validator
This script validates your Claude Desktop configuration and tests the MCP server
"""

import json
import os
import sys
import subprocess
from pathlib import Path

def find_claude_config():
    """Find Claude Desktop configuration file"""
    if sys.platform == "darwin":  # macOS
        config_path = Path.home() / "Library/Application Support/Claude/claude_desktop_config.json"
    elif sys.platform == "win32":  # Windows
        config_path = Path(os.environ["APPDATA"]) / "Claude/claude_desktop_config.json"
    else:
        print("❌ Unsupported platform for Claude Desktop")
        return None
    
    return config_path if config_path.exists() else None

def validate_config(config_path):
    """Validate Claude Desktop configuration"""
    print(f"📁 Found config at: {config_path}")
    
    try:
        with open(config_path, 'r') as f:
            config = json.load(f)
    except json.JSONDecodeError as e:
        print(f"❌ Invalid JSON in config file: {e}")
        return None
    except Exception as e:
        print(f"❌ Error reading config file: {e}")
        return None
    
    if "mcpServers" not in config:
        print("⚠️  No 'mcpServers' section found in config")
        return None
    
    whisker_configs = []
    for name, server_config in config["mcpServers"].items():
        if "whisker" in name.lower() or "calico" in name.lower():
            whisker_configs.append((name, server_config))
    
    if not whisker_configs:
        print("⚠️  No Calico Whisker MCP server configuration found")
        return None
    
    print(f"✅ Found {len(whisker_configs)} Whisker MCP server configuration(s)")
    return whisker_configs

def test_server_path(command, args):
    """Test if the MCP server can be executed"""
    print(f"\n🧪 Testing server execution...")
    print(f"Command: {command}")
    print(f"Args: {args}")
    
    # Check if command exists
    if not os.path.exists(command) and not any(os.path.exists(os.path.join(p, command)) for p in os.environ.get("PATH", "").split(os.pathsep)):
        print(f"❌ Command not found: {command}")
        return False
    
    # Test execution with initialize request
    test_request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "config-validator", "version": "1.0"}
        }
    }
    
    try:
        process = subprocess.Popen(
            [command] + args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            timeout=10
        )
        
        stdout, stderr = process.communicate(input=json.dumps(test_request) + '\n')
        
        if stderr:
            print(f"🔧 Server stderr: {stderr}")
        
        if stdout.strip():
            try:
                response = json.loads(stdout.strip())
                if "result" in response:
                    server_info = response["result"].get("serverInfo", {})
                    print(f"✅ Server responding: {server_info.get('name')} v{server_info.get('version')}")
                    return True
                else:
                    print(f"❌ Server returned error: {response.get('error', 'Unknown error')}")
            except json.JSONDecodeError:
                print(f"❌ Invalid JSON response: {stdout}")
        else:
            print("❌ No response from server")
        
    except subprocess.TimeoutExpired:
        print("❌ Server timed out (>10 seconds)")
        process.kill()
    except Exception as e:
        print(f"❌ Error testing server: {e}")
    
    return False

def suggest_fixes(name, config):
    """Suggest configuration fixes"""
    print(f"\n🔧 Suggestions for '{name}':")
    
    command = config.get("command", "")
    args = config.get("args", [])
    
    # Suggest wrapper scripts
    if command.endswith("mcp-whisker"):
        print("💡 Try using a wrapper script instead of the binary directly:")
        base_path = os.path.dirname(command)
        print(f"   Shell wrapper: {base_path}/mcp-whisker-server.sh")
        print(f"   Python wrapper: python3 {base_path}/mcp-whisker-server.py")
    
    # Check kubeconfig
    env = config.get("env", {})
    kubeconfig = env.get("KUBECONFIG", "")
    if kubeconfig and not os.path.exists(kubeconfig):
        print(f"⚠️  KUBECONFIG path not found: {kubeconfig}")
        print(f"   Default would be: {Path.home()}/.kube/config")

def main():
    print("🔍 Claude Desktop MCP Configuration Validator")
    print("=" * 50)
    
    # Find Claude config
    config_path = find_claude_config()
    if not config_path:
        print("❌ Claude Desktop configuration file not found")
        print("Expected locations:")
        print("  macOS: ~/Library/Application Support/Claude/claude_desktop_config.json")
        print("  Windows: %APPDATA%/Claude/claude_desktop_config.json")
        return
    
    # Validate configuration
    whisker_configs = validate_config(config_path)
    if not whisker_configs:
        return
    
    # Test each configuration
    for name, config in whisker_configs:
        print(f"\n🚀 Testing configuration: {name}")
        print("-" * 30)
        
        command = config.get("command", "")
        args = config.get("args", [])
        
        if not command:
            print("❌ No command specified in configuration")
            continue
        
        success = test_server_path(command, args)
        if not success:
            suggest_fixes(name, config)
    
    print(f"\n📋 Configuration Summary:")
    print(f"Config file: {config_path}")
    print(f"Whisker servers found: {len(whisker_configs)}")
    
    print(f"\n💡 For manual testing, use:")
    print(f"cd tests && python3 quick_test.py")

if __name__ == "__main__":
    main()