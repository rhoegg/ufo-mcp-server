package tools

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

func TestUpdateEffectTool_Definition(t *testing.T) {
	store := effects.NewStore("/tmp/test-effects.json")
	tool := NewUpdateEffectTool(store)
	def := tool.Definition()

	if def.Name != "updateEffect" {
		t.Errorf("expected tool name 'updateEffect', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check required parameters
	schema := def.InputSchema
	if len(schema.Required) != 1 || schema.Required[0] != "name" {
		t.Error("only 'name' should be required")
	}

	// Check optional parameters exist
	optionalParams := []string{"description", "pattern", "duration"}
	for _, param := range optionalParams {
		if _, exists := schema.Properties[param]; !exists {
			t.Errorf("expected optional parameter '%s' to be defined", param)
		}
	}
}

func TestUpdateEffectTool_Execute_Success(t *testing.T) {
	// Use a temp file for testing
	tmpFile := "/tmp/test-update-effects.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load() // Initialize with seed effects

	// Add test effects
	testEffect := &effects.Effect{
		Name:        "testEffect",
		Description: "Original description",
		Pattern:     "top=0|15|FF0000",
		Duration:    30,
	}
	store.Add(testEffect)
	store.Save()

	tool := NewUpdateEffectTool(store)

	tests := []struct {
		name           string
		arguments      map[string]interface{}
		expectedFields []string
	}{
		{
			name: "update description only",
			arguments: map[string]interface{}{
				"name":        "testEffect",
				"description": "Updated description",
			},
			expectedFields: []string{"description"},
		},
		{
			name: "update pattern only",
			arguments: map[string]interface{}{
				"name":    "testEffect",
				"pattern": "bottom=0|15|00FF00&bottom_whirl=300",
			},
			expectedFields: []string{"pattern"},
		},
		{
			name: "update duration only",
			arguments: map[string]interface{}{
				"name":     "testEffect",
				"duration": 120,
			},
			expectedFields: []string{"duration"},
		},
		{
			name: "update all fields",
			arguments: map[string]interface{}{
				"name":        "testEffect",
				"description": "Fully updated effect",
				"pattern":     "top=0|5|FF0000|10|5|00FF00&logo=1",
				"duration":    300,
			},
			expectedFields: []string{"description", "pattern", "duration"},
		},
		{
			name: "update seed effect",
			arguments: map[string]interface{}{
				"name":        "rainbow",
				"description": "Modified rainbow effect",
			},
			expectedFields: []string{"description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(context.Background(), tt.arguments)
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
			if !strings.Contains(textContent.Text, "Successfully updated effect") {
				t.Error("Expected success message")
			}

			// Check updated fields are mentioned
			for _, field := range tt.expectedFields {
				if !strings.Contains(textContent.Text, field) {
					t.Errorf("Expected field '%s' to be mentioned in update message", field)
				}
			}

			// Verify effect was actually updated in store
			effectName := tt.arguments["name"].(string)
			updatedEffect, exists := store.Get(effectName)
			if !exists {
				t.Errorf("Effect '%s' not found after update", effectName)
			} else {
				// Verify specific field updates
				if descVal, hasDesc := tt.arguments["description"]; hasDesc {
					if updatedEffect.Description != descVal.(string) {
						t.Error("Description was not updated correctly")
					}
				}
				if patternVal, hasPattern := tt.arguments["pattern"]; hasPattern {
					if updatedEffect.Pattern != patternVal.(string) {
						t.Error("Pattern was not updated correctly")
					}
				}
				if durationVal, hasDuration := tt.arguments["duration"]; hasDuration {
					var expectedDuration int
					switch v := durationVal.(type) {
					case float64:
						expectedDuration = int(v)
					case int:
						expectedDuration = v
					}
					if updatedEffect.Duration != expectedDuration {
						t.Errorf("Duration was not updated correctly: expected %d, got %d", 
							expectedDuration, updatedEffect.Duration)
					}
				}
			}
		})
	}
}

func TestUpdateEffectTool_Execute_Errors(t *testing.T) {
	tmpFile := "/tmp/test-update-effects-errors.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load()

	// Add a test effect
	store.Add(&effects.Effect{
		Name:        "existingEffect",
		Description: "Test effect",
		Pattern:     "test=1",
		Duration:    10,
	})
	store.Save()

	tool := NewUpdateEffectTool(store)

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name:        "missing name",
			arguments:   map[string]interface{}{
				"description": "Updated",
			},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "empty name",
			arguments:   map[string]interface{}{
				"name":        "",
				"description": "Updated",
			},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "non-existent effect",
			arguments:   map[string]interface{}{
				"name":        "doesNotExist",
				"description": "Updated",
			},
			expectError: true,
			expectText:  "not found",
		},
		{
			name:        "empty description when provided",
			arguments:   map[string]interface{}{
				"name":        "existingEffect",
				"description": "",
			},
			expectError: true,
			expectText:  "'description' must be a non-empty string",
		},
		{
			name:        "empty pattern when provided",
			arguments:   map[string]interface{}{
				"name":    "existingEffect",
				"pattern": "",
			},
			expectError: true,
			expectText:  "'pattern' must be a non-empty string",
		},
		{
			name:        "invalid duration type",
			arguments:   map[string]interface{}{
				"name":     "existingEffect",
				"duration": "sixty",
			},
			expectError: true,
			expectText:  "'duration' must be a number",
		},
		{
			name:        "duration out of range",
			arguments:   map[string]interface{}{
				"name":     "existingEffect",
				"duration": 4000,
			},
			expectError: true,
			expectText:  "must be between 0 and 3600 seconds",
		},
		{
			name:        "no updates provided",
			arguments:   map[string]interface{}{
				"name": "existingEffect",
			},
			expectError: true,
			expectText:  "No updates provided",
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

func TestUpdateEffectTool_PartialUpdates(t *testing.T) {
	// Test that partial updates preserve non-updated fields
	tmpFile := "/tmp/test-partial-update-effects.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load()

	// Add a test effect with all fields
	originalEffect := &effects.Effect{
		Name:        "testPartial",
		Description: "Original description",
		Pattern:     "top=0|15|FF0000",
		Duration:    60,
	}
	store.Add(originalEffect)
	store.Save()

	tool := NewUpdateEffectTool(store)

	// Update only description
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"name":        "testPartial",
		"description": "New description",
	})

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.IsError {
		t.Error("Execute returned error result")
	}

	// Verify the effect
	updatedEffect, exists := store.Get("testPartial")
	if !exists {
		t.Fatal("Effect not found after update")
	}

	// Check that only description changed
	if updatedEffect.Description != "New description" {
		t.Error("Description was not updated")
	}
	if updatedEffect.Pattern != originalEffect.Pattern {
		t.Error("Pattern should not have changed")
	}
	if updatedEffect.Duration != originalEffect.Duration {
		t.Error("Duration should not have changed")
	}
}