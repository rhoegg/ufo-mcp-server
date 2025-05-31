#!/bin/bash

# Test that resources are properly registered

UFO_IP="192.168.1.72"
export UFO_IP

echo "🧪 Testing UFO MCP Server Resources"
echo "==================================="
echo ""

# Create a test session to list resources
cat > /tmp/mcp-resources-test.json << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}
EOF

echo "Listing all available resources..."
echo ""

cd /Users/rhoegg/src/starspace46/ufo
/Users/rhoegg/.local/bin/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file /Users/rhoegg/.local/share/ufo-mcp/effects.json < /tmp/mcp-resources-test.json 2>/dev/null | \
  grep -A20 "\"id\":2" | jq '.result.resources' 2>/dev/null || \
  (/Users/rhoegg/.local/bin/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file /Users/rhoegg/.local/share/ufo-mcp/effects.json < /tmp/mcp-resources-test.json 2>&1 | grep -A20 "resources")

echo ""
echo "✅ You should see two resources:"
echo "   1. ufo://status - UFO Status"
echo "   2. ufo://ledstate - UFO LED State"