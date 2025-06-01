# UFO MCP Server

A Model Context Protocol (MCP) server for controlling Dynatrace UFO lighting devices.

**Version 1.0.0** - Implements MCP Specification 2025-03-26

## Features

- **Unified Control**: `configureLighting` tool controls entire UFO in one command
- **Perpetual Effects**: Effects can run indefinitely until explicitly stopped
- **Effect Management**: Store and manage custom lighting effects in JSON
- **Shadow State**: Track current LED colors and brightness in memory
- **Real-time Events**: Stream state changes and progress updates
- **Resource Access**: Query UFO status and LED state

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

### Option 2: HTTP Transport (2025-03-26 Spec)

Start the server in HTTP mode:
```bash
./ufo-mcp --transport http --port 8080 --ufo-ip 192.168.1.100
```

The server provides:
- Single streamable HTTP endpoint at `POST /mcp`
- Health check at `GET /healthz`
- HTTP/2 support with streaming responses
- Session management with 30-minute timeout
- JSON-RPC batch request support

Configure Claude Desktop or other MCP clients:
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
- `"Play the breathing green effect"` (runs perpetually)
- `"Configure the UFO with rainbow top and blue bottom"`

## Lighting Effects

Effects are stored in `effects.json` and can be either:
- **Perpetual**: Run indefinitely until another command (`duration: 0, perpetual: true`)
- **Timed**: Run for a specific duration then stop (`duration: X, perpetual: false`)

### Default Effects:
- `rainbow` - Perpetual rotating rainbow colors
- `breathingGreen` - Perpetual pulsing green
- `oceanWave` - Perpetual calming blue wave
- `fireGlow` - Perpetual flickering fire
- `policeLights` - 30-second police light bar
- `alertPulse` - 20-second red alert
- `pipelineDemo` - 10-second two-color demo

## Current Implementation Status

✅ **Core Infrastructure**
- MCP server framework
- UFO device communication
- Effect storage with persistence
- Event broadcasting system

✅ **Available Tools (7/8 exposed)**
- `configureLighting` - Control entire UFO in one command (NEW)
- `sendRawApi` - Execute raw UFO API commands (use dim=0-255 for brightness)
- `setRingPattern` - Control ring lighting patterns
- `setLogo` - Control Dynatrace logo LED  
- `getLedState` - Get current LED shadow state
- `listEffects` - Show all available effects
- `playEffect` - Play a lighting effect by name

🔲 **Remaining Tools (1/8)**
- `stopEffects` - Cancel running effects

💾 **Implemented but not exposed via MCP**
- `setBrightness` - Adjust brightness (use dim parameter in patterns instead)
- `addEffect` - Create new effects (available internally)
- `updateEffect` - Modify existing effects (available internally)
- `deleteEffect` - Remove effects (available internally)

✅ **Resources (2/2)**
- `ufo://status` - UFO device status
- `ufo://ledstate` - Current LED shadow state

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
