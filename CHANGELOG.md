# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-06-02

### Added
- **Millisecond-based interface**: All time values use milliseconds for consistency
  - `morph` parameter uses object format `{"brightnessMs": 1000, "fadeMs": 333}` for intuitive control
  - Effect `duration` in milliseconds for precision
  - `whirl` parameter in milliseconds
- Enhanced shadow state to track animation parameters (whirl, morph)
- Morph conversion utilities (`ConvertMorphToDevice`, `ConvertMorphFromDevice`)
- Auto-migration for effects files from prototype versions
- Comprehensive UFO API reference documentation
- Frame-by-frame morph timing analysis

### Changed
- Improved morph documentation based on empirical testing
- More intuitive morph control with explicit millisecond values

### Technical Details
- Conversion formulas: `ticks = ms / 6.67`, `speed = 3333 / fade_ms`
- Based on 60fps frame analysis of actual UFO behavior
- Fade transitions are symmetric at all speed settings (1-10)

## [0.4.1] - 2025-05-31

### Added
- `configureLighting` tool for unified UFO control in a single command
- Perpetual effects support - effects can now run indefinitely
- `perpetual` field in effects.json to distinguish between timed and continuous effects
- Default effects.json shipped with installation
- **Effect Stack System** - effects can now be layered with push/pop behavior
- `stopEffect` tool to pop current effect and resume previous one
- New event types: `effect_resumed` for when previous effects are restored
- Stack depth tracking in effect events

### Changed
- **BREAKING**: Effects system refactored - effects are now pure data (JSON) not code
- Logo control now uses pipe-delimited format (e.g., `ff0000|ffffff|ff0000|ffffff`)
- Fixed counter-clockwise rotation to use `|ccw` suffix
- Morph parameters now use correct format: `duration|speed` where speed is 1-10
- Perpetual effects have `duration: 0` and `perpetual: true`
- Effects no longer auto-terminate unless marked as non-perpetual
- `playEffect` now pushes effects onto a stack instead of replacing
- Timed effects automatically pop from stack and resume previous effect

### Fixed
- Logo turns off properly when colors have been set
- `breathingGreen` effect now actually breathes (added background color)
- All effects updated to use valid UFO API parameters
- Fixed effects loading from file system instead of hardcoded values
- Improved responsiveness by reducing separate API calls

### Removed
- Removed hardcoded seed effects from Go code
- Removed `ipDisplay` effect (non-existent `effect=ip` parameter)
- Removed exposed CRUD tools from MCP interface (still available internally)

## [0.4.0] - 2025-05-31

### Added
- Full support for MCP 2025-03-26 specification
- Streamable HTTP transport with single `/mcp` endpoint
- HTTP/2 support with h2c (HTTP/2 cleartext)
- Session management with 30-minute idle timeout
- JSON-RPC batch request support
- Health check endpoint at `/healthz` with build info
- Proper streaming with HTTP chunked transfer encoding
- Version information in binaries via ldflags

### Changed
- **BREAKING**: HTTP transport now uses single `/mcp` endpoint instead of separate `/sse` and `/messages`
- **BREAKING**: Removed legacy SSE/messages endpoints completely
- Upgraded to use custom streamable HTTP implementation for spec compliance
- Stdio transport remains unchanged for backward compatibility with Claude Desktop

### Fixed
- Proper Content-Type headers for all responses
- Session ID handling in Mcp-Session-Id header
- Graceful shutdown for HTTP server

### Technical Details
- Uses `golang.org/x/net/http2` for HTTP/2 support
- In-memory session management with automatic cleanup
- Supports both single and batch JSON-RPC requests
- Returns 200 OK for all responses per spec

## [0.3.0] - Previous version
- Initial MCP implementation with basic tools
- UFO device control via HTTP API
- Effects management system
- Shadow LED state tracking