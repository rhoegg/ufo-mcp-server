package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

// AddEffectTool implements the addEffect MCP tool
type AddEffectTool struct {
	store *effects.Store
}

// NewAddEffectTool creates a new addEffect tool instance
func NewAddEffectTool(store *effects.Store) *AddEffectTool {
	return &AddEffectTool{
		store: store,
	}
}

// Definition returns the MCP tool definition for addEffect
func (t *AddEffectTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "addEffect",
		Description: "Add a new custom lighting effect. The effect will be persisted to the effects database. Name must be unique.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Unique name for the effect (e.g. 'myRainbow', 'alertPulse')",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Human-readable description of what this effect does",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "UFO API pattern string (e.g. 'top=0|5|FF0000|10|5|00FF00&bottom_whirl=300')",
				},
				"duration": map[string]interface{}{
					"type":        "number",
					"description": "Duration in seconds (0-3600, 0 means infinite)",
				},
			},
			Required: []string{"name", "description", "pattern"},
		},
	}
}

// Execute runs the addEffect tool
func (t *AddEffectTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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

	// Validate name format (alphanumeric + underscore)
	if !isValidEffectName(name) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Effect name must contain only letters, numbers, and underscores",
				},
			},
			IsError: true,
		}, nil
	}

	// Check if effect already exists
	_, exists := t.store.Get(name)
	if exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Effect '%s' already exists. Use updateEffect to modify it.", name),
				},
			},
			IsError: true,
		}, nil
	}

	// Extract description
	description, ok := arguments["description"].(string)
	if !ok || description == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'description' parameter is required and must be a non-empty string",
				},
			},
			IsError: true,
		}, nil
	}

	// Extract pattern
	pattern, ok := arguments["pattern"].(string)
	if !ok || pattern == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'pattern' parameter is required and must be a non-empty string",
				},
			},
			IsError: true,
		}, nil
	}

	// Extract duration (optional, defaults to 0)
	duration := 0
	if durationVal, exists := arguments["duration"]; exists {
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
	}

	// Validate duration range
	if duration < 0 || duration > 3600 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'duration' must be between 0 and 3600 seconds",
				},
			},
			IsError: true,
		}, nil
	}

	// Create the new effect
	newEffect := &effects.Effect{
		Name:        name,
		Description: description,
		Pattern:     pattern,
		Duration:    duration,
	}

	// Add to store
	t.store.Add(newEffect)

	// Save to disk
	if err := t.store.Save(); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Failed to save effect: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Success message
	message := fmt.Sprintf("Successfully added new effect '%s'\n\n", name)
	message += fmt.Sprintf("Details:\n")
	message += fmt.Sprintf("• Name: %s\n", name)
	message += fmt.Sprintf("• Description: %s\n", description)
	message += fmt.Sprintf("• Pattern: %s\n", pattern)
	message += fmt.Sprintf("• Duration: %d seconds", duration)
	if duration == 0 {
		message += " (infinite)"
	}
	message += "\n\nYou can now use playEffect to activate this effect."

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

// isValidEffectName checks if the effect name contains only valid characters
func isValidEffectName(name string) bool {
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}