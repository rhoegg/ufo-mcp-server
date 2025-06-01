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
							"type":        "string",
							"description": "Morph/fade pattern 'STAY|SPEED' in ms",
							"examples":    []string{"1000|500", "2000|1000"},
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
							"type":        "string",
							"description": "Morph/fade pattern",
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

	// Process morph
	if morphVal, hasMorph := config["morph"]; hasMorph {
		morph, ok := morphVal.(string)
		if !ok {
			return "", "", fmt.Errorf("morph must be a string")
		}
		if !isValidMorphSpec(morph) {
			return "", "", fmt.Errorf("invalid morph spec: %s", morph)
		}
		queryParts = append(queryParts, fmt.Sprintf("%s_morph=%s", ring, morph))
		message = append(message, fmt.Sprintf("morphing %s", morph))
	}

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