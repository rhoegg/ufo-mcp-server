package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

func TestListEffectsTool_Definition(t *testing.T) {
	store := effects.NewStore("/tmp/test-effects.json")
	tool := NewListEffectsTool(store)
	def := tool.Definition()

	if def.Name != "listEffects" {
		t.Errorf("expected tool name 'listEffects', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check schema has no required parameters
	schema := def.InputSchema
	if len(schema.Required) != 0 {
		t.Error("listEffects should have no required parameters")
	}
}

func TestListEffectsTool_Execute(t *testing.T) {
	// Create a temporary store with test effects
	store := effects.NewStore("/tmp/test-effects.json")
	
	// Add some test effects
	testEffects := []*effects.Effect{
		{
			Name:        "testEffect1",
			Description: "Test effect one",
			Pattern:     "test=1",
			Duration:    10,
		},
		{
			Name:        "testEffect2",
			Description: "Test effect two",
			Pattern:     "test=2",
			Duration:    20,
		},
	}
	
	// Manually add effects to the store
	store.Load() // Initialize with seed effects
	for _, effect := range testEffects {
		store.Add(effect)
	}

	tool := NewListEffectsTool(store)

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

	// Verify the content
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent type")
	}

	// Check that our test effects are included
	if !strings.Contains(textContent.Text, "testEffect1") {
		t.Error("testEffect1 not found in output")
	}
	if !strings.Contains(textContent.Text, "testEffect2") {
		t.Error("testEffect2 not found in output")
	}
	
	// Check for seed effects (should be included)
	if !strings.Contains(textContent.Text, "rainbow") {
		t.Error("rainbow seed effect not found in output")
	}
	
	// Check total count (5 seed effects + 2 test effects = 7)
	if !strings.Contains(textContent.Text, "Total effects: 7") {
		t.Error("Expected total of 7 effects")
	}
}