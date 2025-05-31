package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
)

func TestSendRawApiTool_Definition(t *testing.T) {
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSendRawApiTool(client, broadcaster)
	def := tool.Definition()

	if def.Name != "sendRawApi" {
		t.Errorf("expected tool name 'sendRawApi', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check that input schema has required fields
	schema := def.InputSchema

	if _, exists := schema.Properties["query"]; !exists {
		t.Error("schema should have 'query' property")
	}

	if len(schema.Required) == 0 || schema.Required[0] != "query" {
		t.Error("query should be required")
	}
}

func TestSendRawApiTool_Execute(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("UFO Response: " + query))
	}))
	defer server.Close()

	// Set UFO_IP to test server
	os.Setenv("UFO_IP", server.URL[7:]) // Remove "http://" prefix
	defer os.Unsetenv("UFO_IP")

	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSendRawApiTool(client, broadcaster)

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		expectText  string
	}{
		{
			name: "valid query",
			arguments: map[string]interface{}{
				"query": "effect=rainbow",
			},
			expectError: false,
			expectText:  "Raw API executed successfully",
		},
		{
			name:        "missing query parameter",
			arguments:   map[string]interface{}{},
			expectError: true,
			expectText:  "Error: 'query' parameter is required",
		},
		{
			name: "invalid query type",
			arguments: map[string]interface{}{
				"query": 123,
			},
			expectError: true,
			expectText:  "Error: 'query' parameter must be a string",
		},
		{
			name: "suspicious query",
			arguments: map[string]interface{}{
				"query": "<script>alert('xss')</script>",
			},
			expectError: true,
			expectText:  "Error: Query contains potentially unsafe characters",
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

			text := content.Text

			if !contains(text, tt.expectText) {
				t.Errorf("expected text to contain '%s', got '%s'", tt.expectText, text)
			}
		})
	}
}

func TestSendRawApiTool_EventPublishing(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	os.Setenv("UFO_IP", server.URL[7:])
	defer os.Unsetenv("UFO_IP")

	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	// Subscribe to events
	sub := broadcaster.Subscribe("test")

	tool := NewSendRawApiTool(client, broadcaster)

	// Execute a successful query
	arguments := map[string]interface{}{
		"query": "effect=test",
	}

	_, err := tool.Execute(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that an event was published (with timeout)
	select {
	case event := <-sub.Channel:
		if event.Type != events.EventRawExecuted {
			t.Errorf("expected event type %s, got %s", events.EventRawExecuted, event.Type)
		}

		query, ok := event.Data["query"].(string)
		if !ok || query != "effect=test" {
			t.Errorf("expected query 'effect=test', got %v", query)
		}

		result, ok := event.Data["result"].(string)
		if !ok || result != "OK" {
			t.Errorf("expected result 'OK', got %v", result)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event to be published")
	}
}

func TestContainsSuspiciousChars(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"effect=rainbow", false},
		{"dim=100", false},
		{"top_init=1&top=ff0000", false},
		{"<script>alert('xss')</script>", true},
		{"javascript:alert(1)", true},
		{"../etc/passwd", true},
		{"file://test", true},
		{"normal=query", false},
		{"EFFECT=RAINBOW", false}, // Should handle uppercase
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsSuspiciousChars(tt.input)
			if result != tt.expected {
				t.Errorf("containsSuspiciousChars(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
