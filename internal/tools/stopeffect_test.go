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
	t.Setenv("UFO_IP", server.URL[7:]) // Remove "http://" prefix

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
}