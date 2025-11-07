# MCP Whisker Server Configuration

## Claude Desktop Configuration

Add this to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "calico-whisker": {
      "command": "/Users/aadhilamajeed/Library/CloudStorage/OneDrive-Personal/k8/mcp-whisker-go/mcp-whisker",
      "args": ["server"],
      "env": {
        "KUBECONFIG": "/Users/aadhilamajeed/.kube/config"
      }
    }
  }
}
```

**Note**: This configuration uses the Go binary directly without wrapper scripts. The server has been updated to handle Claude Desktop's protocol requirements natively.

## Troubleshooting Claude Desktop Integration

### Error: "Invalid or unexpected token" 
This error has been resolved. The Go binary now handles Claude Desktop's protocol requirements natively without wrapper scripts.

### Error: "Permission denied"
Make sure the scripts are executable:
```bash
chmod +x mcp-whisker-server.sh
chmod +x mcp-whisker-server.py
chmod +x mcp-whisker
```

### Error: "Server disconnected" 
Check the Claude Desktop logs for specific errors. Common issues:
- Binary not found at specified path
- Kubeconfig path incorrect
- No access to Kubernetes cluster

### Test the server manually:
```bash
# Test with direct binary
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./mcp-whisker server

# Or use the test suite
cd tests && python3 quick_test.py
```

## VS Code with MCP Extension

1. Install the MCP extension for VS Code
2. Add to your VS Code settings.json:

```json
{
  "mcp.servers": [
    {
      "name": "calico-whisker",
      "command": "/path/to/your/mcp-whisker",
      "args": ["server"],
      "env": {
        "KUBECONFIG": "/path/to/your/kubeconfig"
      }
    }
  ]
}
```

## Continue.dev Configuration

Add to your `.continue/config.json`:

```json
{
  "models": [
    // your existing models
  ],
  "mcpServers": [
    {
      "name": "calico-whisker",
      "command": "/path/to/your/mcp-whisker",
      "args": ["server"],
      "env": {
        "KUBECONFIG": "/path/to/your/kubeconfig"
      }
    }
  ]
}
```

## Generic MCP Client Configuration

For any MCP client that supports external servers:

```json
{
  "server": {
    "command": "/path/to/your/mcp-whisker",
    "args": ["server"],
    "env": {
      "KUBECONFIG": "/path/to/your/kubeconfig"
    }
  }
}
```

## Available Tools

Once configured, you'll have access to these tools in your MCP client:

1. **setup_port_forward** - Setup port-forward to Calico Whisker service
2. **get_flow_logs** - Retrieve flow logs from Calico Whisker  
3. **analyze_namespace_flows** - Analyze flows for a specific namespace
4. **analyze_blocked_flows** - Analyze blocked flows and identify blocking policies
5. **check_whisker_service** - Check if Calico Whisker service is available

## Example Usage in Claude Desktop

After configuring, you can ask Claude:

- "Can you check if the Calico Whisker service is available?"
- "Get the current flow logs from Whisker"
- "Analyze flows for the 'production' namespace"
- "Show me any blocked flows and why they're being blocked"
- "Setup port forwarding to the Whisker service"

## Troubleshooting

1. **Command not found**: Make sure the path to `mcp-whisker` executable is absolute
2. **Kubeconfig issues**: Ensure the KUBECONFIG environment variable points to a valid kubeconfig file
3. **Permission denied**: Make sure the `mcp-whisker` executable has execute permissions (`chmod +x mcp-whisker`)
4. **Connection issues**: Verify you have access to the Kubernetes cluster and the calico-system namespace

## Testing the Server

You can test the MCP server manually using stdio:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-whisker server
```

This should return an initialization response confirming the server is working.