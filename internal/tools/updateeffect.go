package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

// UpdateEffectTool implements the updateEffect MCP tool
type UpdateEffectTool struct {
	store *effects.Store
}

// NewUpdateEffectTool creates a new updateEffect tool instance
func NewUpdateEffectTool(store *effects.Store) *UpdateEffectTool {
	return &UpdateEffectTool{
		store: store,
	}
}

// Definition returns the MCP tool definition for updateEffect
func (t *UpdateEffectTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "updateEffect",
		Description: "Update an existing custom lighting effect. You can update the description, pattern, and/or duration. The effect name cannot be changed.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the effect to update",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "New description (optional, leave unset to keep current)",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "New UFO API pattern string (optional, leave unset to keep current)",
				},
				"duration": map[string]interface{}{
					"type":        "number",
					"description": "New duration in milliseconds 0-3600000 (optional, leave unset to keep current)",
				},
			},
			Required: []string{"name"},
		},
	}
}

// Execute runs the updateEffect tool
func (t *UpdateEffectTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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
	existingEffect, exists := t.store.Get(name)
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Effect '%s' not found. Use addEffect to create it first.", name),
				},
			},
			IsError: true,
		}, nil
	}

	// Create updated effect starting with existing values
	updatedEffect := &effects.Effect{
		Name:        existingEffect.Name,
		Description: existingEffect.Description,
		Pattern:     existingEffect.Pattern,
		Duration:    existingEffect.Duration,
	}

	// Track what was updated
	var updates []string

	// Update description if provided
	if descVal, hasDesc := arguments["description"]; hasDesc {
		description, ok := descVal.(string)
		if !ok || description == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'description' must be a non-empty string when provided",
					},
				},
				IsError: true,
			}, nil
		}
		updatedEffect.Description = description
		updates = append(updates, "description")
	}

	// Update pattern if provided
	if patternVal, hasPattern := arguments["pattern"]; hasPattern {
		pattern, ok := patternVal.(string)
		if !ok || pattern == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'pattern' must be a non-empty string when provided",
					},
				},
				IsError: true,
			}, nil
		}
		updatedEffect.Pattern = pattern
		updates = append(updates, "pattern")
	}

	// Update duration if provided
	if durationVal, hasDuration := arguments["duration"]; hasDuration {
		var duration int
		switch v := durationVal.(type) {
		case float64:
			duration = int(v)
		case int:
			duration = v
		default:
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'duration' must be a number",
					},
				},
				IsError: true,
			}, nil
		}

		// Validate duration range
		if duration < 0 || duration > 3600000 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: 'duration' must be between 0 and 3600000 milliseconds (1 hour)",
					},
				},
				IsError: true,
			}, nil
		}

		updatedEffect.Duration = duration
		updates = append(updates, "duration")
	}

	// Check if any updates were provided
	if len(updates) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: No updates provided. Specify at least one of: description, pattern, or duration",
				},
			},
			IsError: true,
		}, nil
	}

	// Update the effect in the store (Update saves automatically)
	if err := t.store.Update(updatedEffect); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Failed to update effect: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Build success message
	message := fmt.Sprintf("Successfully updated effect '%s'\n\n", name)
	message += fmt.Sprintf("Updated fields: %v\n\n", updates)
	message += "Current values:\n"
	message += fmt.Sprintf("• Name: %s\n", updatedEffect.Name)
	message += fmt.Sprintf("• Description: %s\n", updatedEffect.Description)
	message += fmt.Sprintf("• Pattern: %s\n", updatedEffect.Pattern)
	message += fmt.Sprintf("• Duration: %d seconds", updatedEffect.Duration)
	if updatedEffect.Duration == 0 {
		message += " (infinite)"
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