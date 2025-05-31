# UFO MCP Server

A Model Context Protocol (MCP) server for controlling Dynatrace UFO lighting devices.

## Features

- **sendRawApi**: Execute raw API commands on the UFO device
- **Effect Management**: Store and manage custom lighting effects  
- **Real-time Events**: Stream state changes and progress updates
- **Resource Access**: Query UFO status and configuration

## Installation

### Build from Source

```bash
go build -o ufo-mcp ./cmd/server
```

### Configuration Options

- `--transport` or `-t`: Transport type (`stdio` or `http`, default: `stdio`)
- `--port`: HTTP port when using http transport (default: `8080`)
- `--ufo-ip`: UFO device IP address (default: `$UFO_IP` or `ufo`)
- `--effects-file`: Path to effects JSON file (default: `/data/effects.json`)

## Claude Desktop Configuration

Add this configuration to your Claude Desktop `claude_desktop_config.json`:

### Option 1: Stdio Transport (Recommended)

```json
{
  "mcpServers": {
    "ufo": {
      "command": "/absolute/path/to/ufo-mcp",
      "args": [
        "--transport", "stdio",
        "--ufo-ip", "192.168.1.100",
        "--effects-file", "/absolute/path/to/effects.json"
      ],
      "env": {
        "UFO_IP": "192.168.1.100"
      }
    }
  }
}
```

### Option 2: HTTP Transport

Start the server in HTTP mode:
```bash
./ufo-mcp --transport http --port 8080 --ufo-ip 192.168.1.100
```

Then configure Claude Desktop:
```json
{
  "mcpServers": {
    "ufo": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-http-proxy",
        "http://localhost:8080/mcp"
      ]
    }
  }
}
```

## Usage Examples

Once configured, you can ask Claude to:

- `"Turn the UFO red"`
- `"Show me the current UFO status"`  
- `"Create a police lights effect"`
- `"List all available lighting effects"`
- `"Execute the rainbow effect for 30 seconds"`

## Current Implementation Status

✅ **Core Infrastructure**
- MCP server framework
- UFO device communication
- Effect storage with persistence
- Event broadcasting system

✅ **Available Tools (1/11)**
- `sendRawApi` - Execute raw UFO API commands

🔲 **Remaining Tools (10/11)**
- `setRingPattern` - Control ring lighting patterns
- `setLogo` - Control Dynatrace logo LED  
- `setBrightness` - Adjust global brightness
- `playEffect` - Run lighting effects with progress
- `stopEffects` - Cancel running effects
- `addEffect` - Create new effects
- `updateEffect` - Modify existing effects  
- `deleteEffect` - Remove effects
- `listEffects` - Show all available effects

✅ **Resources**
- `getStatus` - UFO device status

🔲 **Streaming**
- `stateEvents` - Real-time event stream (SSE)

## Development

### Running Tests

```bash
go test ./...
```

### Testing with Mock UFO

For development without a physical UFO device:

```bash
# Start a mock UFO server
python3 -m http.server 8081 &

# Configure UFO_IP to point to mock
export UFO_IP=localhost:8081
./ufo-mcp --transport stdio
```

## Environment Variables

- `UFO_IP`: UFO device IP address or hostname
- `LOG_LEVEL`: Logging level (default: `info`)

## Architecture

```
cmd/server/          # Main server application
internal/
  device/            # UFO HTTP client
  effects/           # Effect storage & CRUD  
  events/            # Event broadcasting
  tools/             # MCP tool implementations
data/                # Effect storage
```