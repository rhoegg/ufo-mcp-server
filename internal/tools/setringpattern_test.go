package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
	"github.com/starspace46/ufo-mcp-go/internal/state"
)

func TestSetRingPatternTool_Definition(t *testing.T) {
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetRingPatternTool(client, broadcaster, state.NewManager(broadcaster))
	def := tool.Definition()

	if def.Name != "setRingPattern" {
		t.Errorf("expected tool name 'setRingPattern', got %s", def.Name)
	}

	// Check required fields
	schema := def.InputSchema
	if _, exists := schema.Properties["ring"]; !exists {
		t.Error("schema should have 'ring' property")
	}

	if len(schema.Required) == 0 || schema.Required[0] != "ring" {
		t.Error("ring should be required")
	}
}

func TestSetRingPatternTool_Execute_WithSegments(t *testing.T) {
	// Create test server
	var lastQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	os.Setenv("UFO_IP", server.URL[7:])
	defer os.Unsetenv("UFO_IP")

	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetRingPatternTool(client, broadcaster, state.NewManager(broadcaster))

	tests := []struct {
		name           string
		arguments      map[string]interface{}
		expectError    bool
		expectText     string
		expectedQuery  string
	}{
		{
			name: "simple red segment",
			arguments: map[string]interface{}{
				"ring": "top",
				"segments": []interface{}{"0|15|FF0000"},
			},
			expectError:   false,
			expectText:    "Ring pattern applied to top ring successfully",
			expectedQuery: "top_init=1&top=0|15|FF0000",
		},
		{
			name: "multiple segments",
			arguments: map[string]interface{}{
				"ring": "top",
				"segments": []interface{}{"0|5|FF0000", "10|5|00FF00"},
			},
			expectError:   false,
			expectText:    "Ring pattern applied to top ring successfully",
			expectedQuery: "top_init=1&top=0|5|FF0000|10|5|00FF00",
		},
		{
			name: "with background and whirl",
			arguments: map[string]interface{}{
				"ring": "bottom",
				"segments": []interface{}{"0|3|FF0000"},
				"background": "202020",
				"whirlMs": 300,
			},
			expectError:   false,
			expectText:    "Ring pattern applied to bottom ring successfully",
			expectedQuery: "bottom_init=1&bottom=0|3|FF0000&bottom_bg=202020&bottom_whirl=300",
		},
		{
			name: "with morph effect",
			arguments: map[string]interface{}{
				"ring": "top",
				"segments": []interface{}{"0|5|FF0000"},
				"morphSpec": "1000|500",
			},
			expectError:   false,
			expectText:    "Ring pattern applied to top ring successfully",
			expectedQuery: "top_init=1&top=0|5|FF0000&top_morph=1000|500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastQuery = "" // Reset
			
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

			if !contains(content.Text, tt.expectText) {
				t.Errorf("expected text to contain '%s', got '%s'", tt.expectText, content.Text)
			}

			// Check that correct API call was made
			if !tt.expectError && tt.expectedQuery != "" {
				if lastQuery != tt.expectedQuery {
					t.Errorf("expected query '%s', got '%s'", tt.expectedQuery, lastQuery)
				}
			}
		})
	}
}

// CRITICAL TEST: This tests the bug we discovered in manual testing
func TestSetRingPatternTool_Execute_NoSegments_ShouldFail(t *testing.T) {
	// This test captures the issue: when Claude says "set the top ring to red"
	// but doesn't provide segments, our tool currently just sends "top_init=1"
	// which turns the ring OFF instead of red
	
	var lastQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	os.Setenv("UFO_IP", server.URL[7:])
	defer os.Unsetenv("UFO_IP")

	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetRingPatternTool(client, broadcaster, state.NewManager(broadcaster))

	// Test: ring only, no segments (this is what caused the bug)
	arguments := map[string]interface{}{
		"ring": "top",
		// No segments provided - this is the problem case
	}

	_, err := tool.Execute(context.Background(), arguments)
	
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Current implementation generates: "top_init=1" (just clears the ring)
	// This should either:
	// 1. Generate a default pattern like "top_init=1&top=0|15|FFFFFF" 
	// 2. Return an error saying segments are required
	
	t.Logf("Query generated with no segments: %s", lastQuery)
	
	// For now, document the current behavior
	// TODO: Fix this to generate a sensible default or require segments
	if lastQuery != "top_init=1" {
		t.Errorf("current behavior: expected 'top_init=1', got '%s'", lastQuery)
	}
	
	// This test will remind us that sending just "top_init=1" turns the ring OFF
	// The fix should generate something like "top_init=1&top=0|15|FFFFFF" for "set ring to red"
}

func TestSetRingPatternTool_Execute_ValidationErrors(t *testing.T) {
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetRingPatternTool(client, broadcaster, state.NewManager(broadcaster))

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name:        "missing ring parameter",
			arguments:   map[string]interface{}{},
			expectError: true,
			expectText:  "Error: 'ring' parameter is required",
		},
		{
			name: "invalid ring value",
			arguments: map[string]interface{}{
				"ring": "middle",
			},
			expectError: true,
			expectText:  "Error: 'ring' must be either 'top' or 'bottom'",
		},
		{
			name: "invalid segment format",
			arguments: map[string]interface{}{
				"ring": "top",
				"segments": []interface{}{"invalid"},
			},
			expectError: true,
			expectText:  "Error: invalid segment format",
		},
		{
			name: "invalid background color",
			arguments: map[string]interface{}{
				"ring": "top",
				"background": "invalid",
			},
			expectError: true,
			expectText:  "Error: 'background' must be a valid hex color",
		},
		{
			name: "whirl out of range",
			arguments: map[string]interface{}{
				"ring": "top",
				"whirlMs": 600,
			},
			expectError: true,
			expectText:  "Error: 'whirlMs' must be between 0 and 510",
		},
		{
			name: "invalid morph spec",
			arguments: map[string]interface{}{
				"ring": "top",
				"morphSpec": "invalid",
			},
			expectError: true,
			expectText:  "Error: 'morphSpec' must be in format 'STAY|SPEED'",
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

			if !contains(content.Text, tt.expectText) {
				t.Errorf("expected text to contain '%s', got '%s'", tt.expectText, content.Text)
			}
		})
	}
}

func TestSetRingPatternTool_HelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func() bool
		expected bool
	}{
		{"valid segment format", func() bool { return isValidSegmentFormat("0|5|FF0000") }, true},
		{"invalid segment format - wrong parts", func() bool { return isValidSegmentFormat("0|5") }, false},
		{"invalid segment format - bad color", func() bool { return isValidSegmentFormat("0|5|GGGGGG") }, false},
		{"valid hex color", func() bool { return isValidHexColor("FF0000") }, true},
		{"invalid hex color - too short", func() bool { return isValidHexColor("FF00") }, false},
		{"invalid hex color - bad chars", func() bool { return isValidHexColor("GGGGGG") }, false},
		{"valid morph spec", func() bool { return isValidMorphSpec("1000|500") }, true},
		{"invalid morph spec", func() bool { return isValidMorphSpec("1000") }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildRingPatternCommand(t *testing.T) {
	tests := []struct {
		name       string
		ring       string
		segments   []string
		background string
		whirlMs    int
		morphSpec  string
		expected   string
	}{
		{
			name:     "basic ring init only",
			ring:     "top",
			expected: "top_init=1",
		},
		{
			name:     "ring with single segment",
			ring:     "top",
			segments: []string{"0|15|FF0000"},
			expected: "top_init=1&top=0|15|FF0000",
		},
		{
			name:       "ring with multiple segments",
			ring:       "bottom",
			segments:   []string{"0|5|FF0000", "10|5|00FF00"},
			expected:   "bottom_init=1&bottom=0|5|FF0000|10|5|00FF00",
		},
		{
			name:       "full options",
			ring:       "top",
			segments:   []string{"0|5|FF0000"},
			background: "202020",
			whirlMs:    300,
			morphSpec:  "1000|500",
			expected:   "top_init=1&top=0|5|FF0000&top_bg=202020&top_whirl=300&top_morph=1000|500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRingPatternCommand(tt.ring, tt.segments, tt.background, tt.whirlMs, tt.morphSpec)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}