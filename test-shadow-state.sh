#!/bin/bash

# Test script for UFO MCP Server shadow state functionality

MCP_URL="http://localhost:8080/mcp"
UFO_IP="192.168.1.72"

echo "🧪 Testing UFO MCP Server Shadow State"
echo "======================================"
echo "UFO IP: $UFO_IP"
echo "MCP URL: $MCP_URL"
echo ""

# Helper function to make MCP calls
mcp_call() {
    local method=$1
    local params=$2
    local id=$3
    
    curl -s -X POST $MCP_URL \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\",\"params\":$params}" | jq .
}

# 1. Initialize
echo "1️⃣ Initializing MCP connection..."
mcp_call "initialize" '{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}' 1

# 2. Test setBrightness (updates shadow state)
echo -e "\n2️⃣ Testing setBrightness (should update shadow state)..."
mcp_call "tools/call" '{"name":"setBrightness","arguments":{"level":200}}' 2

# 3. Test setLogo (updates shadow state)  
echo -e "\n3️⃣ Testing setLogo (should update shadow state)..."
mcp_call "tools/call" '{"name":"setLogo","arguments":{"state":"on"}}' 3

# 4. Test setRingPattern (updates shadow state)
echo -e "\n4️⃣ Testing setRingPattern (should update shadow state and emit ring_update)..."
mcp_call "tools/call" '{"name":"setRingPattern","arguments":{"ring":"top","segments":["0|5|FF0000","5|5|00FF00","10|5|0000FF"]}}' 4

# 5. Get UFO Status
echo -e "\n5️⃣ Getting UFO status..."
mcp_call "resources/read" '{"uri":"ufo://status"}' 5

# 6. Test direct API access
echo -e "\n6️⃣ Testing direct UFO API access..."
curl -s "http://$UFO_IP/api?effect=rainbow" | head -5

echo -e "\n✅ Test complete!"
echo ""
echo "📋 What to verify:"
echo "1. setBrightness should have updated the UFO brightness to 200/255"
echo "2. setLogo should have turned on the logo LED"
echo "3. setRingPattern should have set top ring to red/green/blue segments"
echo "4. Check server logs for 'ring_update' events"
echo ""
echo "🔍 To monitor events in real-time, the server needs SSE endpoint (not yet implemented)"