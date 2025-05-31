package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

func TestGetLedStateTool_Definition(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()
	stateManager := state.NewManager(broadcaster)

	tool := NewGetLedStateTool(stateManager)
	def := tool.Definition()

	if def.Name != "getLedState" {
		t.Errorf("expected tool name 'getLedState', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check schema - should have no required parameters
	schema := def.InputSchema
	if len(schema.Required) != 0 {
		t.Error("getLedState should have no required parameters")
	}
}

func TestGetLedStateTool_Execute(t *testing.T) {
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()
	stateManager := state.NewManager(broadcaster)

	// Set some state
	stateManager.UpdateBrightness(150)
	stateManager.UpdateLogo(true)
	stateManager.UpdateRingSegments("top", []string{"FF0000", "00FF00", "0000FF"}, "")

	tool := NewGetLedStateTool(stateManager)

	// Execute the tool
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Error("Execute returned error result")
	}

	// Check that we got content
	if len(result.Content) == 0 {
		t.Error("No content returned")
	}

	// Verify the content is valid JSON
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent type")
	}
	contentText := textContent.Text
	
	// Extract JSON from the response (skip the header line)
	jsonStart := len("Current UFO LED State:\n")
	jsonContent := contentText[jsonStart:]

	var ledState struct {
		Top    []string `json:"top"`
		Bottom []string `json:"bottom"`
		LogoOn bool     `json:"logoOn"`
		Effect string   `json:"effect"`
		Dim    int      `json:"dim"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &ledState); err != nil {
		t.Fatalf("Failed to parse LED state JSON: %v", err)
	}

	// Verify the state values
	if ledState.Dim != 150 {
		t.Errorf("Expected brightness 150, got %d", ledState.Dim)
	}

	if !ledState.LogoOn {
		t.Error("Expected logo to be on")
	}

	if ledState.Top[0] != "FF0000" {
		t.Errorf("Expected top[0] to be FF0000, got %s", ledState.Top[0])
	}
}