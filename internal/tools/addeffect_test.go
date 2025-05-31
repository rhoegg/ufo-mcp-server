package tools

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/effects"
)

func TestAddEffectTool_Definition(t *testing.T) {
	store := effects.NewStore("/tmp/test-effects.json")
	tool := NewAddEffectTool(store)
	def := tool.Definition()

	if def.Name != "addEffect" {
		t.Errorf("expected tool name 'addEffect', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check required parameters
	schema := def.InputSchema
	requiredParams := []string{"name", "description", "pattern"}
	
	if len(schema.Required) != len(requiredParams) {
		t.Errorf("expected %d required parameters, got %d", len(requiredParams), len(schema.Required))
	}

	for _, param := range requiredParams {
		found := false
		for _, req := range schema.Required {
			if req == param {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected '%s' to be a required parameter", param)
		}
	}
}

func TestAddEffectTool_Execute_Success(t *testing.T) {
	// Use a temp file for testing
	tmpFile := "/tmp/test-add-effects.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load() // Initialize with seed effects

	tool := NewAddEffectTool(store)

	tests := []struct {
		name      string
		arguments map[string]interface{}
	}{
		{
			name: "add simple effect",
			arguments: map[string]interface{}{
				"name":        "testEffect1",
				"description": "A test effect",
				"pattern":     "top=0|15|FF0000",
			},
		},
		{
			name: "add effect with duration",
			arguments: map[string]interface{}{
				"name":        "testEffect2",
				"description": "Another test effect",
				"pattern":     "bottom_whirl=300&bottom=0|15|00FF00",
				"duration":    60,
			},
		},
		{
			name: "add complex effect",
			arguments: map[string]interface{}{
				"name":        "complexEffect",
				"description": "Complex multi-ring effect",
				"pattern":     "top=0|5|FF0000|10|5|00FF00&bottom_whirl=400|ccw&logo=1",
				"duration":    120,
			},
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
			effectName := tt.arguments["name"].(string)
			if !strings.Contains(textContent.Text, "Successfully added new effect") {
				t.Error("Expected success message")
			}
			if !strings.Contains(textContent.Text, effectName) {
				t.Errorf("Expected effect name '%s' in response", effectName)
			}

			// Verify effect was actually added to store
			addedEffect, exists := store.Get(effectName)
			if !exists {
				t.Errorf("Effect '%s' was not added to store", effectName)
			} else {
				if addedEffect.Description != tt.arguments["description"].(string) {
					t.Error("Effect description mismatch")
				}
				if addedEffect.Pattern != tt.arguments["pattern"].(string) {
					t.Error("Effect pattern mismatch")
				}
			}
		})
	}
}

func TestAddEffectTool_Execute_Errors(t *testing.T) {
	tmpFile := "/tmp/test-add-effects-errors.json"
	defer os.Remove(tmpFile)

	store := effects.NewStore(tmpFile)
	store.Load()

	// Add an existing effect for duplicate test
	store.Add(&effects.Effect{
		Name:        "existingEffect",
		Description: "Already exists",
		Pattern:     "test=1",
		Duration:    10,
	})

	tool := NewAddEffectTool(store)

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name:        "missing name",
			arguments:   map[string]interface{}{
				"description": "Test",
				"pattern":     "test=1",
			},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "empty name",
			arguments:   map[string]interface{}{
				"name":        "",
				"description": "Test",
				"pattern":     "test=1",
			},
			expectError: true,
			expectText:  "'name' parameter is required",
		},
		{
			name:        "invalid name characters",
			arguments:   map[string]interface{}{
				"name":        "test-effect!", // Invalid characters
				"description": "Test",
				"pattern":     "test=1",
			},
			expectError: true,
			expectText:  "must contain only letters, numbers, and underscores",
		},
		{
			name:        "duplicate name",
			arguments:   map[string]interface{}{
				"name":        "existingEffect",
				"description": "Duplicate",
				"pattern":     "test=2",
			},
			expectError: true,
			expectText:  "already exists",
		},
		{
			name:        "missing description",
			arguments:   map[string]interface{}{
				"name":    "testEffect",
				"pattern": "test=1",
			},
			expectError: true,
			expectText:  "'description' parameter is required",
		},
		{
			name:        "missing pattern",
			arguments:   map[string]interface{}{
				"name":        "testEffect",
				"description": "Test",
			},
			expectError: true,
			expectText:  "'pattern' parameter is required",
		},
		{
			name:        "invalid duration type",
			arguments:   map[string]interface{}{
				"name":        "testEffect",
				"description": "Test",
				"pattern":     "test=1",
				"duration":    "sixty", // String instead of number
			},
			expectError: true,
			expectText:  "'duration' must be a number",
		},
		{
			name:        "duration out of range",
			arguments:   map[string]interface{}{
				"name":        "testEffect",
				"description": "Test",
				"pattern":     "test=1",
				"duration":    4000, // > 3600
			},
			expectError: true,
			expectText:  "must be between 0 and 3600 seconds",
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

func TestIsValidEffectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "myEffect", true},
		{"valid with numbers", "effect123", true},
		{"valid with underscores", "my_cool_effect", true},
		{"valid mixed", "Effect_123_Test", true},
		{"invalid with hyphen", "my-effect", false},
		{"invalid with space", "my effect", false},
		{"invalid with special chars", "effect!", false},
		{"invalid with dot", "effect.test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidEffectName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidEffectName(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}