# Release v1.0.0 - Production Ready MCP Server

## 🎉 First Stable Release

This is the first production-ready release of the UFO MCP server, featuring a complete and intuitive interface for controlling Dynatrace UFO devices.

### Key Features

1. **Millisecond-Based Time Interface**
   - All time values use milliseconds for consistency and precision
   - Morph effects use intuitive object format: `{"brightnessMs": 1000, "fadeMs": 333}`
   - Direct control over timing without abstract units

2. **Complete Tool Set**
   - `configureLighting` - One-command UFO configuration
   - `setRingPattern` - Advanced ring control
   - `playEffect` / `stopEffects` - Effect management with stacking
   - `getLedState` - Shadow state query
   - Full CRUD operations for custom effects

3. **Shadow State Tracking**
   - Complete LED state maintained in memory
   - Real-time events via SSE
   - Animation parameter tracking

4. **Effects System**
   - JSON-based effect definitions
   - Auto-migration from development versions
   - Perpetual and timed effects
   - Effect stacking with push/pop behavior

### Technical Highlights

- **MCP 2025-03-26 Specification** compliant
- **HTTP/2** support with h2c
- **Docker** ready (< 20MB image)
- **Race-free** concurrent operations
- **60fps tested** animation timing

### Morph Parameter Details

Based on extensive frame-by-frame analysis:
- Brightness duration: `ticks = milliseconds / 6.67`
- Fade speed: `speed = 3333 / fade_milliseconds`
- Symmetric fade transitions at all speeds

### Example Usage

```json
{
  "tool": "configureLighting",
  "arguments": {
    "top": {
      "segments": ["0|8|FF0000", "8|7|444444"],
      "background": "000000",
      "morph": {
        "brightnessMs": 1000,
        "fadeMs": 333
      },
      "whirl": 300
    },
    "logo": {
      "state": "on",
      "color1": "FF0000",
      "color2": "444444"
    },
    "brightness": 150
  }
}
```

### Installation

```bash
# Using the install script
curl -fsSL https://raw.githubusercontent.com/starspace46/ufo/main/install.sh | bash

# Or download directly
VERSION=v1.0.0
OS=darwin  # or linux
ARCH=arm64 # or amd64
curl -L https://github.com/starspace46/ufo/releases/download/${VERSION}/ufo-mcp-${OS}-${ARCH} -o ufo-mcp
chmod +x ufo-mcp
```

### Documentation

- [UFO API Reference](UFO_API_REFERENCE.md) - Complete UFO device API documentation
- [Testing Guide](TESTING.md) - How to test the MCP server
- [MCP UFO Plan](MCP_UFO_PLAN.md) - Original design specification

### What's Next

Future releases will focus on:
- Additional effect presets
- Enhanced event streaming
- Performance optimizations
- Extended device support