#!/bin/bash

# UFO MCP Server Installation Script

set -e

echo "🛸 Installing UFO MCP Server..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.23+ first."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
REQUIRED_VERSION="1.23"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go $REQUIRED_VERSION+ required, found $GO_VERSION"
    exit 1
fi

echo "✅ Go $GO_VERSION detected"

# Create installation directory
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

# Build the server
echo "🔨 Building UFO MCP Server..."
go build -o "$INSTALL_DIR/ufo-mcp" ./cmd/server

echo "✅ UFO MCP Server installed to $INSTALL_DIR/ufo-mcp"

# Create data directory
DATA_DIR="$HOME/.local/share/ufo-mcp"
mkdir -p "$DATA_DIR"

echo "✅ Data directory created at $DATA_DIR"

# Copy default effects if they don't exist
if [ ! -f "$DATA_DIR/effects.json" ]; then
    if [ -f "./data/effects.json" ]; then
        cp ./data/effects.json "$DATA_DIR/effects.json"
        echo "✅ Default effects copied to $DATA_DIR/effects.json"
    else
        echo "⚠️  Default effects.json not found. It will be created on first run."
    fi
fi

# Check if directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  $INSTALL_DIR is not in your PATH"
    echo "   Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "📋 Next steps:"
echo "1. Configure your UFO IP address:"
echo "   export UFO_IP=192.168.1.100  # Replace with your UFO's IP"
echo ""
echo "2. Test the server:"
echo "   ufo-mcp --help"
echo ""
echo "3. Configure Claude Desktop (see README.md for details)"
echo ""
echo "4. Example Claude Desktop config:"
cat << 'EOF'
   {
     "mcpServers": {
       "ufo": {
         "command": "/Users/yourusername/.local/bin/ufo-mcp",
         "args": [
           "--transport", "stdio",
           "--ufo-ip", "YOUR_UFO_IP",
           "--effects-file", "/Users/yourusername/.local/share/ufo-mcp/effects.json"
         ]
       }
     }
   }
EOF