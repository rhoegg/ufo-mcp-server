package device

import (
	"testing"
)

func TestConvertMorphToDevice(t *testing.T) {
	tests := []struct {
		name     string
		config   *MorphConfig
		expected string
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: "",
		},
		{
			name: "1 second brightness, fast fade",
			config: &MorphConfig{
				BrightnessMs: 1000,
				FadeMs:       333,
			},
			expected: "150|10",
		},
		{
			name: "1 second brightness, slow fade",
			config: &MorphConfig{
				BrightnessMs: 1000,
				FadeMs:       3333,
			},
			expected: "150|1",
		},
		{
			name: "6.67 seconds brightness, medium fade",
			config: &MorphConfig{
				BrightnessMs: 6670,
				FadeMs:       2000,
			},
			expected: "1000|2",
		},
		{
			name: "very fast fade clamped to 10",
			config: &MorphConfig{
				BrightnessMs: 1000,
				FadeMs:       100, // Would be speed 33, clamped to 10
			},
			expected: "150|10",
		},
		{
			name: "very slow fade clamped to 1",
			config: &MorphConfig{
				BrightnessMs: 1000,
				FadeMs:       10000, // Would be speed 0.3, clamped to 1
			},
			expected: "150|1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertMorphToDevice(tt.config)
			if result != tt.expected {
				t.Errorf("ConvertMorphToDevice() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConvertMorphFromDevice(t *testing.T) {
	tests := []struct {
		name      string
		morphSpec string
		expected  *MorphConfig
	}{
		{
			name:      "empty string",
			morphSpec: "",
			expected:  nil,
		},
		{
			name:      "invalid format",
			morphSpec: "150",
			expected:  nil,
		},
		{
			name:      "150|10 - 1s bright, fast fade",
			morphSpec: "150|10",
			expected: &MorphConfig{
				BrightnessMs: 1000, // 150 * 6.67
				FadeMs:       333,  // 3333 / 10
			},
		},
		{
			name:      "150|1 - 1s bright, slow fade",
			morphSpec: "150|1",
			expected: &MorphConfig{
				BrightnessMs: 1000, // 150 * 6.67
				FadeMs:       3333, // 3333 / 1
			},
		},
		{
			name:      "1000|5 - 6.67s bright, medium fade",
			morphSpec: "1000|5",
			expected: &MorphConfig{
				BrightnessMs: 6670, // 1000 * 6.67
				FadeMs:       666,  // 3333 / 5
			},
		},
		{
			name:      "invalid speed",
			morphSpec: "150|0",
			expected:  nil,
		},
		{
			name:      "invalid speed too high",
			morphSpec: "150|11",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertMorphFromDevice(tt.morphSpec)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ConvertMorphFromDevice() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("ConvertMorphFromDevice() = nil, want %v", tt.expected)
				} else if result.BrightnessMs != tt.expected.BrightnessMs || result.FadeMs != tt.expected.FadeMs {
					t.Errorf("ConvertMorphFromDevice() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that converting to device format and back preserves approximate values
	configs := []*MorphConfig{
		{BrightnessMs: 1000, FadeMs: 333},
		{BrightnessMs: 2000, FadeMs: 1000},
		{BrightnessMs: 500, FadeMs: 2000},
	}

	for _, original := range configs {
		deviceFormat := ConvertMorphToDevice(original)
		converted := ConvertMorphFromDevice(deviceFormat)

		// Allow for rounding errors
		brightnessDiff := abs(original.BrightnessMs - converted.BrightnessMs)
		fadeDiff := abs(original.FadeMs - converted.FadeMs)

		if brightnessDiff > 50 { // Allow 50ms tolerance
			t.Errorf("Brightness round trip failed: original=%d, converted=%d", original.BrightnessMs, converted.BrightnessMs)
		}
		if fadeDiff > 400 { // Allow 400ms tolerance for fade due to integer division rounding
			t.Errorf("Fade round trip failed: original=%d, converted=%d", original.FadeMs, converted.FadeMs)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestConvertWhirlToDevice(t *testing.T) {
	tests := []struct {
		name       string
		rotationMs int
		want       int
	}{
		{
			name:       "no rotation",
			rotationMs: 0,
			want:       0,
		},
		{
			name:       "1 second rotation",
			rotationMs: 1000,
			want:       189, // 256 - (1000/15) = 256 - 66.67 ≈ 189
		},
		{
			name:       "2 second rotation",
			rotationMs: 2000,
			want:       123, // 256 - (2000/15) = 256 - 133.33 ≈ 123
		},
		{
			name:       "3 second rotation",
			rotationMs: 3000,
			want:       56, // 256 - (3000/15) = 256 - 200 = 56
		},
		{
			name:       "500ms rotation (fast)",
			rotationMs: 500,
			want:       223, // 256 - (500/15) = 256 - 33.33 ≈ 223
		},
		{
			name:       "minimum valid (very fast)",
			rotationMs: 15, // One step per LED
			want:       255, // 256 - 1 = 255
		},
		{
			name:       "maximum valid (slow)",
			rotationMs: 7650, // 510ms per step × 15 LEDs
			want:       1, // Minimum speed
		},
		{
			name:       "beyond maximum",
			rotationMs: 10000,
			want:       1, // Clamped to minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertWhirlToDevice(tt.rotationMs)
			if got != tt.want {
				t.Errorf("ConvertWhirlToDevice(%d) = %d, want %d", tt.rotationMs, got, tt.want)
			}
		})
	}
}

func TestConvertDeviceWhirlToMs(t *testing.T) {
	tests := []struct {
		name       string
		whirlSpeed int
		want       int
	}{
		{
			name:       "no rotation",
			whirlSpeed: 0,
			want:       0,
		},
		{
			name:       "speed 189 (≈1 second)",
			whirlSpeed: 189,
			want:       1005, // (256-189) × 15 = 67 × 15 = 1005
		},
		{
			name:       "speed 123 (≈2 seconds)",
			whirlSpeed: 123,
			want:       1995, // (256-123) × 15 = 133 × 15 = 1995
		},
		{
			name:       "speed 56 (3 seconds)",
			whirlSpeed: 56,
			want:       3000, // (256-56) × 15 = 200 × 15 = 3000
		},
		{
			name:       "maximum speed 255",
			whirlSpeed: 255,
			want:       15, // (256-255) × 15 = 1 × 15 = 15
		},
		{
			name:       "beyond normal range",
			whirlSpeed: 300,
			want:       15, // Negative delay clamped to 1ms × 15 = 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertDeviceWhirlToMs(tt.whirlSpeed)
			if got != tt.want {
				t.Errorf("ConvertDeviceWhirlToMs(%d) = %d, want %d", tt.whirlSpeed, got, tt.want)
			}
		})
	}
}