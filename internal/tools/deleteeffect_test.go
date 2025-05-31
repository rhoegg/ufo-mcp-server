package tools

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

func TestDeleteEffectTool_Definition(t *testing.T) {
	store := effects.NewStore("/tmp/test-effects.json")
	tool := NewDeleteEffectTool(store)
	def := tool.Definition()

	if def.Name != "deleteEffect" {
		t.Errorf("expected tool name 'deleteEffect', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check required parameters
	schema := def.InputSchema
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Error("only 'name' should be required")
	}
}

func TestDeleteEffectTool_Execute_Success(t *testing.T) {
	// Use a temp file for testing
	tmpFile := "/tmp/test-delete-effects.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load() // Initialize with seed effects

	// Add test effects
	testEffects := []string{"testEffect1", "testEffect2", "testEffect3"}
	for i, name := range testEffects {
		store.Add(&effects.Effect{
			Name:        name,
			Description: "Test effect " + name,
			Pattern:     "test=1",
			Duration:    10 * (i + 1),
		})
	}
	store.Save()

	tool := NewDeleteEffectTool(store)

	// Delete each test effect
	for _, effectName := range testEffects {
		t.Run("delete "+effectName, func(t *testing.T) {
			// Verify effect exists before deletion
			_, exists := store.Get(effectName)
			if !exists {
				t.Fatalf("Effect '%s' should exist before deletion", effectName)
			}

			result, err := tool.Execute(context.Background(), map[string]interface{}{
				"name": effectName,
			})

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

			// Check success message
			if !strings.Contains(textContent.Text, "Successfully deleted effect") {
				t.Error("Expected success message")
			}
			if !strings.Contains(textContent.Text, effectName) {
				t.Errorf("Expected effect name '%s' in response", effectName)
			}
			if !strings.Contains(textContent.Text, "permanent") {
				t.Error("Expected warning about permanent deletion")
			}

			// Verify effect was actually deleted from store
			_, exists = store.Get(effectName)
			if exists {
				t.Errorf("Effect '%s' should not exist after deletion", effectName)
			}
		})
	}
}

func TestDeleteEffectTool_Execute_Errors(t *testing.T) {
	tmpFile := "/tmp/test-delete-effects-errors.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load() // Load seed effects

	// Add a custom effect for testing
	store.Add(&effects.Effect{
		Name:        "customEffect",
		Description: "A custom effect",
		Pattern:     "test=1",
		Duration:    10,
	})
	store.Save()

	tool := NewDeleteEffectTool(store)

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name:        "missing name",
			arguments:   map[string]interface{}{},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "empty name",
			arguments:   map[string]interface{}{
				"name": "",
			},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "non-existent effect",
			arguments:   map[string]interface{}{
				"name": "doesNotExist",
			},
			expectError: true,
			expectText:  "not found",
		},
		{
			name:        "delete seed effect rainbow",
			arguments:   map[string]interface{}{
				"name": "rainbow",
			},
			expectError: true,
			expectText:  "Cannot delete seed effect",
		},
		{
			name:        "delete seed effect policeLights",
			arguments:   map[string]interface{}{
				"name": "policeLights",
			},
			expectError: true,
			expectText:  "Cannot delete seed effect",
		},
		{
			name:        "delete seed effect breathingGreen",
			arguments:   map[string]interface{}{
				"name": "breathingGreen",
			},
			expectError: true,
			expectText:  "Cannot delete seed effect",
		},
		{
			name:        "delete seed effect pipelineDemo",
			arguments:   map[string]interface{}{
				"name": "pipelineDemo",
			},
			expectError: true,
			expectText:  "Cannot delete seed effect",
		},
		{
			name:        "delete seed effect ipDisplay",
			arguments:   map[string]interface{}{
				"name": "ipDisplay",
			},
			expectError: true,
			expectText:  "Cannot delete seed effect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.arguments)
			
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}

			if result.IsError != tt.expectError {
				t.Errorf("expected IsError=%v, got %v", tt.expectError, result.IsError)
			}

			if len(result.Content) == 0 {
				t.Fatal("result should have content")
			}

			content, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatalf("content should be TextContent, got %T", result.Content[0])
			}

			if !strings.Contains(content.Text, tt.expectText) {
				t.Errorf("expected text to contain '%s', got '%s'", tt.expectText, content.Text)
			}
		})
	}
}

func TestDeleteEffectTool_VerifyPermanentDeletion(t *testing.T) {
	// Test that deletion is actually permanent (survives reload)
	tmpFile := "/tmp/test-permanent-delete.json"
	defer os.Remove(tmpFile)

	// Create first store instance
	store1 := effects.NewStore(tmpFile)
	store1.Load()

	// Add an effect
	store1.Add(&effects.Effect{
		Name:        "permanentTest",
		Description: "Will be deleted",
		Pattern:     "test=1",
		Duration:    10,
	})
	store1.Save()

	// Delete the effect
	tool := NewDeleteEffectTool(store1)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"name": "permanentTest",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.IsError {
		t.Fatal("Delete failed")
	}

	// Create new store instance and reload from disk
	store2 := effects.NewStore(tmpFile)
	store2.Load()

	// Verify effect is still gone
	_, exists := store2.Get("permanentTest")
	if exists {
		t.Error("Deleted effect should not exist after reload")
	}
}