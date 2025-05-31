#!/bin/bash

# Test the getLedState resource

UFO_IP="192.168.1.72"
export UFO_IP

echo "🧪 Testing UFO MCP Server getLedState Resource"
echo "=============================================="
echo ""

# Create a test session with the MCP server
cat > /tmp/mcp-ledstate-test.json << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"resources/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"setBrightness","arguments":{"level":200}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"setLogo","arguments":{"state":"on"}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"setRingPattern","arguments":{"ring":"top","segments":["0|5|FF0000","5|5|00FF00","10|5|0000FF"]}}}
{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"ufo://ledstate"}}
{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"setRingPattern","arguments":{"ring":"bottom","segments":["0|15|FFFF00"],"background":"800080"}}}
{"jsonrpc":"2.0","id":8,"method":"resources/read","params":{"uri":"ufo://ledstate"}}
EOF

echo "Running MCP commands and checking LED state..."
echo ""

cd /Users/rhoegg/src/starspace46/ufo
./build/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file ./data/effects.json < /tmp/mcp-ledstate-test.json 2>&1 | \
  grep -A20 "\"id\":2" | grep -A5 "ledstate" | head -10

echo ""
echo "Getting LED state after changes..."
./build/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file ./data/effects.json < /tmp/mcp-ledstate-test.json 2>&1 | \
  grep -A50 "\"id\":6" | grep -A30 "ledstate" | head -35

echo ""
echo "Final LED state after bottom ring update..."
./build/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file ./data/effects.json < /tmp/mcp-ledstate-test.json 2>&1 | \
  grep -A50 "\"id\":8" | grep -A30 "ledstate" | head -35

echo ""
echo "✅ The getLedState resource should show:"
echo "   - Brightness: 200"
echo "   - Logo: true (on)"
echo "   - Top ring: Red (FF0000), Green (00FF00), Blue (0000FF) for first 15 LEDs"
echo "   - Bottom ring: Yellow (FFFF00) with purple background (800080)"