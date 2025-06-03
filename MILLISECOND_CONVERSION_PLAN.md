# Millisecond Conversion Plan for UFO MCP Server

## Overview
This document outlines the implementation plan for converting UFO device units to milliseconds in the MCP server, providing a more intuitive interface for users and AI agents.

## 1. Morph Parameter Conversion

### Current State
- **Device format**: `PERIOD|SPEED`
- **PERIOD**: Device ticks (~150 ticks/second)
- **SPEED**: 1-10 (inversely affects fade duration)

### Proposed MCP Interface
```json
{
  "morph": {
    "brightnessMs": 1000,    // Time at full brightness in ms
    "fadeMs": 500            // Fade transition time in ms
  }
}
```

### Implementation in MCP Server

#### A. Update Tool Parameters
```go
// In configureLighting tool
"morph": {
  "type": "object",
  "properties": {
    "brightnessMs": {
      "type": "number",
      "description": "Duration at full brightness in milliseconds"
    },
    "fadeMs": {
      "type": "number", 
      "description": "Fade transition duration in milliseconds"
    }
  }
}
```

#### B. Conversion Functions
```go
// internal/device/conversions.go
package device

import "math"

// ConvertMorphToDevice converts millisecond-based morph to device format
func ConvertMorphToDevice(brightnessMs, fadeMs float64) string {
    // Convert brightness milliseconds to ticks
    ticks := int(brightnessMs / 6.67)
    
    // Convert fade milliseconds to speed (1-10)
    // Based on empirical data: fade_ms = 3333/speed
    speed := int(math.Round(3333.0 / fadeMs))
    if speed < 1 {
        speed = 1
    } else if speed > 10 {
        speed = 10
    }
    
    return fmt.Sprintf("%d|%d", ticks, speed)
}

// ConvertMorphFromDevice converts device format to milliseconds
func ConvertMorphFromDevice(morphSpec string) (brightnessMs, fadeMs float64) {
    parts := strings.Split(morphSpec, "|")
    if len(parts) != 2 {
        return 0, 0
    }
    
    ticks, _ := strconv.Atoi(parts[0])
    speed, _ := strconv.Atoi(parts[1])
    
    brightnessMs = float64(ticks) * 6.67
    fadeMs = 3333.0 / float64(speed)
    
    return brightnessMs, fadeMs
}
```

#### C. Update Shadow State
```go
// internal/state/state.go
type MorphState struct {
    BrightnessMs float64 `json:"brightnessMs"`
    FadeMs       float64 `json:"fadeMs"`
    // Keep original for accuracy
    DeviceString string  `json:"deviceString,omitempty"`
}
```

## 2. All Effects Millisecond Standardization

### Affected Parameters

#### A. Whirl (Rotation)
**Current**: Device units (0-510)
**COMPLETED**: Now accepts rotation period in milliseconds
- Whirl parameter now represents full rotation time in milliseconds
- Conversion formula: `whirlDevice = 256 - (rotationMs / 15)`
- Maximum rotation period: 7650ms (~7.65 seconds)

#### B. Effect Duration
**Current**: Seconds in tool interface
**Change**: Convert to milliseconds for consistency

#### C. Morph
**Current**: Device format string "PERIOD|SPEED"
**Change**: Object with millisecond values

### Direct Breaking Change Implementation

#### Single-Phase Approach
1. Update all tools to accept milliseconds
2. Convert internally to device format
3. Update shadow state to store milliseconds
4. Bump to version 2.0.0

### New Interface Structure

```go
// configureLighting tool
type RingConfig struct {
    Segments         []string     `json:"segments,omitempty"`
    Background       string       `json:"background,omitempty"`
    Whirl            int          `json:"whirl,omitempty"`           // Keep name, already in ms
    CounterClockwise bool         `json:"counterClockwise,omitempty"`
    Morph            *MorphConfig `json:"morph,omitempty"`           // Changed from string to object
}

type MorphConfig struct {
    BrightnessMs int `json:"brightnessMs"` // Time at full brightness
    FadeMs       int `json:"fadeMs"`       // Fade transition time
}

// playEffect tool  
type PlayEffectArgs struct {
    Name     string `json:"name"`
    Duration int    `json:"duration,omitempty"` // Keep name, now in milliseconds
}

// Effect storage
type Effect struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Pattern     string `json:"pattern"`
    Duration    int    `json:"duration,omitempty"` // Keep name, now in milliseconds
}
```

## 3. Benefits

1. **Intuitive for humans**: "1000ms bright, 500ms fade" vs "150|7"
2. **AI-friendly**: LLMs understand milliseconds better than device ticks
3. **Consistent units**: All time values in milliseconds
4. **Precision**: Exact timing instead of speed approximations
5. **Future-proof**: Easy to adapt if device firmware changes

## 4. Implementation Steps

1. **Update conversion functions** (internal/device/conversions.go)
   - Create morph conversion utilities
   - Test with known values from our measurements

2. **Update all tool interfaces** 
   - configureLighting: Change morph from string to object
   - setRingPattern: Same change
   - playEffect/addEffect/updateEffect: duration now in ms instead of seconds
   - Update all tool descriptions to clarify millisecond units
   
3. **Update shadow state** (internal/state/state.go)
   - Store all time values in milliseconds
   - Update getLedState to return millisecond values

4. **Update effects storage** (internal/effects/effect.go)
   - Change Duration field to DurationMs
   - Add migration for existing effects.json files

5. **Update tests and documentation**
   - Fix all broken tests
   - Update CLAUDE.md with new interfaces
   - Update tool descriptions

6. **Release as v2.0.0**
   - Clear breaking change notes
   - Migration guide for users

## 5. Migration Examples

### Before (v1.x)
```json
{
  "tool": "configureLighting",
  "arguments": {
    "top": {
      "morph": "150|10",
      "whirl": 300
    }
  }
}

{
  "tool": "playEffect",
  "arguments": {
    "name": "rainbow",
    "duration": 30  // seconds
  }
}
```

### After (v2.0.0)
```json
{
  "tool": "configureLighting", 
  "arguments": {
    "top": {
      "morph": {
        "brightnessMs": 1000,
        "fadeMs": 333
      },
      "whirl": 300  // still milliseconds, no rename
    }
  }
}

{
  "tool": "playEffect",
  "arguments": {
    "name": "rainbow",
    "duration": 30000  // now milliseconds
  }
}
```