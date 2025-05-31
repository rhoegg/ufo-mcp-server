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

func TestSetLogoTool_Definition(t *testing.T) {
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetLogoTool(client, broadcaster)
	def := tool.Definition()

	if def.Name != "setLogo" {
		t.Errorf("expected tool name 'setLogo', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check schema has required fields
	schema := def.InputSchema
	if _, exists := schema.Properties["state"]; !exists {
		t.Error("schema should have 'state' property")
	}

	if len(schema.Required) == 0 || schema.Required[0] != "state" {
		t.Error("state should be required")
	}

	// Check enum values
	stateProperty := schema.Properties["state"].(map[string]interface{})
	enum := stateProperty["enum"].([]string)
	if len(enum) != 2 || enum[0] != "on" || enum[1] != "off" {
		t.Errorf("expected enum [on, off], got %v", enum)
	}
}

func TestSetLogoTool_Execute(t *testing.T) {
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

	tool := NewSetLogoTool(client, broadcaster)

	tests := []struct {
		name           string
		arguments      map[string]interface{}
		expectError    bool
		expectText     string
		expectedQuery  string
	}{
		{
			name: "valid logo on",
			arguments: map[string]interface{}{
				"state": "on",
			},
			expectError:   false,
			expectText:    "Logo LED turned on successfully",
			expectedQuery: "logo=on",
		},
		{
			name: "valid logo off",
			arguments: map[string]interface{}{
				"state": "off",
			},
			expectError:   false,
			expectText:    "Logo LED turned off successfully",
			expectedQuery: "logo=off",
		},
		{
			name:        "missing state parameter",
			arguments:   map[string]interface{}{},
			expectError: true,
			expectText:  "Error: 'state' parameter is required",
		},
		{
			name: "invalid state type",
			arguments: map[string]interface{}{
				"state": 123,
			},
			expectError: true,
			expectText:  "Error: 'state' parameter must be a string",
		},
		{
			name: "invalid state value",
			arguments: map[string]interface{}{
				"state": "invalid",
			},
			expectError: true,
			expectText:  "Error: 'state' must be either 'on' or 'off'",
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

			// Check that correct API call was made (for successful cases)
			if !tt.expectError && tt.expectedQuery != "" {
				if lastQuery != tt.expectedQuery {
					t.Errorf("expected query '%s', got '%s'", tt.expectedQuery, lastQuery)
				}
			}
		})
	}
}

func TestSetLogoTool_EventPublishing(t *testing.T) {
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

	sub := broadcaster.Subscribe("test")
	tool := NewSetLogoTool(client, broadcaster)

	arguments := map[string]interface{}{
		"state": "on",
	}

	_, err := tool.Execute(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that an event was published
	select {
	case event := <-sub.Channel:
		if event.Type != events.EventRawExecuted {
			t.Errorf("expected event type %s, got %s", events.EventRawExecuted, event.Type)
		}
		
		query, ok := event.Data["query"].(string)
		if !ok || query != "logo=on" {
			t.Errorf("expected query 'logo=on', got %v", query)
		}
		
		result, ok := event.Data["result"].(string)
		if !ok || result != "OK" {
			t.Errorf("expected result 'OK', got %v", result)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event to be published")
	}
}