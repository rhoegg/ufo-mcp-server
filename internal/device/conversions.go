package device

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MorphConfig represents morph settings in milliseconds
type MorphConfig struct {
	BrightnessMs int `json:"brightnessMs"`
	FadeMs       int `json:"fadeMs"`
}

// ConvertMorphToDevice converts millisecond-based morph config to device format
func ConvertMorphToDevice(config *MorphConfig) string {
	if config == nil {
		return ""
	}

	// Convert brightness milliseconds to ticks
	// Based on empirical data: ~150 ticks per second
	ticks := int(math.Round(float64(config.BrightnessMs) / 6.67))

	// Convert fade milliseconds to speed (1-10)
	// Based on empirical data: fade_frames = 200/speed at 60fps
	// 200 frames = 3333ms, so fade_ms = 3333/speed
	speed := int(math.Round(3333.0 / float64(config.FadeMs)))
	if speed < 1 {
		speed = 1
	} else if speed > 10 {
		speed = 10
	}

	return fmt.Sprintf("%d|%d", ticks, speed)
}

// ConvertMorphFromDevice converts device format to milliseconds
func ConvertMorphFromDevice(morphSpec string) *MorphConfig {
	if morphSpec == "" {
		return nil
	}

	parts := strings.Split(morphSpec, "|")
	if len(parts) != 2 {
		return nil
	}

	ticks, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}

	speed, err := strconv.Atoi(parts[1])
	if err != nil || speed < 1 || speed > 10 {
		return nil
	}

	brightnessMs := int(float64(ticks) * 6.67)
	fadeMs := int(3333.0 / float64(speed))

	return &MorphConfig{
		BrightnessMs: brightnessMs,
		FadeMs:       fadeMs,
	}
}

// ConvertWhirlToDevice converts rotation period in milliseconds to device whirl value
// The whirl parameter controls rotation speed using a tick-based countdown mechanism:
// - whirlTick starts at (0xFF - whirlSpeed) 
// - Decrements each millisecond (Display() called with 1ms delay)
// - When reaches 0, advances LED position and resets to (0xFF - whirlSpeed)
// - Therefore: rotation_step_delay_ms = 256 - whirlSpeed
// - And: full_rotation_ms = (256 - whirlSpeed) * 15 LEDs
func ConvertWhirlToDevice(rotationMs int) int {
	if rotationMs <= 0 {
		return 0 // No rotation
	}
	
	// Calculate step delay needed for desired rotation period
	// rotationMs = stepDelayMs * 15 LEDs
	stepDelayMs := float64(rotationMs) / 15.0
	
	// whirlSpeed = 256 - stepDelayMs
	whirlSpeed := 256.0 - stepDelayMs
	
	// Clamp to valid range (1-255)
	// Note: Documentation mentions max 510, but actual range depends on 
	// whether the firmware extends beyond uint8. Testing shows values
	// up to ~490 work, suggesting extended range.
	if whirlSpeed < 1 {
		whirlSpeed = 1
	}
	if whirlSpeed > 510 {
		whirlSpeed = 510
	}
	
	return int(math.Round(whirlSpeed))
}

// ConvertDeviceWhirlToMs converts device whirl value to rotation period in milliseconds
func ConvertDeviceWhirlToMs(whirlSpeed int) int {
	if whirlSpeed <= 0 {
		return 0 // No rotation
	}
	
	// Calculate step delay: stepDelayMs = 256 - whirlSpeed
	// For values > 255, this gives negative delay (very fast rotation)
	stepDelayMs := 256 - whirlSpeed
	if stepDelayMs < 1 {
		stepDelayMs = 1 // Minimum 1ms per step
	}
	
	// Calculate full rotation time: rotationMs = stepDelayMs * 15 LEDs
	rotationMs := stepDelayMs * 15
	
	return rotationMs
}

// ConvertDurationToMs converts seconds to milliseconds
func ConvertDurationToMs(seconds int) int {
	return seconds * 1000
}

// ConvertDurationFromMs converts milliseconds to seconds
func ConvertDurationFromMs(milliseconds int) int {
	return milliseconds / 1000
}