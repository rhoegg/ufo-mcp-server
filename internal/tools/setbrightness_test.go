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

func TestSetBrightnessTool_Definition(t *testing.T) {
	client := device.NewClient()
	broadcaster := events.NewBroadcaster()
	defer broadcaster.Close()

	tool := NewSetBrightnessTool(client, broadcaster)
	def := tool.Definition()

	if def.Name != "setBrightness" {
		t.Errorf("expected tool name 'setBrightness', got %s", def.Name)
	}

	if def.Description == "" {
		t.Error("tool description should not be empty")
	}

	// Check schema has required fields
	schema := def.InputSchema
	if _, exists := schema.Properties["level"]; !exists {
		t.Error("schema should have 'level' property")
	}

	if len(schema.Required) == 0 || schema.Required[0] != "level" {
		t.Error("level should be required")
	}

	// Check level constraints
	levelProperty := schema.Properties["level"].(map[string]interface{})
	if levelProperty["type"] != "integer" {
		t.Error("level should be integer type")
	}
	if levelProperty["minimum"] != 0 {
		t.Error("level minimum should be 0")
	}
	if levelProperty["maximum"] != 255 {
		t.Error("level maximum should be 255")
	}
}

func TestSetBrightnessTool_Execute(t *testing.T) {
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

	tool := NewSetBrightnessTool(client, broadcaster)

	tests := []struct {
		name           string
		arguments      map[string]interface{}
		expectError    bool
		expectText     string
		expectedQuery  string
	}{
		{
			name: "valid brightness integer",
			arguments: map[string]interface{}{
				"level": 128,
			},
			expectError:   false,
			expectText:    "Brightness set to 128/255 (50%)",
			expectedQuery: "dim=128",
		},
		{
			name: "valid brightness float64",
			arguments: map[string]interface{}{
				"level": 128.0,
			},
			expectError:   false,
			expectText:    "Brightness set to 128/255 (50%)",
			expectedQuery: "dim=128",
		},
		{
			name: "minimum brightness",
			arguments: map[string]interface{}{
				"level": 0,
			},
			expectError:   false,
			expectText:    "Brightness set to 0/255 (0%)",
			expectedQuery: "dim=0",
		},
		{
			name: "maximum brightness",
			arguments: map[string]interface{}{
				"level": 255,
			},
			expectError:   false,
			expectText:    "Brightness set to 255/255 (100%)",
			expectedQuery: "dim=255",
		},
		{
			name:        "missing level parameter",
			arguments:   map[string]interface{}{},
			expectError: true,
			expectText:  "Error: 'level' parameter is required",
		},
		{
			name: "invalid level type",
			arguments: map[string]interface{}{
				"level": "invalid",
			},
			expectError: true,
			expectText:  "Error: 'level' parameter must be a number",
		},
		{
			name: "level too low",
			arguments: map[string]interface{}{
				"level": -1,
			},
			expectError: true,
			expectText:  "Error: brightness level must be between 0 and 255",
		},
		{
			name: "level too high",
			arguments: map[string]interface{}{
				"level": 256,
			},
			expectError: true,
			expectText:  "Error: brightness level must be between 0 and 255",
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

func TestSetBrightnessTool_EventPublishing(t *testing.T) {
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
	tool := NewSetBrightnessTool(client, broadcaster)

	arguments := map[string]interface{}{
		"level": 128,
	}

	_, err := tool.Execute(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that events were published
	eventsReceived := 0
	expectedEvents := 2 // raw_executed and dim_changed

	for eventsReceived < expectedEvents {
		select {
		case event := <-sub.Channel:
			eventsReceived++
			
			if event.Type == events.EventRawExecuted {
				query, ok := event.Data["query"].(string)
				if !ok || query != "dim=128" {
					t.Errorf("expected query 'dim=128', got %v", query)
				}
			} else if event.Type == events.EventDimChanged {
				level, ok := event.Data["level"].(int)
				if !ok || level != 128 {
					t.Errorf("expected level 128, got %v", level)
				}
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("timeout waiting for events, received %d/%d", eventsReceived, expectedEvents)
			return
		}
	}
}