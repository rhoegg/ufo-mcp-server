package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

// StopEffectTool implements the stopEffect MCP tool
type StopEffectTool struct {
	client       *device.Client
	broadcaster  *events.Broadcaster
	stateManager *state.Manager
}

// NewStopEffectTool creates a new stopEffect tool instance
func NewStopEffectTool(client *device.Client, broadcaster *events.Broadcaster, stateManager *state.Manager) *StopEffectTool {
	return &StopEffectTool{
		client:       client,
		broadcaster:  broadcaster,
		stateManager: stateManager,
	}
}

// Definition returns the MCP tool definition for stopEffect
func (t *StopEffectTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "stopEffect",
		Description: "Stop the current effect and resume the previous one from the stack. If no previous effect exists, the UFO will be cleared.",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}
}

// Execute runs the stopEffect tool
func (t *StopEffectTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Get the current effect before popping
	currentEffect := t.stateManager.GetCurrentEffect()
	if currentEffect == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "No effect is currently running",
				},
			},
			IsError: false,
		}, nil
	}

	// Pop the current effect and get the previous one
	previousEffect := t.stateManager.PopEffect()
	
	var message string
	if previousEffect != nil {
		// Resume the previous effect
		query := previousEffect.Pattern
		_, err := t.client.SendRawQuery(ctx, query)
		if err != nil {
			t.broadcaster.PublishRawExecuted(query, fmt.Sprintf("ERROR: %v", err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to resume previous effect: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		
		t.broadcaster.PublishRawExecuted(query, "OK")
		
		// Emit effect resumed event
		t.broadcaster.Publish(events.Event{
			Type: events.EventEffectResumed,
			Data: map[string]interface{}{
				"effect":     previousEffect.Name,
				"stackDepth": t.stateManager.GetEffectStackDepth(),
			},
		})
		
		message = fmt.Sprintf("⏹️ Stopped '%s' and resumed '%s' (stack depth: %d)", 
			currentEffect.Name, previousEffect.Name, t.stateManager.GetEffectStackDepth())
	} else {
		// No previous effect, clear the UFO
		query := "top_init=1&bottom_init=1&logo=off"
		_, err := t.client.SendRawQuery(ctx, query)
		if err != nil {
			t.broadcaster.PublishRawExecuted(query, fmt.Sprintf("ERROR: %v", err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Failed to clear UFO: %v", err),
					},
				},
				IsError: true,
			}, nil
		}
		
		t.broadcaster.PublishRawExecuted(query, "OK")
		
		// Update LED state to all black
		t.stateManager.UpdateTopRing(make([]string, 15))
		t.stateManager.UpdateBottomRing(make([]string, 15))
		t.stateManager.UpdateLogo(false)
		
		message = fmt.Sprintf("⏹️ Stopped '%s' and cleared all LEDs (stack empty)", currentEffect.Name)
	}
	
	// Emit effect stopped event
	t.broadcaster.Publish(events.Event{
		Type: events.EventEffectStopped,
		Data: map[string]interface{}{
			"effect":     currentEffect.Name,
			"manual":     true,
			"stackDepth": t.stateManager.GetEffectStackDepth(),
		},
	})
	
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