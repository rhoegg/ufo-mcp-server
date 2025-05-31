#!/bin/bash

# Claude Desktop Configuration Helper for UFO MCP Server

set -e

echo "🛸 UFO MCP Server - Claude Desktop Configuration Helper"
echo ""

# Detect OS and find Claude Desktop config location
CONFIG_DIR=""
CONFIG_FILE=""

if [[ "$OSTYPE" == "darwin"* ]]; then
    CONFIG_DIR="$HOME/Library/Application Support/Claude"
    CONFIG_FILE="$CONFIG_DIR/claude_desktop_config.json"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    CONFIG_DIR="$HOME/.config/claude"
    CONFIG_FILE="$CONFIG_DIR/claude_desktop_config.json"
else
    echo "❌ Unsupported OS: $OSTYPE"
    echo "   Please manually configure Claude Desktop"
    exit 1
fi

echo "📁 Claude Desktop config directory: $CONFIG_DIR"
echo "📄 Config file: $CONFIG_FILE"
echo ""

# Check if ufo-mcp is installed
UFO_MCP_PATH=""
if command -v ufo-mcp &> /dev/null; then
    UFO_MCP_PATH=$(which ufo-mcp)
    echo "✅ Found ufo-mcp at: $UFO_MCP_PATH"
elif [[ -f "$HOME/.local/bin/ufo-mcp" ]]; then
    UFO_MCP_PATH="$HOME/.local/bin/ufo-mcp"
    echo "✅ Found ufo-mcp at: $UFO_MCP_PATH"
else
    echo "❌ ufo-mcp not found. Please install it first:"
    echo "   ./install.sh"
    exit 1
fi

# Get UFO IP from user
echo ""
read -p "🌐 Enter your UFO device IP address: " UFO_IP
if [[ -z "$UFO_IP" ]]; then
    echo "❌ UFO IP address is required"
    exit 1
fi

# Data directory
DATA_DIR="$HOME/.local/share/ufo-mcp"
mkdir -p "$DATA_DIR"

# Create Claude Desktop config directory if it doesn't exist
mkdir -p "$CONFIG_DIR"

# Create or update config file
if [[ -f "$CONFIG_FILE" ]]; then
    echo "📝 Backing up existing config to $CONFIG_FILE.backup"
    cp "$CONFIG_FILE" "$CONFIG_FILE.backup"
    
    # Check if config already has ufo server
    if grep -q '"ufo"' "$CONFIG_FILE"; then
        echo "⚠️  UFO server configuration already exists in $CONFIG_FILE"
        echo "   Please update manually or remove the existing configuration first"
        exit 1
    fi
    
    # Add UFO server to existing config
    echo "📝 Adding UFO server to existing Claude Desktop config..."
    
    # Create temporary config with UFO server added
    python3 << EOF
import json
import sys

try:
    with open('$CONFIG_FILE', 'r') as f:
        config = json.load(f)
except:
    config = {}

if 'mcpServers' not in config:
    config['mcpServers'] = {}

config['mcpServers']['ufo'] = {
    'command': '$UFO_MCP_PATH',
    'args': [
        '--transport', 'stdio',
        '--ufo-ip', '$UFO_IP',
        '--effects-file', '$DATA_DIR/effects.json'
    ],
    'env': {
        'UFO_IP': '$UFO_IP'
    }
}

with open('$CONFIG_FILE', 'w') as f:
    json.dump(config, f, indent=2)

print("✅ UFO server added to Claude Desktop configuration")
EOF

else
    echo "📝 Creating new Claude Desktop config..."
    cat > "$CONFIG_FILE" << EOF
{
  "mcpServers": {
    "ufo": {
      "command": "$UFO_MCP_PATH",
      "args": [
        "--transport", "stdio",
        "--ufo-ip", "$UFO_IP",
        "--effects-file", "$DATA_DIR/effects.json"
      ],
      "env": {
        "UFO_IP": "$UFO_IP"
      }
    }
  }
}
EOF
    echo "✅ Claude Desktop config created"
fi

echo ""
echo "🎉 Configuration complete!"
echo ""
echo "📋 Next steps:"
echo "1. Restart Claude Desktop if it's running"
echo "2. Start a new conversation in Claude Desktop"
echo "3. Test the UFO connection with: 'Show me the UFO status'"
echo ""
echo "🔧 Configuration details:"
echo "   UFO IP: $UFO_IP"
echo "   Server path: $UFO_MCP_PATH"
echo "   Effects file: $DATA_DIR/effects.json"
echo "   Config file: $CONFIG_FILE"
echo ""
echo "💡 Example commands to try in Claude Desktop:"
echo "   - 'Show me the current UFO status'"
echo "   - 'Send a raw command to the UFO: effect=rainbow'"
echo "   - 'Turn the UFO logo on'"