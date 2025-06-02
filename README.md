# UFO MCP Server

Control your Dynatrace UFO device through MCP-compatible clients like Claude Desktop.

**Version 1.0.0** - MCP Specification 2025-03-26

## 🚀 Quick Start

### Prerequisites
- A Dynatrace UFO device on your network
- Find your UFO's IP address (check your router or use `ufo.local` if mDNS is available)

## 📦 Installation

### Option 1: Direct Download (Recommended)

```bash
# macOS (Apple Silicon)
curl -L https://github.com/starspace46/ufo/releases/download/v1.0.0/ufo-mcp-darwin-arm64 -o /usr/local/bin/ufo-mcp
chmod +x /usr/local/bin/ufo-mcp

# macOS (Intel)
curl -L https://github.com/starspace46/ufo/releases/download/v1.0.0/ufo-mcp-darwin-amd64 -o /usr/local/bin/ufo-mcp
chmod +x /usr/local/bin/ufo-mcp

# Linux
curl -L https://github.com/starspace46/ufo/releases/download/v1.0.0/ufo-mcp-linux-amd64 -o /usr/local/bin/ufo-mcp
chmod +x /usr/local/bin/ufo-mcp
```

### Option 2: Docker

```bash
docker pull starspace46/ufo-mcp:latest

# Run with Docker
docker run -d \
  --name ufo-mcp \
  -e UFO_IP=YOUR_UFO_IP_HERE \
  -v ~/.ufo-effects:/data \
  starspace46/ufo-mcp:latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/starspace46/ufo-mcp-server.git
cd ufo-mcp-server
go build -o ufo-mcp ./cmd/server
sudo mv ufo-mcp /usr/local/bin/
```

## 🔧 Client Setup

### Claude Desktop

Add to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`  
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

#### Direct Installation
```json
{
  "mcpServers": {
    "ufo": {
      "command": "/usr/local/bin/ufo-mcp",
      "args": [
        "--transport", "stdio",
        "--ufo-ip", "YOUR_UFO_IP_HERE"
      ]
    }
  }
}
```

#### Docker Installation
```json
{
  "mcpServers": {
    "ufo": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "UFO_IP=YOUR_UFO_IP_HERE",
        "-v", "~/.ufo-effects:/data",
        "starspace46/ufo-mcp:latest",
        "--transport", "stdio"
      ]
    }
  }
}
```

### Cline (VS Code Extension)

Add to VS Code settings.json:

#### Direct Installation
```json
{
  "mcp.servers": {
    "ufo": {
      "command": "/usr/local/bin/ufo-mcp",
      "args": ["--transport", "stdio", "--ufo-ip", "YOUR_UFO_IP_HERE"]
    }
  }
}
```

#### Docker Installation
```json
{
  "mcp.servers": {
    "ufo": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "UFO_IP=YOUR_UFO_IP_HERE",
        "-v", "~/.ufo-effects:/data",
        "starspace46/ufo-mcp:latest",
        "--transport", "stdio"
      ]
    }
  }
}
```

### Other MCP Clients

For HTTP-based clients, start the server in HTTP mode:

```bash
# Direct
ufo-mcp --transport http --port 8080 --ufo-ip YOUR_UFO_IP_HERE

# Docker
docker run -d \
  --name ufo-mcp-http \
  -p 8080:8080 \
  -e UFO_IP=YOUR_UFO_IP_HERE \
  -v ~/.ufo-effects:/data \
  starspace46/ufo-mcp:latest \
  --transport http
```

Then configure your client to connect to `http://localhost:8080/mcp`

## 💡 Usage Examples

Once configured, ask Claude to:

- **"Turn the UFO red"** - Sets all LEDs to red
- **"Play the rainbow effect"** - Starts a colorful animation
- **"Make the top ring blue and bottom ring green"** - Independent ring control
- **"Set brightness to 50%"** - Adjust overall brightness
- **"Add rotation to the top ring"** - Animate ring patterns
- **"Show current UFO state"** - See what's currently displayed

## 🎨 Built-in Effects

The server includes pre-configured effects:

- **rainbow** - Rotating rainbow colors (perpetual)
- **breathingGreen** - Calming green pulse (perpetual)
- **policeLights** - Emergency light pattern (30 seconds)
- **oceanWave** - Soothing blue waves (perpetual)
- **fireGlow** - Flickering fire effect (perpetual)
- **alertPulse** - Red alert flash (20 seconds)

## 🛠️ Available Tools

- `configureLighting` - One-command UFO configuration
- `setRingPattern` - Advanced ring control
- `playEffect` / `stopEffects` - Effect management
- `getLedState` - Query current LED state
- `listEffects` - Show available effects
- Plus CRUD operations for custom effects

## 📝 Configuration Options

### Command Line Arguments
- `--transport` or `-t`: Transport type (`stdio` or `http`, default: `stdio`)
- `--port`: HTTP port when using http transport (default: `8080`)
- `--ufo-ip`: UFO device IP address (overrides `UFO_IP` env var)
- `--effects-file`: Path to effects JSON file (default: `/data/effects.json`)

### Environment Variables
- `UFO_IP`: UFO device IP address or hostname (e.g., `192.168.1.100` or `ufo.local`)

## 🐛 Troubleshooting

1. **Can't find UFO**: Check your UFO's IP address in your router's admin panel
2. **Connection refused**: Ensure UFO is powered on and connected to your network
3. **Effects not saving**: Check write permissions for the effects file location
4. **Docker permission issues**: Ensure the effects directory is writable

## 📄 License

MIT License - See LICENSE file for details