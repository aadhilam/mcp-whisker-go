# MCP Whisker Go# MCP Whisker Go



A Model Context Protocol (MCP) server for analyzing Calico Whisker flow logs in Kubernetes environments.Go implementation of the Calico Whisker MCP Server for flow log analysis.



## 📁 Project Structure## Overview



```This is a Go port of the TypeScript MCP Whisker project, providing Model Context Protocol (MCP) server functionality for analyzing Calico Whisker flow logs in Kubernetes environments.

mcp-whisker-go/

├── cmd/                    # Application entry points## Features

│   └── server/            # MCP server main

├── internal/              # Private application code- **Port Forward Management**: Automatically manages kubectl port-forward to Calico Whisker service

│   ├── mcp/              # MCP protocol implementation- **Flow Log Analysis**: Retrieves and analyzes network flow logs from Calico Whisker

│   ├── whisker/          # Flow log analysis logic- **Aggregated Flow Reports**: Comprehensive traffic analysis with categorization, top sources/destinations, namespace activity, and security posture

│   ├── kubernetes/       # Kubernetes client utilities- **Namespace Filtering**: Generate detailed flow summaries for specific namespaces

│   └── portforward/      # Port-forward management- **Blocked Flow Analysis**: Identify and analyze blocked network flows and their blocking policies

├── pkg/                   # Public library code- **Policy Integration**: Retrieve and analyze Calico network policies that affect traffic flows

│   └── types/            # Shared type definitions

├── docs/                  # Documentation## Installation

│   ├── user-guide/       # User-facing documentation

│   ├── development/      # Development guides```bash

│   └── troubleshooting/  # Fix guides and troubleshooting# Clone the repository

├── examples/              # Example files and usagegit clone https://github.com/aadhilam/mcp-whisker-go

│   └── calico-traces/    # Sample Calico policy trace JSON filescd mcp-whisker-go

├── scripts/               # Utility scripts

│   └── integration-tests/ # Integration test scripts# Build the application

└── build/                 # Build artifacts (gitignored)go build -o mcp-whisker-go ./cmd/server

```

# Or install directly

## 🚀 Quick Startgo install ./cmd/server

```

For detailed setup instructions, see:

- [MCP Client Setup](docs/user-guide/MCP_CLIENT_SETUP.md)## Usage

- [Direct Binary Setup](docs/user-guide/DIRECT_BINARY_SETUP.md)

- [Kubernetes Integration](docs/user-guide/KUBERNETES_INTEGRATION.md)### As MCP Server (Default)



### PrerequisitesThe binary runs as an MCP server by default, using stdin/stdout for JSON-RPC communication:



- Go 1.21+```bash

- Kubernetes cluster with Calico Whisker installed# Run as MCP server (default behavior)

- kubectl configured with cluster access./mcp-whisker-go --kubeconfig ~/.kube/config



### Build# Or explicitly use the 'server' command

./mcp-whisker-go server --kubeconfig ~/.kube/config

```bash```

make build

```**Note:** When running as an MCP server:

- All JSON-RPC messages use stdout

### Run- All logs and diagnostics go to stderr

- No help text or banners are shown

```bash

# Setup port-forward to Whisker service### CLI Commands

./bin/mcp-whisker-go setup-port-forward```bash

# Setup port-forward to Whisker service

# Get flow logs./mcp-whisker-go setup-port-forward --kubeconfig ~/.kube/config

./bin/mcp-whisker-go get-flows

# Get flow logs (raw JSON)

# Analyze namespace./mcp-whisker-go get-flows

./bin/mcp-whisker-go analyze-namespace --namespace production

```# Get aggregated flow logs with traffic analysis (Markdown format)

./mcp-whisker-go get-aggregated-flows

## 📖 Documentation

# Get aggregated flow logs as JSON

- **User Guide**: [docs/user-guide/](docs/user-guide/)./mcp-whisker-go get-aggregated-flows --markdown=false

- **Development**: [docs/development/](docs/development/)

- **Troubleshooting**: [docs/troubleshooting/](docs/troubleshooting/)# Get aggregated flow logs with time filtering

./mcp-whisker-go get-aggregated-flows --start-time "2025-10-17T14:00:00Z" --end-time "2025-10-17T15:00:00Z"

## 🧪 Testing

# Analyze flows for a specific namespace

```bash./mcp-whisker-go analyze-namespace --namespace production

# Run all tests

make test# Analyze blocked flows

./mcp-whisker-go analyze-blocked --namespace production

# Run with coverage```

make test-coverage

## Dependencies

# Run integration tests

cd scripts/integration-tests- Go 1.21+

python run_all_tests.py- kubectl configured with access to your Kubernetes cluster

```- Calico Whisker deployed in the cluster (calico-system namespace)



## 🔧 Development## Configuration



See [Development Guide](docs/development/DEVELOPMENT.md) for detailed development instructions.The service expects:

- Calico Whisker service running in `calico-system` namespace

```bash- Service accessible on port 8081

# Setup development environment- kubectl access with permissions to port-forward and read network policies

make dev-setup

## Development

# Format code

make fmt```bash

# Run tests

# Lint codego test ./...

make lint

```# Run with development flags

go run ./cmd/server --kubeconfig ~/.kube/config --debug

## 📝 License```



See [LICENSE](docs/LICENSE) for details.## Aggregated Flow Logs


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