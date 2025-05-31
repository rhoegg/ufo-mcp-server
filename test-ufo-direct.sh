#!/bin/bash

# Direct test of UFO MCP server functionality

UFO_IP="192.168.1.72"
export UFO_IP

echo "🧪 Testing UFO MCP Server Direct Communication"
echo "=============================================="
echo "UFO IP: $UFO_IP"
echo ""

# Test 1: Direct UFO API call
echo "1️⃣ Testing direct UFO API connection..."
echo "   Calling: http://$UFO_IP/api?effect=rainbow"
if curl -s --max-time 5 "http://$UFO_IP/api?effect=rainbow" > /dev/null; then
    echo "   ✅ UFO is reachable and responding!"
else
    echo "   ❌ UFO is not reachable at $UFO_IP"
    echo "   Please check the IP address and network connection"
    exit 1
fi

echo ""
echo "2️⃣ Testing MCP server tools..."

# Create a test session with the MCP server
cat > /tmp/mcp-test.json << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"setBrightness","arguments":{"level":200}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"setLogo","arguments":{"state":"on"}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"setRingPattern","arguments":{"ring":"top","segments":["0|5|FF0000","5|5|00FF00","10|5|0000FF"]}}}
{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"ufo://status"}}
EOF

echo "   Running MCP commands..."
cd /Users/rhoegg/src/starspace46/ufo
./build/ufo-mcp --transport stdio --ufo-ip $UFO_IP --effects-file ./data/effects.json < /tmp/mcp-test.json 2>&1 | grep -E "(result|error|brightness|logo|ring)" | head -20

echo ""
echo "3️⃣ Visual verification:"
echo "   - The UFO should now show rainbow effect"
echo "   - Brightness should be at ~78% (200/255)"
echo "   - Logo LED should be ON"
echo "   - Top ring should show Red, Green, Blue segments"
echo ""
echo "✅ Test complete! Check your UFO device for the visual changes."