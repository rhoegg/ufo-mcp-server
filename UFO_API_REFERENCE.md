# Dynatrace UFO API Reference

This document provides a comprehensive reference for the Dynatrace UFO (2nd generation ESP32) REST API, including all parameters, effects, and implementation details discovered through testing and source code analysis.

## Overview

The UFO device exposes a REST API on port 80 at the `/api` endpoint. All control is done through GET requests with query parameters.

**Base URL**: `http://{ufo-ip}/api`

## LED Layout

- **Top Ring**: 15 LEDs (indices 0-14)
- **Bottom Ring**: 15 LEDs (indices 0-14)
- **Logo**: 1 LED (on/off only)
- **Total**: 31 LEDs

## Query Parameters

### Basic LED Control

#### `top` / `bottom`
Controls LED segments on the respective ring.

**Format**: `LED_INDEX|COUNT|RRGGBB`
- `LED_INDEX`: Starting LED position (0-14)
- `COUNT`: Number of LEDs to set
- `RRGGBB`: Hex color code

**Multiple segments**: Chain with `|`
- Example: `top=0|5|FF0000|8|7|00FF00` (5 red LEDs starting at 0, 7 green LEDs starting at 8)

#### `top_init` / `bottom_init`
Clears the ring before applying new colors.

**Values**: `0` or `1`
- `1`: Clear the ring first
- `0`: Apply colors on top of existing

**Best practice**: Always use `init=1` when setting a new pattern.

#### `top_bg` / `bottom_bg`
Sets the background color for unlit LEDs.

**Format**: `RRGGBB` (hex color)
- Default: `000000` (black/off)
- Example: `top_bg=000033` (dim blue background)

### Logo Control

#### `logo`
Controls the Dynatrace logo LED.

**Values**: `on` or `off`
- Note: ESP32 version only supports on/off, not color control

### Brightness

#### `dim`
Global brightness control for all LEDs.

**Range**: `0-255`
- `0`: LEDs off
- `255`: Maximum brightness
- Recommended: `60-150` for typical use

### Animation Effects

#### `top_whirl` / `bottom_whirl`
Rotates the LED pattern around the ring.

**Format**: `SPEED` or `SPEED|ccw`
- `SPEED`: Rotation speed in milliseconds (0-510)
  - Lower values = faster rotation
  - `0`: No rotation
- `|ccw`: Add for counter-clockwise rotation
- Example: `top_whirl=300|ccw` (counter-clockwise at 300ms speed)

#### `top_morph` / `bottom_morph`
Creates a pulsing/fading effect between foreground and background colors.

**Format**: `PERIOD|SPEED`
- `PERIOD`: Duration of full-brightness phase in device ticks
  - **Tick rate**: ~150 ticks per second
  - **Conversion**: milliseconds = ticks × 6.67
  - Example: 150 ticks = ~1000ms (1 second)
- `SPEED`: Fade transition speed (1-10)
  - Controls fade duration: higher = faster
  - See fade timing table below

**Fade Transition Timing** (measured at 60fps):
| Speed | Fade Duration | Frames | Milliseconds |
|-------|---------------|--------|--------------|
| 1     | Slowest       | 200    | 3333ms       |
| 5     | Medium        | 120    | 2000ms       |
| 10    | Fastest       | 20     | 333ms        |

**Morph Cycle Behavior**:
1. **Full brightness**: Display at full color for PERIOD ticks
2. **Fade out**: Transition to background color (duration depends on SPEED)
3. **Fade in**: Transition back to full color (same duration as fade out)

**Example Timings**:
- `150|10`: 1s bright, 0.33s fade out, 0.33s fade in (1.67s total)
- `150|1`: 1s bright, 3.33s fade out, 3.33s fade in (7.67s total)
- `1000|1`: 6.67s bright, 3.33s fade out, 3.33s fade in (13.33s total)

## Common Patterns and Examples

### Static Ring Pattern
```
/api?top_init=1&top=0|15|FF0000&logo=on&dim=100
```
Sets entire top ring to red at ~40% brightness with logo on.

### Split Color Ring
```
/api?top_init=1&top=0|8|FF0000|8|7|00FF00&top_bg=000000
```
Top ring: 8 red LEDs, 7 green LEDs, black background.

### Rotating Rainbow
```
/api?top_init=1&top=0|5|FF0000|5|5|00FF00|10|5|0000FF&top_whirl=200
```
Three color segments rotating clockwise every 200ms.

### Pulsing Effect
```
/api?top_init=1&top=0|15|FF00FF&top_bg=000000&top_morph=100|10
```
Purple ring pulsing between full color and black.

### Counter-Rotating Rings
```
/api?top_init=1&top=0|8|FF0000&top_whirl=300&bottom_init=1&bottom=0|8|0000FF&bottom_whirl=300|ccw
```
Red segment on top rotating clockwise, blue segment on bottom rotating counter-clockwise.

## Implementation Notes

### URL Encoding
- The UFO accepts raw pipe characters (`|`) in query parameters
- Avoid URL encoding pipes - send them as-is
- Use `&` to separate parameters normally

### State Management
- The UFO device does not provide a way to query current LED state
- The MCP server maintains a "shadow state" to track current configuration
- Always use `init=1` when setting new patterns to ensure consistency

### Performance
- Multiple parameters can be combined in a single request
- The ESP32 processes commands quickly with minimal latency
- Animations (whirl/morph) run on the device without additional commands

### Limitations
- No built-in effects library (unlike some LED controllers)
- Logo is single color (on/off) in ESP32 version
- Cannot query device for current state
- Maximum 15 LEDs per ring

## Tips for MCP Server Implementation

1. **Always track state**: Since the UFO doesn't report state, maintain shadow state for `getLedState` tool
2. **Use init flags**: Always set `top_init=1` or `bottom_init=1` when changing patterns
3. **Validate indices**: LED indices must be 0-14, ensure COUNT doesn't exceed ring boundary
4. **Handle brightness**: Convert percentage (0-100) to device range (0-255)
5. **Morph timing**: Remember morph creates asymmetric on/off cycles
6. **Whirl direction**: Parse and handle the `|ccw` suffix for counter-clockwise rotation

## Error Handling

The UFO device typically returns:
- `200 OK` with body `OK` on success
- No detailed error messages
- Silently ignores invalid parameters

The MCP server should validate all inputs before sending to the device.