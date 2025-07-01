# UFO MCP Desktop Extension

This Desktop Extension (DXT) enables Claude Desktop to connect to the UFO MCP HTTP server, providing shared state coordination for multi-agent UFO control.

## Overview

The UFO MCP Desktop Extension solves a key limitation: Claude Desktop only supports stdio-based MCP servers, not HTTP ones. This extension acts as a stdio-to-HTTP bridge, allowing multiple Claude Desktop instances to share the same UFO state through a centralized HTTP MCP server.

## Architecture

```
Claude Desktop 1 ──> UFO Extension ──┐
                     (stdio)          │
                                      ├──> UFO MCP HTTP Server ──> UFO Hardware
Claude Desktop 2 ──> UFO Extension ──┘     (shared state)
                     (stdio)
```

## Features

- **Shared State**: Multiple Claude Desktop instances see the same UFO state
- **Multi-Agent Coordination**: Different agents can collaborate on UFO control
- **Effect Stack Management**: Supports layered effects with proper stacking
- **Real-time Updates**: Changes from one agent are visible to others
- **Error Handling**: Robust connection and timeout handling

## Prerequisites

1. **Node.js** (>= 18.0.0) - Required to build and run the extension
2. **UFO MCP HTTP Server** - Must be running and accessible

## Installation

### Step 1: Build the Extension

From the UFO MCP server project root:

```bash
make dxt
```

This will:
- Install Node.js dependencies
- Build the extension package
- Create `build/ufo-mcp.dxt`

### Step 2: Start the HTTP Server

Start the UFO MCP HTTP server:

```bash
docker run -d --name ufo-mcp-shared -p 8080:8080 -v "$(pwd)/data:/data" ufo-mcp:local --transport http --port 8080
```

### Step 3: Install in Claude Desktop

1. Open Claude Desktop
2. Go to **Settings > Extensions**
3. Drag `build/ufo-mcp.dxt` into the extensions area
4. The extension should appear as "UFO MCP Extension"

## Configuration

The extension can be configured with environment variables:

- `UFO_MCP_SERVER`: URL of the UFO MCP HTTP server (default: `http://localhost:8080/mcp`)

In Claude Desktop, you can set this in the extension settings if your server is running on a different host/port.

## Usage

Once installed, the extension provides the same MCP tools as the server:

### Available Tools

- **configureLighting**: Set up complex lighting patterns with morphing and rotation
- **playEffect**: Play predefined lighting effects (rainbow, police lights, etc.)
- **stopEffect**: Stop the current effect and resume the previous one
- **getLedState**: Get the current LED state (shared across all agents)
- **listEffects**: List available lighting effects
- **sendRawApi**: Send raw commands to the UFO hardware

### Example Multi-Agent Workflow

1. **Agent A** configures a base lighting pattern:
   ```
   configureLighting: Set red top ring, blue bottom ring
   ```

2. **Agent B** can see the current state:
   ```
   getLedState: Shows red top, blue bottom
   ```

3. **Agent B** adds a temporary effect:
   ```
   playEffect: rainbow for 10 seconds
   ```

4. **Agent A** can see the rainbow effect when querying state

5. After 10 seconds, both agents see the original red/blue pattern

## Troubleshooting

### Extension Won't Install
- Ensure you have Node.js >= 18.0.0 installed
- Rebuild the extension with `make dxt`
- Check that `build/ufo-mcp.dxt` exists and is not corrupted

### Connection Errors
- Verify the UFO MCP HTTP server is running: `docker ps | grep ufo-mcp`
- Check the server URL in extension settings
- Look at server logs: `docker logs ufo-mcp-shared`

### State Not Shared Between Agents
- Confirm both agents are using the same extension (not different MCP server instances)
- Check that the UFO HTTP server is accessible from both Claude Desktop instances
- Verify you're not running multiple UFO MCP servers on different ports

### Tools Not Available
- Restart Claude Desktop after installing the extension
- Check extension status in Settings > Extensions
- Verify the HTTP server is responding: `curl http://localhost:8080/healthz`

## Development

### Building from Source

```bash
cd extension
npm install
npm run build
```

### Testing the Bridge

Test the stdio-to-HTTP bridge manually:

```bash
cd extension
echo '{"jsonrpc": "2.0", "method": "tools/list", "id": 1}' | node index.js
```

### Debugging

The extension logs debug information to stderr (visible in Claude Desktop's developer tools):
- Connection status to HTTP server
- Request/response flow
- Error details

## Architecture Details

### Bridge Implementation

The extension implements a JSON-RPC bridge:

1. **Input**: Receives stdio JSON-RPC requests from Claude Desktop
2. **Translation**: Forwards requests to HTTP MCP server via POST
3. **Output**: Returns HTTP responses as stdio JSON-RPC responses

### Error Handling

- **Connection errors**: Graceful fallback with informative error messages
- **Timeouts**: 30-second timeout for tool calls, 5-second for health checks
- **Server unavailable**: Clear instructions for starting the server

### State Synchronization

State is maintained by the HTTP MCP server, not the extensions. Each extension is stateless and simply forwards requests, ensuring all agents see the same UFO state.

## Related Documentation

- [UFO MCP Server README](../README.md) - Main server documentation
- [Multi-Agent Coordination](../docs/multi-agent.md) - Advanced coordination patterns
- [UFO API Reference](../UFO_API_REFERENCE.md) - Low-level UFO commands