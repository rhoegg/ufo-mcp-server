package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// SetRingPatternTool implements the setRingPattern MCP tool
type SetRingPatternTool struct {
	client       *device.Client
	broadcaster  *events.Broadcaster
	stateManager *state.Manager
}

// NewSetRingPatternTool creates a new setRingPattern tool instance
func NewSetRingPatternTool(client *device.Client, broadcaster *events.Broadcaster, stateManager *state.Manager) *SetRingPatternTool {
	return &SetRingPatternTool{
		client:       client,
		broadcaster:  broadcaster,
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for setRingPattern
func (t *SetRingPatternTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "setRingPattern",
		Description: "Set LED patterns on the UFO rings with advanced control over segments, colors, background, rotation, and morphing effects. This is a high-level wrapper around the UFO's ring control API.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"ring": map[string]interface{}{
					"type":        "string",
					"description": "Which ring to control: 'top' or 'bottom'",
					"enum":        []string{"top", "bottom"},
					"examples":    []string{"top", "bottom"},
				},
				"segments": map[string]interface{}{
					"type":        "array",
					"description": "Array of LED segments in format 'LED_INDEX|COUNT|RRGGBB' (e.g., ['0|5|FF0000', '10|3|00FF00'])",
					"items": map[string]interface{}{
						"type":        "string",
						"description": "Segment format: LED_INDEX|COUNT|RRGGBB",
						"pattern":     "^\\d+\\|\\d+\\|[0-9A-Fa-f]{6}$",
					},
					"examples": [][]string{
						{"0|5|FF0000", "10|5|00FF00"},
						{"0|15|FF0000"},
						{"0|3|FF0000", "5|3|00FF00", "10|3|0000FF"},
					},
				},
				"background": map[string]interface{}{
					"type":        "string",
					"description": "Background color for unlit LEDs (hex format RRGGBB, optional)",
					"pattern":     "^[0-9A-Fa-f]{6}$",
					"examples":    []string{"000000", "202020", "FFFFFF"},
				},
				"whirlMs": map[string]interface{}{
					"type":        "integer",
					"description": "Rotation speed in milliseconds (0-510, optional). Lower values = faster rotation",
					"minimum":     0,
					"maximum":     510,
					"examples":    []int{100, 200, 300, 500},
				},
				"counterClockwise": map[string]interface{}{
					"type":        "boolean",
					"description": "Set to true for counter-clockwise rotation (optional, default is false for clockwise)",
					"default":     false,
					"examples":    []bool{true, false},
				},
				"morph": map[string]interface{}{
					"type":        "string",
					"description": "Fade effect specification in format 'STAY|SPEED' in milliseconds (optional)",
					"pattern":     "^\\d+\\|\\d+$",
					"examples":    []string{"1000|500", "2000|200", "500|100"},
				},
			},
			Required: []string{"ring"},
		},
	}
}

// Execute runs the setRingPattern tool
func (t *SetRingPatternTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract ring parameter
	ringArg, exists := arguments["ring"]
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'ring' parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	ring, ok := ringArg.(string)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'ring' parameter must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Validate ring value
	if ring != "top" && ring != "bottom" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'ring' must be either 'top' or 'bottom'",
				},
			},
			IsError: true,
		}, nil
	}

	// Extract optional segments
	var segments []string
	if segmentsArg, exists := arguments["segments"]; exists {
		if segmentsArray, ok := segmentsArg.([]interface{}); ok {
			for i, segment := range segmentsArray {
				if segmentStr, ok := segment.(string); ok {
					// Validate segment format
					if !isValidSegmentFormat(segmentStr) {
						return &mcp.CallToolResult{
							Content: []mcp.Content{
								mcp.TextContent{
									Type: "text",
									Text: fmt.Sprintf("Error: invalid segment format at index %d. Expected format: 'LED_INDEX|COUNT|RRGGBB'", i),
								},
							},
							IsError: true,
						}, nil
					}
					segments = append(segments, segmentStr)
				} else {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: fmt.Sprintf("Error: segment at index %d must be a string", i),
							},
						},
						IsError: true,
					}, nil
				}
			}
		} else {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'segments' parameter must be an array",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Extract optional background
	var background string
	if bgArg, exists := arguments["background"]; exists {
		if bgStr, ok := bgArg.(string); ok {
			if !isValidHexColor(bgStr) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'background' must be a valid hex color (RRGGBB format)",
						},
					},
					IsError: true,
				}, nil
			}
			background = bgStr
		} else {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'background' parameter must be a string",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Extract optional whirl speed
	var whirlMs int
	if whirlArg, exists := arguments["whirlMs"]; exists {
		switch v := whirlArg.(type) {
		case int:
			whirlMs = v
		case float64:
			whirlMs = int(v)
		default:
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'whirlMs' parameter must be a number",
					},
				},
				IsError: true,
			}, nil
		}

		if whirlMs < 0 || whirlMs > 510 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'whirlMs' must be between 0 and 510",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Extract optional counter-clockwise flag
	var counterClockwise bool
	if ccwArg, exists := arguments["counterClockwise"]; exists {
		switch v := ccwArg.(type) {
		case bool:
			counterClockwise = v
		default:
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'counterClockwise' parameter must be a boolean",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Extract optional morph spec
	var morphSpec string
	if morphArg, exists := arguments["morph"]; exists {
		if morphStr, ok := morphArg.(string); ok {
			if !isValidMorphSpec(morphStr) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'morphSpec' must be in format 'STAY|SPEED' (e.g., '1000|500')",
						},
					},
					IsError: true,
				}, nil
			}
			morphSpec = morphStr
		} else {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'morphSpec' parameter must be a string",
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Execute the ring pattern command
	err := t.client.SetRingPattern(ctx, ring, segments, background, whirlMs, counterClockwise, morphSpec)
	if err != nil {
		// Publish the failed execution event
		command := buildRingPatternCommand(ring, segments, background, whirlMs, counterClockwise, morphSpec)
		t.broadcaster.PublishRawExecuted(command, fmt.Sprintf("ERROR: %v", err))
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to set ring pattern: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Update shadow state
	// Parse segments and update LED state
	ledColors := parseLedColors(segments, background)
	t.stateManager.UpdateRingSegments(ring, ledColors, background)
	
	// Publish the successful execution event
	command := buildRingPatternCommand(ring, segments, background, whirlMs, counterClockwise, morphSpec)
	t.broadcaster.PublishRawExecuted(command, "OK")

	// Build success message
	message := fmt.Sprintf("Ring pattern applied to %s ring successfully", ring)
	if len(segments) > 0 {
		message += fmt.Sprintf(" with %d segment(s)", len(segments))
	}
	if background != "" {
		message += fmt.Sprintf(", background: #%s", background)
	}
	if whirlMs > 0 {
		direction := "clockwise"
		if counterClockwise {
			direction = "counter-clockwise"
		}
		message += fmt.Sprintf(", rotation: %dms %s", whirlMs, direction)
	}
	if morphSpec != "" {
		message += fmt.Sprintf(", fade: %s", morphSpec)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: message,
			},
		},
		IsError: false,
	}, nil
}

// Helper functions

func isValidSegmentFormat(segment string) bool {
	parts := strings.Split(segment, "|")
	if len(parts) != 3 {
		return false
	}
	// Basic validation - could be more thorough
	return len(parts[2]) == 6 && isValidHexColor(parts[2])
}

func isValidHexColor(color string) bool {
	if len(color) != 6 {
		return false
	}
	for _, c := range color {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func isValidMorphSpec(spec string) bool {
	parts := strings.Split(spec, "|")
	return len(parts) == 2
}

func buildRingPatternCommand(ring string, segments []string, background string, whirlMs int, counterClockwise bool, morphSpec string) string {
	var parts []string
	
	// Add init
	parts = append(parts, fmt.Sprintf("%s_init=1", ring))
	
	// Add segments
	if len(segments) > 0 {
		segmentStr := strings.Join(segments, "|")
		parts = append(parts, fmt.Sprintf("%s=%s", ring, segmentStr))
	}
	
	// Add background
	if background != "" {
		parts = append(parts, fmt.Sprintf("%s_bg=%s", ring, background))
	}
	
	// Add whirl with optional counter-clockwise
	if whirlMs > 0 {
		whirlValue := fmt.Sprintf("%d", whirlMs)
		if counterClockwise {
			whirlValue += "|ccw"
		}
		parts = append(parts, fmt.Sprintf("%s_whirl=%s", ring, whirlValue))
	}
	
	// Add morph
	if morphSpec != "" {
		parts = append(parts, fmt.Sprintf("%s_morph=%s", ring, morphSpec))
	}
	
	return strings.Join(parts, "&")
}

// parseLedColors converts segment format to individual LED colors
func parseLedColors(segments []string, background string) []string {
	// Initialize with 15 LEDs (UFO has 15 LEDs per ring)
	colors := make([]string, 15)
	
	// Fill with background color or black if no background
	defaultColor := "000000"
	if background != "" {
		defaultColor = background
	}
	for i := range colors {
		colors[i] = defaultColor
	}
	
	// Apply segments
	for _, segment := range segments {
		parts := strings.Split(segment, "|")
		if len(parts) != 3 {
			continue
		}
		
		// Parse segment parameters
		var startLed, count int
		color := parts[2]
		
		if _, err := fmt.Sscanf(parts[0], "%d", &startLed); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(parts[1], "%d", &count); err != nil {
			continue
		}
		
		// Apply color to LEDs
		for i := 0; i < count && startLed+i < 15; i++ {
			colors[startLed+i] = color
		}
	}
	
	return colors
}