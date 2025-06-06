package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// PlayEffectTool implements the playEffect MCP tool
type PlayEffectTool struct {
	client       *device.Client
	broadcaster  *events.Broadcaster
	store        *effects.Store
	stateManager *state.Manager
}

// NewPlayEffectTool creates a new playEffect tool instance
func NewPlayEffectTool(client *device.Client, broadcaster *events.Broadcaster, store *effects.Store, stateManager *state.Manager) *PlayEffectTool {
	return &PlayEffectTool{
		client:       client,
		broadcaster:  broadcaster,
		store:        store,
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for playEffect
func (t *PlayEffectTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "playEffect",
		Description: "Play a lighting effect by name. Effects run for their configured duration or until stopped. Returns immediately while the effect plays.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the effect to play (e.g., 'rainbow', 'policeLights')",
				},
				"duration": map[string]interface{}{
					"type":        "number",
					"description": "Override duration in milliseconds (optional, uses effect's default if not specified)",
				},
			},
			Required: []string{"name"},
		},
	}
}

// Execute runs the playEffect tool
func (t *PlayEffectTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract effect name
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

	// Get the effect from store
	effect, exists := t.store.Get(name)
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Effect '%s' not found. Use listEffects to see available effects.", name),
				},
			},
			IsError: true,
		}, nil
	}

	// Check for duration override
	duration := effect.Duration
	if durationVal, hasDuration := arguments["duration"]; hasDuration {
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

		// Validate duration
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
	}

	// Send the effect pattern to the UFO
	query := effect.Pattern
	if _, err := t.client.SendRawQuery(ctx, query); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error: Failed to send effect to UFO: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Push effect onto stack
	effectContext := map[string]interface{}{
		"duration":  duration,
		"perpetual": effect.Perpetual,
		"startTime": time.Now(),
	}
	t.stateManager.PushEffect(name, effect.Pattern, effectContext)

	// Emit effect started event
	t.broadcaster.Publish(events.Event{
		Type: events.EventEffectStarted,
		Data: map[string]interface{}{
			"effect":     name,
			"duration":   duration,
			"pattern":    effect.Pattern,
			"stackDepth": t.stateManager.GetEffectStackDepth(),
		},
	})

	// Build response message
	message := fmt.Sprintf("✨ Effect '%s' started!\n\n", name)
	message += fmt.Sprintf("• Description: %s\n", effect.Description)
	if effect.Perpetual {
		message += "• Duration: Perpetual (runs until stopped)\n"
	} else if duration > 0 {
		message += fmt.Sprintf("• Duration: %d ms (%.1f seconds)\n", duration, float64(duration)/1000)
		message += fmt.Sprintf("• Will stop at: %s\n", time.Now().Add(time.Duration(duration)*time.Millisecond).Format("15:04:05"))
	} else {
		message += "• Duration: Infinite (use stopEffects to stop)\n"
	}
	message += fmt.Sprintf("\nPattern sent: %s", effect.Pattern)

	// Start a goroutine to handle effect completion for timed effects
	if duration > 0 && !effect.Perpetual {
		go func() {
			time.Sleep(time.Duration(duration) * time.Millisecond)
			
			// Pop the effect and get the previous one
			previousEffect := t.stateManager.PopEffect()
			
			if previousEffect != nil {
				// Resume the previous effect
				t.client.SendRawQuery(context.Background(), previousEffect.Pattern)
				
				// Emit effect resumed event
				t.broadcaster.Publish(events.Event{
					Type: events.EventEffectResumed,
					Data: map[string]interface{}{
						"effect":     previousEffect.Name,
						"stackDepth": t.stateManager.GetEffectStackDepth(),
					},
				})
			} else {
				// No previous effect, clear the UFO
				t.client.SendRawQuery(context.Background(), "top_init=1&bottom_init=1")
			}
			
			// Emit effect completed event
			t.broadcaster.Publish(events.Event{
				Type: events.EventEffectCompleted,
				Data: map[string]interface{}{
					"effect":     name,
					"stackDepth": t.stateManager.GetEffectStackDepth(),
				},
			})
		}()
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