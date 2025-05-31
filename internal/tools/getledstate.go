package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// GetLedStateTool implements a tool to get the current LED state
type GetLedStateTool struct {
	stateManager *state.Manager
}

// NewGetLedStateTool creates a new getLedState tool instance
func NewGetLedStateTool(stateManager *state.Manager) *GetLedStateTool {
	return &GetLedStateTool{
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for getLedState
func (t *GetLedStateTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "getLedState",
		Description: "Get the current LED state showing all LED colors, brightness level, logo state, and any running effect. Returns the shadow state maintained by the MCP server.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
	}
}

// Execute runs the getLedState tool
func (t *GetLedStateTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get the current LED state as JSON
	ledStateJSON, err := t.stateManager.ToJSON()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Failed to get LED state: " + err.Error(),
				},
			},
			IsError: true,
		}, nil
	}

	// Return formatted response
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "Current UFO LED State:\n" + ledStateJSON,
			},
		},
		IsError: false,
	}, nil
}