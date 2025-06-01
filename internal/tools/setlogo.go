package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// SetLogoTool implements the setLogo MCP tool
type SetLogoTool struct {
	client       *device.Client
	broadcaster  *events.Broadcaster
	stateManager *state.Manager
}

// NewSetLogoTool creates a new setLogo tool instance
func NewSetLogoTool(client *device.Client, broadcaster *events.Broadcaster, stateManager *state.Manager) *SetLogoTool {
	return &SetLogoTool{
		client:       client,
		broadcaster:  broadcaster,
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for setLogo
func (t *SetLogoTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "setLogo",
		Description: "Control the Dynatrace logo LED. Can turn it on/off or set custom colors. The logo supports two colors that can create gradients or patterns.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Logo LED state - 'on' to turn on, 'off' to turn off",
					"enum":        []string{"on", "off"},
					"examples":    []string{"on", "off"},
				},
				"color1": map[string]interface{}{
					"type":        "string",
					"description": "First color in hex format (e.g., 'FF0000' for red). Optional - only used when state is 'on'",
					"pattern":     "^[0-9A-Fa-f]{6}$",
					"examples":    []string{"FF0000", "00FF00", "0000FF", "8B0000"},
				},
				"color2": map[string]interface{}{
					"type":        "string",
					"description": "Second color in hex format (e.g., 'FF6B6B' for light red). Optional - creates gradient with color1",
					"pattern":     "^[0-9A-Fa-f]{6}$",
					"examples":    []string{"FF6B6B", "90EE90", "ADD8E6", "FFB6C1"},
				},
			},
			Required: []string{"state"},
		},
	}
}

// Execute runs the setLogo tool
func (t *SetLogoTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract state parameter
	stateArg, exists := arguments["state"]
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'state' parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	state, ok := stateArg.(string)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'state' parameter must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Validate state value
	if state != "on" && state != "off" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'state' must be either 'on' or 'off'",
				},
			},
			IsError: true,
		}, nil
	}

	// Extract optional color parameters
	color1, _ := arguments["color1"].(string)
	color2, _ := arguments["color2"].(string)

	// Build the query string
	var query string
	
	if state == "off" {
		// When turning off, explicitly set to black pattern to ensure it turns off
		// This handles cases where colors were previously set
		query = "logo=000000|000000|000000|000000"
	} else if state == "on" && (color1 != "" || color2 != "") {
		// When colors are provided, use the pattern format
		pattern := ""
		
		if color1 != "" {
			// Validate color format
			if !isValidHexColor(color1) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'color1' must be a valid 6-character hex color",
						},
					},
					IsError: true,
				}, nil
			}
			pattern = color1
		}
		
		if color2 != "" {
			// Validate color format
			if !isValidHexColor(color2) {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: 'color2' must be a valid 6-character hex color",
						},
					},
					IsError: true,
				}, nil
			}
			if pattern != "" {
				// Create alternating pattern like ff0000|ffffff|ff0000|ffffff
				pattern = fmt.Sprintf("%s|%s|%s|%s", color1, color2, color1, color2)
			} else {
				pattern = color2
			}
		}
		
		query = fmt.Sprintf("logo=%s", pattern)
	} else {
		// Simple on with default color
		query = fmt.Sprintf("logo=%s", state)
	}

	// Execute the logo command
	_, err := t.client.SendRawQuery(ctx, query)
	if err != nil {
		// Publish the failed execution event
		t.broadcaster.PublishRawExecuted(query, fmt.Sprintf("ERROR: %v", err))
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to set logo: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Update shadow state
	t.stateManager.UpdateLogo(state == "on")
	
	// Publish the successful execution event
	t.broadcaster.PublishRawExecuted(query, "OK")

	// Build response message
	message := fmt.Sprintf("Logo LED turned %s successfully", state)
	if state == "on" && (color1 != "" || color2 != "") {
		message += " with colors"
		if color1 != "" {
			message += fmt.Sprintf(" #%s", color1)
		}
		if color2 != "" {
			message += fmt.Sprintf(" and #%s", color2)
		}
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