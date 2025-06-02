# Contributing to UFO MCP Server

Thank you for your interest in contributing to the UFO MCP Server project!

## Development Setup

### Prerequisites
- Go 1.21 or later
- A Dynatrace UFO device (or use the mock server for testing)
- Git

### Building from Source

```bash
git clone https://github.com/starspace46/ufo.git
cd ufo
go build -o ufo-mcp ./cmd/server
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run specific package tests
go test ./internal/effects
go test ./internal/state
```

### Testing with Mock UFO

For development without a physical UFO device:

```bash
# Start the mock UFO server
go run test/mock-ufo/main.go

# In another terminal, run the MCP server
UFO_IP=http://localhost:8080 go run ./cmd/server --transport stdio
```

## Architecture Overview

### Project Structure

```
ufo/
├── cmd/server/          # Main entry point
├── internal/
│   ├── device/         # UFO HTTP client
│   ├── effects/        # Effect management
│   ├── events/         # Event broadcasting
│   ├── state/          # Shadow state tracking
│   ├── tools/          # MCP tool implementations
│   └── transport/      # MCP transport layers
└── data/               # Default effects storage
```

### Key Components

1. **Device Client** (`internal/device`)
   - HTTP wrapper for UFO API calls
   - Retry logic and timeout handling
   - Morph parameter conversion utilities

2. **Effects System** (`internal/effects`)
   - JSON-based effect storage
   - CRUD operations with mutex protection
   - Auto-migration for legacy formats

3. **State Management** (`internal/state`)
   - In-memory LED state tracking
   - Event emission on state changes
   - Thread-safe operations

4. **MCP Tools** (`internal/tools`)
   - Individual tool implementations
   - Input validation and error handling
   - Progress streaming for long operations

## Making Changes

### Code Style
- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and testable

### Testing Requirements
- Write unit tests for new functionality
- Ensure existing tests pass
- Test with race detector enabled
- Include integration tests where appropriate

### Commit Guidelines
- Use clear, descriptive commit messages
- Reference issues when applicable
- Keep commits focused on single changes
- Test before committing

## Docker Development

### Building the Docker Image

```bash
make docker

# Or manually:
docker build -t ufo-mcp:dev .
```

### Testing Docker Image

```bash
docker run --rm -it \
  -e UFO_IP=YOUR_UFO_IP \
  -v $(pwd)/data:/data \
  ufo-mcp:dev --transport stdio
```

## Debugging

### Enable Debug Logging

```bash
# Set log level (not yet implemented)
LOG_LEVEL=debug ufo-mcp --transport stdio
```

### Common Issues

1. **Connection timeouts**: Check UFO_IP and network connectivity
2. **Effect persistence**: Verify write permissions for effects file
3. **State sync issues**: Check for race conditions with `-race` flag

## Release Process

1. Update version in `internal/version/version.go`
2. Update CHANGELOG.md
3. Run full test suite
4. Build release binaries with `make release`
5. Create GitHub release with binaries
6. Update Docker Hub image

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in GitHub Discussions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.