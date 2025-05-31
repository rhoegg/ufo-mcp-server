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
		Description: "Turn the Dynatrace logo LED on or off. The logo is a small LED that displays the Dynatrace logo on the UFO device.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Logo LED state - 'on' to turn on, 'off' to turn off",
					"enum":        []string{"on", "off"},
					"examples":    []string{"on", "off"},
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

	// Execute the logo command
	err := t.client.SetLogo(ctx, state)
	if err != nil {
		// Publish the failed execution event
		t.broadcaster.PublishRawExecuted(fmt.Sprintf("logo=%s", state), fmt.Sprintf("ERROR: %v", err))
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to set logo state: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Update shadow state
	t.stateManager.UpdateLogo(state == "on")
	
	// Publish the successful execution event
	t.broadcaster.PublishRawExecuted(fmt.Sprintf("logo=%s", state), "OK")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Logo LED turned %s successfully", state),
			},
		},
		IsError: false,
	}, nil
}