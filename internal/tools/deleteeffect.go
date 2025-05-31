package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

// DeleteEffectTool implements the deleteEffect MCP tool
type DeleteEffectTool struct {
	store *effects.Store
}

// NewDeleteEffectTool creates a new deleteEffect tool instance
func NewDeleteEffectTool(store *effects.Store) *DeleteEffectTool {
	return &DeleteEffectTool{
		store: store,
	}
}

// Definition returns the MCP tool definition for deleteEffect
func (t *DeleteEffectTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "deleteEffect",
		Description: "Delete a custom lighting effect. Seed effects (built-in effects) cannot be deleted. This operation is permanent.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the effect to delete",
				},
			},
			Required: []string{"name"},
		},
	}
}

// Execute runs the deleteEffect tool
func (t *DeleteEffectTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract and validate name
	name, ok := arguments["name"].(string)
	if !ok || name == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'name' parameter is required and must be a non-empty string",
				},
			},
			IsError: true,
		}, nil
	}

	// Check if effect exists
	effect, exists := t.store.Get(name)
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Effect '%s' not found", name),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if it's a seed effect (seed effects have specific known names)
	seedEffects := []string{"rainbow", "policeLights", "breathingGreen", "pipelineDemo", "ipDisplay"}
	for _, seedName := range seedEffects {
		if name == seedName {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error: Cannot delete seed effect '%s'. Only custom effects can be deleted.", name),
					},
				},
				IsError: true,
			}, nil
		}
	}

	// Delete the effect
	if err := t.store.Delete(name); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Failed to delete effect: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Build success message
	message := fmt.Sprintf("Successfully deleted effect '%s'\n\n", name)
	message += "Effect details that were removed:\n"
	message += fmt.Sprintf("• Description: %s\n", effect.Description)
	message += fmt.Sprintf("• Pattern: %s\n", effect.Pattern)
	message += fmt.Sprintf("• Duration: %d seconds", effect.Duration)
	if effect.Duration == 0 {
		message += " (infinite)"
	}
	message += "\n\nThis operation is permanent and cannot be undone."

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