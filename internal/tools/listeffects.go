package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

// ListEffectsTool implements the listEffects MCP tool
type ListEffectsTool struct {
	store *effects.Store
}

// NewListEffectsTool creates a new listEffects tool instance
func NewListEffectsTool(store *effects.Store) *ListEffectsTool {
	return &ListEffectsTool{
		store: store,
	}
}

// Definition returns the MCP tool definition for listEffects
func (t *ListEffectsTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "listEffects",
		Description: "List all available lighting effects, including both built-in seed effects and user-defined custom effects. Returns an array of effect objects with name, description, pattern, and duration.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
			Required:   []string{},
		},
	}
}

// Execute runs the listEffects tool
func (t *ListEffectsTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get all effects from the store
	effectsList := t.store.List()

	// Convert to JSON for display
	effectsJSON, err := json.MarshalIndent(effectsList, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Failed to serialize effects: " + err.Error(),
				},
			},
			IsError: true,
		}, nil
	}

	// Build a summary message
	message := "Available UFO Lighting Effects:\n"
	message += "================================\n\n"
	
	for _, effect := range effectsList {
		message += fmt.Sprintf("• %s - %s\n", effect.Name, effect.Description)
		message += fmt.Sprintf("  Duration: %d seconds\n", effect.Duration)
		message += fmt.Sprintf("  Pattern: %s\n\n", effect.Pattern)
	}
	
	message += fmt.Sprintf("Total effects: %d\n\n", len(effectsList))
	message += "Full JSON:\n" + string(effectsJSON)

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