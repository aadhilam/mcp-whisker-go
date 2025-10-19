# Makefile for MCP Whisker Go

# Variables
BINARY_NAME=mcp-whisker-go
CMD_DIR=./cmd/server
BUILD_DIR=./bin
GO_FILES=$(shell find . -type f -name '*.go')

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run the application (setup port-forward)
.PHONY: run-setup
run-setup: build
	$(BUILD_DIR)/$(BINARY_NAME) setup-port-forward

# Get flow logs
.PHONY: run-flows
run-flows: build
	$(BUILD_DIR)/$(BINARY_NAME) get-flows

# Analyze namespace (requires NAMESPACE variable)
.PHONY: run-analyze
run-analyze: build
	@if [ -z "$(NAMESPACE)" ]; then echo "Error: NAMESPACE variable is required. Use: make run-analyze NAMESPACE=your-namespace"; exit 1; fi
	$(BUILD_DIR)/$(BINARY_NAME) analyze-namespace --namespace $(NAMESPACE)

# Analyze blocked flows
.PHONY: run-blocked
run-blocked: build
	$(BUILD_DIR)/$(BINARY_NAME) analyze-blocked $(if $(NAMESPACE),--namespace $(NAMESPACE))

# Check service status
.PHONY: run-check
run-check: build
	$(BUILD_DIR)/$(BINARY_NAME) check-service

# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint Go code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting Go code..."
	golangci-lint run

# Install golangci-lint
.PHONY: install-lint
install-lint:
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Development setup
.PHONY: dev-setup
dev-setup: deps install-lint
	@echo "Development environment setup complete"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          - Build the application (default)"
	@echo "  build        - Build the binary"
	@echo "  deps         - Install dependencies"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-setup    - Setup port-forward"
	@echo "  run-flows    - Get flow logs"
	@echo "  run-analyze  - Analyze namespace (use NAMESPACE=name)"
	@echo "  run-blocked  - Analyze blocked flows"
	@echo "  run-check    - Check service status"
	@echo "  fmt          - Format Go code"
	@echo "  lint         - Lint Go code"
	@echo "  install-lint - Install golangci-lint"
	@echo "  dev-setup    - Setup development environment"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make run-analyze NAMESPACE=production"
	@echo "  make run-blocked NAMESPACE=staging"