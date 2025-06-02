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

// ConvertDurationToMs converts seconds to milliseconds
func ConvertDurationToMs(seconds int) int {
	return seconds * 1000
}

// ConvertDurationFromMs converts milliseconds to seconds
func ConvertDurationFromMs(milliseconds int) int {
	return milliseconds / 1000
}