#!/bin/bash

# Start UFO MCP server in background
echo "Starting UFO MCP server..."
./ufo-mcp --transport http --port 8080 --ufo-ip MY_UFO_IP --effects-file ./data/effects.json &
UFO_PID=$!

# Give the server time to start
sleep 2

# Start ngrok tunnel
echo "Starting ngrok tunnel to ufo.beyond.integration.quest..."
# Use local config if it exists, otherwise use direct command
if [ -f ./ngrok.yml ]; then
    ngrok start ufo-mcp --config ./ngrok.yml
else
    ngrok http 8080 --domain=ufo.beyond.integration.quest
fi

# When ngrok exits, kill the UFO server
kill $UFO_PID
