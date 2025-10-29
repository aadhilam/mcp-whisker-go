# MCP Whisker Go

Go implementation of the Calico Whisker MCP Server for flow log analysis.

## Overview

This is a Go port of the TypeScript MCP Whisker project, providing Model Context Protocol (MCP) server functionality for analyzing Calico Whisker flow logs in Kubernetes environments.

## Features

- **Port Forward Management**: Automatically manages kubectl port-forward to Calico Whisker service
- **Flow Log Analysis**: Retrieves and analyzes network flow logs from Calico Whisker
- **Aggregated Flow Reports**: Comprehensive traffic analysis with categorization, top sources/destinations, namespace activity, and security posture
- **Namespace Filtering**: Generate detailed flow summaries for specific namespaces
- **Blocked Flow Analysis**: Identify and analyze blocked network flows and their blocking policies
- **Policy Integration**: Retrieve and analyze Calico network policies that affect traffic flows

## Installation

```bash
# Clone the repository
git clone https://github.com/aadhilam/mcp-whisker-go
cd mcp-whisker-go

# Build the application
go build -o mcp-whisker-go ./cmd/server

# Or install directly
go install ./cmd/server
```

## Usage

### As MCP Server (Default)

The binary runs as an MCP server by default, using stdin/stdout for JSON-RPC communication:

```bash
# Run as MCP server (default behavior)
./mcp-whisker-go --kubeconfig ~/.kube/config

# Or explicitly use the 'server' command
./mcp-whisker-go server --kubeconfig ~/.kube/config
```

**Note:** When running as an MCP server:
- All JSON-RPC messages use stdout
- All logs and diagnostics go to stderr
- No help text or banners are shown

### CLI Commands
```bash
# Setup port-forward to Whisker service
./mcp-whisker-go setup-port-forward --kubeconfig ~/.kube/config

# Get flow logs (raw JSON)
./mcp-whisker-go get-flows

# Get aggregated flow logs with traffic analysis (Markdown format)
./mcp-whisker-go get-aggregated-flows

# Get aggregated flow logs as JSON
./mcp-whisker-go get-aggregated-flows --markdown=false

# Get aggregated flow logs with time filtering
./mcp-whisker-go get-aggregated-flows --start-time "2025-10-17T14:00:00Z" --end-time "2025-10-17T15:00:00Z"

# Analyze flows for a specific namespace
./mcp-whisker-go analyze-namespace --namespace production

# Analyze blocked flows
./mcp-whisker-go analyze-blocked --namespace production
```

## Dependencies

- Go 1.21+
- kubectl configured with access to your Kubernetes cluster
- Calico Whisker deployed in the cluster (calico-system namespace)

## Configuration

The service expects:
- Calico Whisker service running in `calico-system` namespace
- Service accessible on port 8081
- kubectl access with permissions to port-forward and read network policies

## Development

```bash
# Run tests
go test ./...

# Run with development flags
go run ./cmd/server --kubeconfig ~/.kube/config --debug
```

## Aggregated Flow Logs

The `get-aggregated-flows` command provides comprehensive traffic analysis with the following views:

### Traffic Overview
- Aggregated flows by source, destination, protocol, port, and action
- Normalized pod names with wildcards for cleaner output
- Network classification (PRIVATE NETWORK, PUBLIC NETWORK)
- Human-readable packet and byte counts

### Traffic by Category
Automatically categorizes traffic into:
- DNS Queries (port 53)
- API/HTTPS (port 443)
- Metrics Collection (ports 10250, 4443)
- Calico Services (calico-system namespace)
- Monitoring (port 9153)
- HTTP, Database, and other traffic types

### Additional Analytics
- **Top Traffic Sources & Destinations**: Ranked by flow count with primary activity identification
- **Namespace Activity**: Ingress/egress flows and traffic volume per namespace
- **Security Posture**: Allowed vs. denied flows with percentages and active policies

### Output Formats
- **Markdown** (default): Human-readable tables perfect for reports and documentation
- **JSON**: Structured data for programmatic processing

### Example Usage
```bash
# Basic usage (Markdown output)
./mcp-whisker-go get-aggregated-flows

# JSON output for scripting
./mcp-whisker-go get-aggregated-flows --markdown=false | jq '.trafficByCategory'

# Time-filtered analysis
./mcp-whisker-go get-aggregated-flows \
  --start-time "2025-10-17T14:00:00Z" \
  --end-time "2025-10-17T15:00:00Z"
```

## Testing

Comprehensive test suite available in the `tests/` directory:

```bash
# Quick connectivity test
cd tests && python3 quick_test.py

# Full test suite  
cd tests && python3 run_all_tests.py

# Interactive launcher with menu
cd tests && python3 launcher.py

# Individual tool testing
cd tests && python3 test_tool.py check_whisker_service
cd tests && python3 test_tool.py analyze_namespace_flows '{"namespace": "kube-system"}'
```

See `tests/README.md` for detailed testing documentation.

## Project Structure

```
├── cmd/
│   └── server/           # Main application entry point
├── internal/
│   ├── whisker/         # Calico Whisker service client
│   ├── portforward/     # Port forwarding functionality
│   └── mcp/             # MCP server implementation
├── pkg/
│   └── types/           # Shared types and interfaces
├── tests/               # Comprehensive test suite
│   ├── launcher.py      # Interactive test launcher
│   ├── quick_test.py    # Fast connectivity test
│   ├── run_all_tests.py # Full test suite
│   ├── test_tool.py     # Individual tool testing
│   └── README.md        # Testing documentation
└── README.md
```

## License

MIT License