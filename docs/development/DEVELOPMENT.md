# Development Guide

## Project Structure

```
mcp-whisker-go/
├── cmd/
│   └── server/           # Main application entry point
│       └── main.go
├── internal/             # Private application code
│   ├── portforward/     # Port forwarding functionality
│   │   ├── manager.go
│   │   └── manager_test.go
│   └── whisker/         # Whisker service client
│       ├── service.go
│       └── service_test.go
├── pkg/                 # Public packages
│   └── types/          # Shared types and interfaces
│       ├── types.go
│       └── types_test.go
├── bin/                # Build output (created by make build)
├── go.mod              # Go module definition
├── go.sum              # Go module checksums (generated)
├── Makefile           # Build automation
└── README.md          # Project documentation
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- kubectl configured with access to your Kubernetes cluster
- Calico Whisker deployed in your cluster

### Installation

1. Clone the repository:
```bash
git clone https://github.com/aadhilam/mcp-whisker-go
cd mcp-whisker-go
```

2. Install dependencies:
```bash
make deps
```

3. Build the application:
```bash
make build
```

### Development Workflow

1. **Setup development environment:**
```bash
make dev-setup
```

2. **Format code before committing:**
```bash
make fmt
```

3. **Run tests:**
```bash
make test
```

4. **Run tests with coverage:**
```bash
make test-coverage
```

5. **Lint code:**
```bash
make lint
```

## Testing

### Unit Tests
```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/whisker
```

### Integration Tests
```bash
# Run integration tests (requires kubectl and cluster access)
go test -v ./internal/portforward -tags=integration
```

### Benchmark Tests
```bash
# Run benchmarks
go test -bench=. ./...

# Run benchmarks for specific package
go test -bench=. ./pkg/types
```

## Usage Examples

### Setup Port Forward
```bash
# With default kubeconfig
./bin/mcp-whisker-go setup-port-forward

# With custom kubeconfig
./bin/mcp-whisker-go setup-port-forward --kubeconfig /path/to/config
```

### Get Flow Logs
```bash
./bin/mcp-whisker-go get-flows
```

### Analyze Namespace
```bash
./bin/mcp-whisker-go analyze-namespace --namespace production
```

### Analyze Blocked Flows
```bash
# All namespaces
./bin/mcp-whisker-go analyze-blocked

# Specific namespace
./bin/mcp-whisker-go analyze-blocked --namespace staging
```

### Check Service Status
```bash
./bin/mcp-whisker-go check-service
```

## Architecture

### Component Overview

- **cmd/server**: Main application entry point with CLI commands
- **internal/portforward**: Manages kubectl port-forward operations
- **internal/whisker**: Communicates with Whisker service API
- **pkg/types**: Shared data structures and types

### Key Design Decisions

1. **Separation of Concerns**: Each package has a specific responsibility
2. **Context-Aware**: All operations support context cancellation
3. **Error Handling**: Comprehensive error wrapping and reporting
4. **Testing**: Extensive unit and integration test coverage
5. **CLI-First**: Commands can be used individually or as part of MCP server

### Flow Analysis Algorithm

The flow analysis follows this process:

1. **Data Collection**: Retrieve flow logs from Whisker service
2. **Filtering**: Filter logs by namespace if specified
3. **Aggregation**: Combine logs into unique flows based on:
   - Source name and namespace
   - Destination name and namespace  
   - Protocol and port
   - Action (Allow/Deny)
4. **Policy Analysis**: Extract and analyze applied policies
5. **Statistics Generation**: Calculate traffic and policy statistics
6. **Output Formatting**: Format results with emojis and structured data

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests and linting: `make test && make lint`
5. Format code: `make fmt`
6. Commit changes: `git commit -am 'Add feature'`
7. Push to branch: `git push origin feature-name`
8. Create Pull Request

## Code Style

- Follow standard Go conventions
- Use meaningful variable and function names
- Add comments for public functions and complex logic
- Keep functions small and focused
- Use structured logging where appropriate
- Handle errors explicitly

## Performance Considerations

- Flow aggregation is done in-memory for fast processing
- HTTP client has reasonable timeouts
- Context cancellation prevents hanging operations
- Efficient JSON marshaling/unmarshaling
- Minimal external dependencies