package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
	"github.com/starspace46/ufo-mcp-go/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopEffectTool(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Set test server host (without http:// prefix)
	setIP, _ := testutil.SetupTestEnv(t)
	setIP(server.URL[7:]) // Remove "http://" prefix

	// Create dependencies
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()
	stateManager := state.NewManager(broadcaster)

	// Create tool
	tool := NewStopEffectTool(client, broadcaster, stateManager)

	t.Run("Definition", func(t *testing.T) {
		def := tool.Definition()
		assert.Equal(t, "stopEffect", def.Name)
		assert.Contains(t, def.Description, "Stop the current effect")
		assert.Equal(t, "object", def.InputSchema.Type)
		assert.Empty(t, def.InputSchema.Properties)
	})

	t.Run("NoEffectRunning", func(t *testing.T) {
		// Execute with no effect on stack
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "No effect is currently running")
	})

	t.Run("StopEffectWithPrevious", func(t *testing.T) {
		// Push two effects onto the stack
		stateManager.PushEffect("effect1", "pattern1", map[string]interface{}{})
		stateManager.PushEffect("effect2", "pattern2", map[string]interface{}{})

		// Execute stopEffect
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "Stopped 'effect2' and resumed 'effect1'")
		
		// Verify stack depth
		assert.Equal(t, 1, stateManager.GetEffectStackDepth())
		
		// Verify current effect
		current := stateManager.GetCurrentEffect()
		assert.NotNil(t, current)
		assert.Equal(t, "effect1", current.Name)
	})

	t.Run("StopLastEffect", func(t *testing.T) {
		// Execute stopEffect to remove the last effect
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		textContent := result.Content[0].(mcp.TextContent)
		assert.Contains(t, textContent.Text, "Stopped 'effect1' and cleared all LEDs")
		
		// Verify stack is empty
		assert.Equal(t, 0, stateManager.GetEffectStackDepth())
		
		// Verify no current effect
		current := stateManager.GetCurrentEffect()
		assert.Nil(t, current)
		
		// Verify LED state is cleared
		ledState := stateManager.Snapshot()
		for i := 0; i < 15; i++ {
			assert.Equal(t, "000000", ledState.Top[i])
			assert.Equal(t, "000000", ledState.Bottom[i])
		}
		assert.False(t, ledState.LogoOn)
	})

	t.Run("StopEffectWithExpiredInStack", func(t *testing.T) {
		// Clear any existing effects
		for stateManager.GetEffectStackDepth() > 0 {
			stateManager.PopEffect()
		}

		// Push effects: temporary, permanent, temporary (current)
		stateManager.PushEffect("tempEffect1", "pattern1", map[string]interface{}{"duration": 5000})
		stateManager.PushEffect("permEffect", "pattern2", map[string]interface{}{"perpetual": true})
		stateManager.PushEffect("tempEffect2", "pattern3", map[string]interface{}{"duration": 10000})

		// Mark the bottom temporary effect as expired
		stateManager.MarkEffectExpired("tempEffect1")

		// Execute stopEffect
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Should stop tempEffect2 and resume permEffect (skipping expired tempEffect1)
		content := result.Content[0].(mcp.TextContent).Text
		assert.Contains(t, content, "Stopped 'tempEffect2'")
		assert.Contains(t, content, "resumed 'permEffect'")
		assert.Contains(t, content, "stack depth: 2") // 2 because expired effect is still in stack

		// Verify permEffect + expired tempEffect1 remain on stack
		assert.Equal(t, 2, stateManager.GetEffectStackDepth())
		current := stateManager.GetCurrentEffect()
		assert.NotNil(t, current)
		assert.Equal(t, "permEffect", current.Name)
	})

	t.Run("StopEffectAllExpiredBelowCurrent", func(t *testing.T) {
		// Clear any existing effects
		for stateManager.GetEffectStackDepth() > 0 {
			stateManager.PopEffect()
		}

		// Push effects: all temporary
		stateManager.PushEffect("tempEffect1", "pattern1", map[string]interface{}{"duration": 5000})
		stateManager.PushEffect("tempEffect2", "pattern2", map[string]interface{}{"duration": 10000})
		stateManager.PushEffect("tempEffect3", "pattern3", map[string]interface{}{"duration": 15000})

		// Mark all except current as expired
		stateManager.MarkEffectExpired("tempEffect1")
		stateManager.MarkEffectExpired("tempEffect2")

		// Execute stopEffect
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Should stop tempEffect3 and clear (all others expired)
		content := result.Content[0].(mcp.TextContent).Text
		assert.Contains(t, content, "Stopped 'tempEffect3'")
		assert.Contains(t, content, "cleared all LEDs")
		assert.Contains(t, content, "stack empty")

		// Verify stack is empty
		assert.Equal(t, 0, stateManager.GetEffectStackDepth())
	})

	t.Run("StopEffectWithExpiredAndBaseState", func(t *testing.T) {
		// Clear any existing effects
		for stateManager.GetEffectStackDepth() > 0 {
			stateManager.PopEffect()
		}

		// Set a base state
		stateManager.SetBaseState("base_pattern")

		// Push effects
		stateManager.PushEffect("tempEffect1", "pattern1", map[string]interface{}{"duration": 5000})
		stateManager.PushEffect("tempEffect2", "pattern2", map[string]interface{}{"duration": 10000})

		// Mark first effect as expired
		stateManager.MarkEffectExpired("tempEffect1")

		// Execute stopEffect
		result, err := tool.Execute(context.Background(), map[string]interface{}{})
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Should stop tempEffect2 and resume base state (tempEffect1 is expired)
		content := result.Content[0].(mcp.TextContent).Text
		assert.Contains(t, content, "Stopped 'tempEffect2'")
		assert.Contains(t, content, "resumed '__base_state__'")
		
		// Verify stack is empty but base state is active
		assert.Equal(t, 0, stateManager.GetEffectStackDepth())
		
		// Clear base state for other tests
		stateManager.SetBaseState("")
	})
}