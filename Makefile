# UFO MCP Server Makefile

# Variables
BINARY_NAME=ufo-mcp
MAIN_PATH=./cmd/server
BUILD_DIR=./build
INSTALL_DIR=$(HOME)/.local/bin
DATA_DIR=$(HOME)/.local/share/ufo-mcp

# Version info
VERSION := v1.0.1
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
SPEC_VERSION := 2025-03-26

# Build flags
LDFLAGS := -ldflags "-X github.com/starspace46/ufo-mcp-go/internal/version.Version=$(VERSION) \
	-X github.com/starspace46/ufo-mcp-go/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/starspace46/ufo-mcp-go/internal/version.BuildTime=$(BUILD_TIME) \
	-X github.com/starspace46/ufo-mcp-go/internal/version.SpecVersion=$(SPEC_VERSION)"

# Go variables
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
GO_MOD_FILES=go.mod go.sum

.PHONY: all build test clean install uninstall run-stdio run-http deps check configure dxt

# Default target
all: build

# Build the binary
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES) $(GO_MOD_FILES)
	@echo "🔨 Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Built $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./... -v -race
	@echo "✅ All tests passed"

# Test with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Install to local bin directory
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_DIR)
	@mkdir -p $(DATA_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@if [ -f ./data/effects.json ] && [ ! -f $(DATA_DIR)/effects.json ]; then \
		cp ./data/effects.json $(DATA_DIR)/effects.json; \
		echo "✅ Default effects copied to $(DATA_DIR)/effects.json"; \
	fi
	@echo "✅ Installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "💡 Make sure $(INSTALL_DIR) is in your PATH"

# Uninstall from local bin directory
uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Uninstalled $(BINARY_NAME)"

# Run in stdio mode for development
run-stdio: build
	@echo "🚀 Starting UFO MCP Server (stdio mode)..."
	$(BUILD_DIR)/$(BINARY_NAME) --transport stdio --effects-file ./data/effects.json --ufo-ip ${UFO_IP:-localhost}

# Run in HTTP mode for development  
run-http: build
	@echo "🚀 Starting UFO MCP Server (HTTP mode on :8080)..."
	$(BUILD_DIR)/$(BINARY_NAME) --transport http --port 8080 --effects-file ./data/effects.json --ufo-ip ${UFO_IP:-localhost}

# Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies updated"

# Check code quality
check:
	@echo "🔍 Running code quality checks..."
	go vet ./...
	go fmt ./...
	@echo "✅ Code quality checks passed"

# Configure Claude Desktop
configure: install
	@echo "⚙️  Configuring Claude Desktop..."
	./configure-claude.sh

# Development workflow - ALWAYS run tests
dev: clean deps check test build
	@echo "🎉 Development build complete!"

# Pre-commit workflow - run before any code changes
pre-commit: check test
	@echo "✅ Pre-commit checks passed!"

# Code change workflow - MANDATORY before any install/deploy
change: clean test build
	@echo "✅ Code changes validated!"

# Release build (optimized)
release:
	@echo "🚀 Building release version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✅ Release builds complete in $(BUILD_DIR)/"

# Docker build
docker:
	@echo "🐳 Building Docker image..."
	docker build -t starspace46/mcp-server:$(VERSION) -t starspace46/mcp-server:latest .
	@echo "✅ Docker image built: starspace46/mcp-server:$(VERSION)"

# Build Desktop Extension (DXT)
dxt:
	@echo "🛸 Building UFO MCP Desktop Extension..."
	@if ! command -v node >/dev/null 2>&1; then \
		echo "❌ Error: Node.js is required to build the extension"; \
		echo "Please install Node.js from https://nodejs.org/"; \
		exit 1; \
	fi
	@echo "📦 Installing extension dependencies..."
	cd extension && npm install
	@echo "🔨 Building extension package..."
	cd extension && npm run build
	@echo "✅ Desktop Extension built: build/ufo-mcp.dxt"
	@echo ""
	@echo "📱 To install in Claude Desktop:"
	@echo "1. Open Claude Desktop"
	@echo "2. Go to Settings > Extensions"
	@echo "3. Drag build/ufo-mcp.dxt into the extensions area"
	@echo ""
	@echo "🚀 Before using the extension:"
	@echo "Make sure the UFO MCP HTTP server is running:"
	@echo "docker run -d --name ufo-mcp-shared -p 8080:8080 -v \"\$$(pwd)/data:/data\" ufo-mcp:local --transport http --port 8080"

# Show help
help:
	@echo "UFO MCP Server - Available Make targets:"
	@echo ""
	@echo "Development:"
	@echo "  build         Build the binary"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  clean         Clean build artifacts"
	@echo "  dev           Full development workflow"
	@echo "  check         Run code quality checks"
	@echo "  deps          Download and tidy dependencies"
	@echo ""
	@echo "Installation:"
	@echo "  install       Install to $(INSTALL_DIR)"
	@echo "  uninstall     Remove from $(INSTALL_DIR)"
	@echo "  configure     Install and configure Claude Desktop"
	@echo "  dxt           Build Desktop Extension for Claude Desktop"
	@echo ""
	@echo "Running:"
	@echo "  run-stdio     Run in stdio mode (set UFO_IP env var)"
	@echo "  run-http      Run in HTTP mode on :8080"
	@echo ""
	@echo "Release:"
	@echo "  release       Build optimized binaries for all platforms"
	@echo "  docker        Build Docker image"
	@echo ""
	@echo "Environment variables:"
	@echo "  UFO_IP        UFO device IP address (default: localhost)"