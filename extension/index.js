#!/usr/bin/env node

/**
 * UFO MCP Desktop Extension
 * 
 * This is a stdio-to-HTTP bridge that allows Claude Desktop to connect to
 * the UFO MCP HTTP server. It provides a shared state solution for multi-agent
 * coordination where multiple Claude Desktop instances can control the same UFO.
 */

const axios = require('axios');
const readline = require('readline');

// Configuration
const DEFAULT_SERVER_URL = 'http://localhost:8080/mcp';
const SERVER_URL = process.env.UFO_MCP_SERVER || DEFAULT_SERVER_URL;

class MCPBridge {
    constructor(serverUrl) {
        this.serverUrl = serverUrl;
        this.requestId = 1;
        
        // Setup readline interface for stdio communication
        this.rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout,
            terminal: false
        });
        
        this.setupStdioHandler();
    }
    
    setupStdioHandler() {
        this.rl.on('line', async (line) => {
            try {
                const request = JSON.parse(line.trim());
                const response = await this.handleRequest(request);
                if (response) {
                    console.log(JSON.stringify(response));
                }
            } catch (error) {
                console.error(JSON.stringify({
                    jsonrpc: "2.0",
                    id: null,
                    error: {
                        code: -32700,
                        message: "Parse error",
                        data: error.message
                    }
                }));
            }
        });
        
        this.rl.on('close', () => {
            process.exit(0);
        });
    }
    
    async handleRequest(request) {
        try {
            // Forward the request to the HTTP MCP server
            const httpResponse = await axios.post(this.serverUrl, request, {
                headers: {
                    'Content-Type': 'application/json',
                },
                timeout: 30000 // 30 second timeout
            });
            
            return httpResponse.data;
            
        } catch (error) {
            const isTimeout = error.code === 'ECONNABORTED';
            const isConnectionError = error.code === 'ECONNREFUSED' || error.code === 'ENOTFOUND';
            
            return {
                jsonrpc: "2.0",
                id: request.id,
                error: {
                    code: isTimeout ? -32002 : (isConnectionError ? -32001 : -32000),
                    message: isTimeout ? "Request timeout" : (isConnectionError ? "Connection error" : "Server error"),
                    data: `Cannot connect to UFO MCP server at ${this.serverUrl}: ${error.message}`
                }
            };
        }
    }
    
    async checkServerHealth() {
        try {
            const healthUrl = this.serverUrl.replace('/mcp', '/healthz');
            const response = await axios.get(healthUrl, { timeout: 5000 });
            if (response.status === 200) {
                const health = response.data;
                console.error(`Connected to UFO MCP Server v${health.version} (uptime: ${health.uptime})`);
                return true;
            }
        } catch (error) {
            console.error(`Warning: Cannot reach UFO MCP server at ${this.serverUrl}`);
            console.error(`Make sure the Docker container is running: docker run -d --name ufo-mcp-shared -p 8080:8080 ufo-mcp:local --transport http --port 8080`);
        }
        return false;
    }
    
    start() {
        console.error(`UFO MCP Desktop Extension starting...`);
        console.error(`Server URL: ${this.serverUrl}`);
        
        // Check server health in background (don't block startup)
        this.checkServerHealth();
        
        console.error(`Ready for MCP communication via stdio`);
    }
}

// Handle graceful shutdown
process.on('SIGINT', () => {
    console.error('UFO MCP Extension shutting down...');
    process.exit(0);
});

process.on('SIGTERM', () => {
    console.error('UFO MCP Extension shutting down...');
    process.exit(0);
});

// Start the bridge
const bridge = new MCPBridge(SERVER_URL);
bridge.start();