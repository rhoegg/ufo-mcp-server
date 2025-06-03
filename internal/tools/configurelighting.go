package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// ConfigureLightingTool implements the configureLighting MCP tool
type ConfigureLightingTool struct {
	client       *device.Client
	broadcaster  *events.Broadcaster
	stateManager *state.Manager
}

// NewConfigureLightingTool creates a new configureLighting tool instance
func NewConfigureLightingTool(client *device.Client, broadcaster *events.Broadcaster, stateManager *state.Manager) *ConfigureLightingTool {
	return &ConfigureLightingTool{
		client:       client,
		broadcaster:  broadcaster,
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for configureLighting
func (t *ConfigureLightingTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "configureLighting",
		Description: "Configure the entire UFO lighting in one command - top ring, bottom ring, and logo. This is the most efficient way to set UFO lighting patterns.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"top": map[string]interface{}{
					"type":        "object",
					"description": "Top ring configuration",
					"properties": map[string]interface{}{
						"segments": map[string]interface{}{
							"type":        "array",
							"description": "Array of segment patterns in format 'position|length|color'",
							"items":       map[string]interface{}{"type": "string"},
							"examples":    []interface{}{[]string{"0|5|FF0000", "10|5|00FF00"}},
						},
						"background": map[string]interface{}{
							"type":        "string",
							"description": "Background color for unlit LEDs (6-char hex)",
							"pattern":     "^[0-9A-Fa-f]{6}$",
							"examples":    []string{"000000", "202020"},
						},
						"whirl": map[string]interface{}{
							"type":        "integer",
							"description": "Rotation speed in milliseconds (0-510, 0=no rotation)",
							"minimum":     0,
							"maximum":     510,
						},
						"counterClockwise": map[string]interface{}{
							"type":        "boolean",
							"description": "Rotate counter-clockwise if true",
							"default":     false,
						},
						"morph": map[string]interface{}{
							"type":        "object",
							"description": "Morph/fade effect configuration",
							"properties": map[string]interface{}{
								"brightnessMs": map[string]interface{}{
									"type":        "integer",
									"description": "Duration at full brightness in milliseconds",
									"minimum":     0,
									"examples":    []interface{}{1000, 2000, 500},
								},
								"fadeMs": map[string]interface{}{
									"type":        "integer",
									"description": "Fade transition duration in milliseconds",
									"minimum":     100,
									"maximum":     10000,
									"examples":    []interface{}{333, 1000, 2000},
								},
							},
							"required": []string{"brightnessMs", "fadeMs"},
						},
					},
				},
				"bottom": map[string]interface{}{
					"type":        "object",
					"description": "Bottom ring configuration (same options as top)",
					"properties": map[string]interface{}{
						"segments": map[string]interface{}{
							"type":        "array",
							"description": "Array of segment patterns",
							"items":       map[string]interface{}{"type": "string"},
						},
						"background": map[string]interface{}{
							"type":        "string",
							"description": "Background color for unlit LEDs",
							"pattern":     "^[0-9A-Fa-f]{6}$",
						},
						"whirl": map[string]interface{}{
							"type":        "integer",
							"description": "Rotation speed in milliseconds",
							"minimum":     0,
							"maximum":     510,
						},
						"counterClockwise": map[string]interface{}{
							"type":        "boolean",
							"description": "Rotate counter-clockwise",
							"default":     false,
						},
						"morph": map[string]interface{}{
							"type":        "object",
							"description": "Morph/fade effect configuration",
							"properties": map[string]interface{}{
								"brightnessMs": map[string]interface{}{
									"type":        "integer",
									"description": "Duration at full brightness in milliseconds",
									"minimum":     0,
								},
								"fadeMs": map[string]interface{}{
									"type":        "integer",
									"description": "Fade transition duration in milliseconds",
									"minimum":     100,
									"maximum":     10000,
								},
							},
							"required": []string{"brightnessMs", "fadeMs"},
						},
					},
				},
				"logo": map[string]interface{}{
					"type":        "object",
					"description": "Logo configuration",
					"properties": map[string]interface{}{
						"state": map[string]interface{}{
							"type":        "string",
							"description": "Logo state",
							"enum":        []string{"on", "off"},
						},
						"color1": map[string]interface{}{
							"type":        "string",
							"description": "First logo color (6-char hex)",
							"pattern":     "^[0-9A-Fa-f]{6}$",
						},
						"color2": map[string]interface{}{
							"type":        "string",
							"description": "Second logo color (6-char hex)",
							"pattern":     "^[0-9A-Fa-f]{6}$",
						},
					},
				},
				"brightness": map[string]interface{}{
					"type":        "integer",
					"description": "Global brightness (0-255)",
					"minimum":     0,
					"maximum":     255,
				},
				"duration": map[string]interface{}{
					"type":        "integer",
					"description": "Duration in milliseconds (optional, automatically restores previous state when expired)",
					"minimum":     0,
					"examples":    []interface{}{5000, 10000, 30000},
				},
			},
		},
	}
}

// Execute runs the configureLighting tool
func (t *ConfigureLightingTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	var queries []string
	var messages []string

	// Process brightness if provided
	if brightnessVal, hasBrightness := arguments["brightness"]; hasBrightness {
		brightness := 255
		switch v := brightnessVal.(type) {
		case float64:
			brightness = int(v)
		case int:
			brightness = v
		}
		
		if brightness < 0 || brightness > 255 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: brightness must be between 0 and 255",
					},
				},
				IsError: true,
			}, nil
		}
		
		queries = append(queries, fmt.Sprintf("dim=%d", brightness))
		messages = append(messages, fmt.Sprintf("Brightness set to %d", brightness))
		t.stateManager.UpdateBrightness(brightness)
	}

	// Process top ring
	if topConfig, hasTop := arguments["top"].(map[string]interface{}); hasTop {
		query, msg, err := t.buildRingQuery("top", topConfig)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error in top ring config: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		if query != "" {
			queries = append(queries, query)
			messages = append(messages, "Top ring: "+msg)
		}
	}

	// Process bottom ring
	if bottomConfig, hasBottom := arguments["bottom"].(map[string]interface{}); hasBottom {
		query, msg, err := t.buildRingQuery("bottom", bottomConfig)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error in bottom ring config: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		if query != "" {
			queries = append(queries, query)
			messages = append(messages, "Bottom ring: "+msg)
		}
	}

	// Process logo
	if logoConfig, hasLogo := arguments["logo"].(map[string]interface{}); hasLogo {
		query, msg, err := t.buildLogoQuery(logoConfig)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error in logo config: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		if query != "" {
			queries = append(queries, query)
			messages = append(messages, "Logo: "+msg)
		}
	}

	// If no configurations provided
	if len(queries) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "No lighting configuration provided",
				},
			},
			IsError: false,
		}, nil
	}

	// Combine all queries and send in one request
	combinedQuery := strings.Join(queries, "&")
	
	// Send to UFO
	_, err := t.client.SendRawQuery(ctx, combinedQuery)
	if err != nil {
		t.broadcaster.PublishRawExecuted(combinedQuery, fmt.Sprintf("ERROR: %v", err))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to configure lighting: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	t.broadcaster.PublishRawExecuted(combinedQuery, "OK")
	
	// Capture the complete state after update and push to stack
	completeState := t.stateManager.BuildStateQuery()
	t.stateManager.PushEffect("__config__", completeState, map[string]interface{}{
		"synthetic": true,
		"perpetual": true,
	})
	
	// Check for duration parameter
	if durationVal, hasDuration := arguments["duration"]; hasDuration {
		duration := 0
		switch v := durationVal.(type) {
		case float64:
			duration = int(v)
		case int:
			duration = v
		}
		
		// Auto-convert seconds to milliseconds for small values
		if duration > 0 && duration < 50 {
			duration = duration * 1000
		}
		
		// Start timer to restore previous state
		if duration > 0 {
			go func() {
				time.Sleep(time.Duration(duration) * time.Millisecond)
				
				// Pop the configuration and restore previous
				previousEffect := t.stateManager.PopEffect()
				if previousEffect != nil {
					t.client.SendRawQuery(context.Background(), previousEffect.Pattern)
					t.broadcaster.PublishRawExecuted(previousEffect.Pattern, "OK (restored)")
				} else {
					// Clear if no previous state
					t.client.SendRawQuery(context.Background(), "top_init=1&bottom_init=1&logo=off")
					t.broadcaster.PublishRawExecuted("top_init=1&bottom_init=1&logo=off", "OK (cleared)")
				}
			}()
			
			// Add duration info to success message
			messages = append(messages, fmt.Sprintf("Duration: %.1f seconds", float64(duration)/1000))
		}
	}

	// Build success message
	successMsg := "✨ UFO lighting configured successfully!\n\n" + strings.Join(messages, "\n")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: successMsg,
			},
		},
		IsError: false,
	}, nil
}

// buildRingQuery builds a query string for a ring configuration
func (t *ConfigureLightingTool) buildRingQuery(ring string, config map[string]interface{}) (string, string, error) {
	var queryParts []string
	var message []string

	// Always initialize the ring
	queryParts = append(queryParts, fmt.Sprintf("%s_init=1", ring))

	// Process segments
	if segmentsVal, hasSegments := config["segments"]; hasSegments {
		segments, ok := segmentsVal.([]interface{})
		if !ok {
			return "", "", fmt.Errorf("segments must be an array")
		}

		var segmentStrs []string
		for _, seg := range segments {
			segStr, ok := seg.(string)
			if !ok {
				return "", "", fmt.Errorf("segment must be a string")
			}
			if !isValidSegmentFormat(segStr) {
				return "", "", fmt.Errorf("invalid segment format: %s", segStr)
			}
			segmentStrs = append(segmentStrs, segStr)
		}

		if len(segmentStrs) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", ring, strings.Join(segmentStrs, "|")))
			message = append(message, fmt.Sprintf("%d segments", len(segmentStrs)))
		}
	}

	// Process background
	if bgVal, hasBg := config["background"]; hasBg {
		bg, ok := bgVal.(string)
		if !ok {
			return "", "", fmt.Errorf("background must be a string")
		}
		if !isValidHexColor(bg) {
			return "", "", fmt.Errorf("invalid background color: %s", bg)
		}
		queryParts = append(queryParts, fmt.Sprintf("%s_bg=%s", ring, bg))
		message = append(message, fmt.Sprintf("background #%s", bg))
	}

	// Process whirl
	if whirlVal, hasWhirl := config["whirl"]; hasWhirl {
		var whirl int
		switch v := whirlVal.(type) {
		case float64:
			whirl = int(v)
		case int:
			whirl = v
		}
		
		if whirl < 0 || whirl > 510 {
			return "", "", fmt.Errorf("whirl must be between 0 and 510")
		}

		if whirl > 0 {
			whirlStr := fmt.Sprintf("%d", whirl)
			ccw, _ := config["counterClockwise"].(bool)
			if ccw {
				whirlStr += "|ccw"
				message = append(message, fmt.Sprintf("rotating CCW at %dms", whirl))
			} else {
				message = append(message, fmt.Sprintf("rotating CW at %dms", whirl))
			}
			queryParts = append(queryParts, fmt.Sprintf("%s_whirl=%s", ring, whirlStr))
		}
	}

	// Process morph - NEW IMPLEMENTATION
	if morphVal, hasMorph := config["morph"]; hasMorph {
		morphConfig, ok := morphVal.(map[string]interface{})
		if !ok {
			return "", "", fmt.Errorf("morph must be an object with brightnessMs and fadeMs properties")
		}

		// Extract brightnessMs
		var brightnessMs int
		if bVal, ok := morphConfig["brightnessMs"]; ok {
			switch v := bVal.(type) {
			case float64:
				brightnessMs = int(v)
			case int:
				brightnessMs = v
			default:
				return "", "", fmt.Errorf("brightnessMs must be a number")
			}
		} else {
			return "", "", fmt.Errorf("morph requires brightnessMs property")
		}

		// Extract fadeMs
		var fadeMs int
		if fVal, ok := morphConfig["fadeMs"]; ok {
			switch v := fVal.(type) {
			case float64:
				fadeMs = int(v)
			case int:
				fadeMs = v
			default:
				return "", "", fmt.Errorf("fadeMs must be a number")
			}
		} else {
			return "", "", fmt.Errorf("morph requires fadeMs property")
		}

		// Validate ranges
		if brightnessMs < 0 {
			return "", "", fmt.Errorf("brightnessMs must be non-negative")
		}
		if fadeMs < 100 || fadeMs > 10000 {
			return "", "", fmt.Errorf("fadeMs must be between 100 and 10000")
		}

		// Convert to device format
		morphDevice := device.ConvertMorphToDevice(&device.MorphConfig{
			BrightnessMs: brightnessMs,
			FadeMs:       fadeMs,
		})

		queryParts = append(queryParts, fmt.Sprintf("%s_morph=%s", ring, morphDevice))
		message = append(message, fmt.Sprintf("morphing %dms bright, %dms fade", brightnessMs, fadeMs))
		
		// Update state manager with morph data
		t.stateManager.UpdateMorph(ring, &state.MorphData{
			BrightnessMs: brightnessMs,
			FadeMs: fadeMs,
		})
	}
	
	// Update state manager for segments and background
	t.updateRingState(ring, config)

	return strings.Join(queryParts, "&"), strings.Join(message, ", "), nil
}

// buildLogoQuery builds a query string for logo configuration
func (t *ConfigureLightingTool) buildLogoQuery(config map[string]interface{}) (string, string, error) {
	state, _ := config["state"].(string)
	color1, _ := config["color1"].(string)
	color2, _ := config["color2"].(string)

	var query string
	var message string

	if state == "off" {
		query = "logo=000000|000000|000000|000000"
		message = "turned off"
		t.stateManager.UpdateLogo(false)
	} else if state == "on" || color1 != "" || color2 != "" {
		if color1 != "" || color2 != "" {
			// Validate colors
			if color1 != "" && !isValidHexColor(color1) {
				return "", "", fmt.Errorf("invalid color1: %s", color1)
			}
			if color2 != "" && !isValidHexColor(color2) {
				return "", "", fmt.Errorf("invalid color2: %s", color2)
			}

			// Build pattern
			var pattern string
			if color1 != "" && color2 != "" {
				pattern = fmt.Sprintf("%s|%s|%s|%s", color1, color2, color1, color2)
				message = fmt.Sprintf("on with colors #%s and #%s", color1, color2)
			} else if color1 != "" {
				pattern = color1
				message = fmt.Sprintf("on with color #%s", color1)
			} else {
				pattern = color2
				message = fmt.Sprintf("on with color #%s", color2)
			}
			query = fmt.Sprintf("logo=%s", pattern)
		} else {
			query = "logo=on"
			message = "turned on"
		}
		t.stateManager.UpdateLogo(true)
	}

	return query, message, nil
}

// updateRingState updates the state manager with ring configuration
func (t *ConfigureLightingTool) updateRingState(ring string, config map[string]interface{}) {
	// Update whirl state
	if whirlVal, hasWhirl := config["whirl"]; hasWhirl {
		whirl := 0
		switch v := whirlVal.(type) {
		case float64:
			whirl = int(v)
		case int:
			whirl = v
		}
		t.stateManager.UpdateWhirl(ring, whirl)
	}
	
	// Update segments state
	var ledColors []string
	background := ""
	
	if bgVal, hasBg := config["background"]; hasBg {
		background, _ = bgVal.(string)
	}
	
	if segmentsVal, hasSegments := config["segments"]; hasSegments {
		// Build LED color array from segments
		colors := make([]string, 15)
		// Initialize with background or black
		for i := 0; i < 15; i++ {
			if background != "" {
				colors[i] = background
			} else {
				colors[i] = "000000"
			}
		}
		
		// Apply segments
		if segments, ok := segmentsVal.([]interface{}); ok {
			for _, seg := range segments {
				if segStr, ok := seg.(string); ok {
					parts := strings.Split(segStr, "|")
					if len(parts) == 3 {
						start, _ := strconv.Atoi(parts[0])
						count, _ := strconv.Atoi(parts[1])
						color := parts[2]
						for i := 0; i < count && start+i < 15; i++ {
							colors[start+i] = color
						}
					}
				}
			}
		}
		ledColors = colors
	} else if background != "" {
		// Just background, no segments
		colors := make([]string, 15)
		for i := 0; i < 15; i++ {
			colors[i] = background
		}
		ledColors = colors
	}
	
	if len(ledColors) > 0 {
		t.stateManager.UpdateRingSegments(ring, ledColors, background)
	}
}