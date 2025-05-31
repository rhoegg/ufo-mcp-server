package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
)

// SetBrightnessTool implements the setBrightness MCP tool
type SetBrightnessTool struct {
	client      *device.Client
	broadcaster *events.Broadcaster
}

// NewSetBrightnessTool creates a new setBrightness tool instance
func NewSetBrightnessTool(client *device.Client, broadcaster *events.Broadcaster) *SetBrightnessTool {
	return &SetBrightnessTool{
		client:      client,
		broadcaster: broadcaster,
	}
}

// Definition returns the MCP tool definition for setBrightness
func (t *SetBrightnessTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "setBrightness",
		Description: "Set the global brightness level for all UFO LEDs. This affects the intensity of all lighting effects and colors.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"level": map[string]interface{}{
					"type":        "integer",
					"description": "Brightness level from 0 (off) to 255 (maximum brightness)",
					"minimum":     0,
					"maximum":     255,
					"examples":    []int{0, 64, 128, 192, 255},
				},
			},
			Required: []string{"level"},
		},
	}
}

// Execute runs the setBrightness tool
func (t *SetBrightnessTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract level parameter
	levelArg, exists := arguments["level"]
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'level' parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	// Handle both int and float64 (JSON numbers are float64 by default)
	var level int
	switch v := levelArg.(type) {
	case int:
		level = v
	case float64:
		level = int(v)
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'level' parameter must be a number",
				},
			},
			IsError: true,
		}, nil
	}

	// Validate level range
	if level < 0 || level > 255 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: brightness level must be between 0 and 255",
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the brightness command
	err := t.client.SetBrightness(ctx, level)
	if err != nil {
		// Publish the failed execution event
		t.broadcaster.PublishRawExecuted(fmt.Sprintf("dim=%d", level), fmt.Sprintf("ERROR: %v", err))
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to set brightness: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Publish the successful execution event and dim changed event
	t.broadcaster.PublishRawExecuted(fmt.Sprintf("dim=%d", level), "OK")
	t.broadcaster.PublishDimChanged(level)

	// Calculate percentage for user-friendly display
	percentage := int(float64(level) / 255.0 * 100)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Brightness set to %d/255 (%d%%) successfully", level, percentage),
			},
		},
		IsError: false,
	}, nil
}