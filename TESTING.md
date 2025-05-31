# UFO MCP Server - Manual Testing Guide

This guide walks through manually testing the UFO MCP Server with Claude Desktop.

## Prerequisites

1. **UFO MCP Server installed**:
   ```bash
   make install
   # or
   ./install.sh
   ```

2. **Claude Desktop installed** and configured with the UFO server

3. **UFO device accessible** on your network (or mock server for testing)

## Test Scenarios

### 1. Basic MCP Connection Test

**Objective**: Verify Claude Desktop can connect to the UFO MCP server.

**Steps**:
1. Start a new conversation in Claude Desktop
2. Ask: `"What MCP servers are available?"`
3. Look for "dynatrace-ufo" in the response

**Expected Result**: Claude should mention the UFO server and its capabilities.

---

### 2. UFO Status Test

**Objective**: Test the getStatus resource.

**Steps**:
1. Ask: `"Show me the current UFO status"`
2. Claude should call the `ufo://status` resource

**Expected Result**: 
- Status information displayed
- Response includes UFO IP and timestamp
- No connection errors (if UFO is reachable)

---

### 3. Raw API Command Test

**Objective**: Test the sendRawApi tool with basic commands.

**Test Commands**:

```
# Test 1: Simple effect
"Send a raw command to the UFO: effect=rainbow"

# Test 2: Brightness control  
"Send a raw UFO command: dim=128"

# Test 3: Logo control
"Execute this UFO command: logo=on"

# Test 4: Complex ring pattern
"Send this to the UFO: top_init=1&top=ff0000&bottom_init=1&bottom=0000ff"
```

**Expected Results**:
- Each command should execute successfully
- Response shows the raw command and UFO response
- Events should be published (check server logs)

---

### 4. Input Validation Tests

**Objective**: Verify security and validation work correctly.

**Test Commands**:

```
# Test 1: Missing parameter
"Use the sendRawApi tool without any parameters"

# Test 2: Invalid parameter type  
"Call sendRawApi with query set to the number 123"

# Test 3: Suspicious content (should be blocked)
"Send this UFO command: <script>alert('test')</script>"
```

**Expected Results**:
- Missing parameter: Error about required 'query' parameter
- Invalid type: Error about parameter being a string
- Suspicious content: Error about unsafe characters

---

### 5. Error Handling Tests

**Objective**: Test UFO communication error handling.

**Steps**:
1. Configure server with invalid UFO IP:
   ```bash
   UFO_IP=999.999.999.999 ufo-mcp --transport stdio
   ```
2. Ask: `"Send a raw command to the UFO: effect=test"`

**Expected Result**: 
- Error message about UFO communication failure
- Specific error details (timeout, connection refused, etc.)

---

### 6. Effects Management Tests

**Objective**: Test effect storage functionality.

> ⚠️ **Note**: These tests require implementing the remaining tools first.

**Test Commands** (for future implementation):
```
# List effects
"Show me all available UFO lighting effects"

# Add new effect
"Create a new UFO effect called 'myTest' with pattern 'top_init=1&top=00ff00'"

# Play effect
"Run the 'rainbow' effect for 15 seconds"

# Delete effect
"Remove the 'myTest' effect"
```

---

### 7. Concurrent Usage Test

**Objective**: Test multiple Claude Desktop sessions.

**Steps**:
1. Open multiple Claude Desktop windows
2. Use UFO commands in different windows simultaneously
3. Check for race conditions or conflicts

**Expected Result**: All sessions should work independently without interference.

---

## Debugging Tips

### 1. Server Logs

Monitor server logs for debugging:
```bash
# Run with verbose logging
ufo-mcp --transport stdio --ufo-ip YOUR_UFO_IP 2>&1 | tee ufo-mcp.log
```

### 2. HTTP Mode Testing

For easier debugging, use HTTP mode:
```bash
# Terminal 1: Start server
ufo-mcp --transport http --port 8080 --ufo-ip YOUR_UFO_IP

# Terminal 2: Test directly
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### 3. Mock UFO Server

For testing without a physical UFO:
```bash
# Start simple HTTP server
python3 -c "
import http.server
import socketserver

class UFOHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        print(f'UFO API called: {self.path}')
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'UFO: OK')

with socketserver.TCPServer(('', 8081), UFOHandler) as httpd:
    print('Mock UFO server on :8081')
    httpd.serve_forever()
"

# In another terminal
UFO_IP=localhost:8081 ufo-mcp --transport stdio
```

### 4. Claude Desktop Config Validation

Validate your Claude Desktop configuration:
```bash
# Check config file
cat "$HOME/Library/Application Support/Claude/claude_desktop_config.json" | jq .

# Test server manually
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ufo-mcp --transport stdio --ufo-ip localhost
```

## Common Issues

### 1. "MCP server not found"
- Check Claude Desktop config file path and syntax
- Verify `ufo-mcp` binary path is correct
- Restart Claude Desktop after config changes

### 2. "UFO communication error"
- Check UFO_IP environment variable
- Verify UFO device is powered on and accessible
- Test with `ping YOUR_UFO_IP`

### 3. "Effects file error"
- Check permissions on effects file directory
- Use absolute paths in configuration
- Verify disk space availability

### 4. "Tool not available"
- Check server logs for initialization errors
- Verify all dependencies are installed
- Try rebuilding with `make clean && make build`

## Test Results Template

Use this template to document your test results:

```markdown
## Test Results - [Date]

**Environment**:
- UFO MCP Server: [version]
- Claude Desktop: [version]  
- UFO Device: [IP/model]
- OS: [operating system]

**Tests**:
- [ ] Basic MCP Connection
- [ ] UFO Status Resource
- [ ] Raw API Commands
- [ ] Input Validation
- [ ] Error Handling
- [ ] Effects Management (future)
- [ ] Concurrent Usage

**Issues Found**:
1. [Issue description]
2. [Issue description]

**Notes**:
[Additional observations]
```
### 4. LED State Resource Test (Path A)

**Objective** – verify that `getLedState` returns the shadow copy and that a `ring_update` event is emitted.

**Steps**

1. Call `setRingPattern` to paint top LED 0 red.  
2. Immediately call `getLedState`.  
3. Subscribe to `/stateEvents` and watch for `ring_update`.

**Expected**

* `getLedState` JSON shows `top[0] == "FF0000"`.  
* A `ring_update` event arrives within ~1 s.