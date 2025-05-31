package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/starspace46/ufo-mcp-go/internal/device"
	"github.com/starspace46/ufo-mcp-go/internal/events"
)

// SendRawApiTool implements the sendRawApi MCP tool
type SendRawApiTool struct {
	client      *device.Client
	broadcaster *events.Broadcaster
}

// NewSendRawApiTool creates a new sendRawApi tool instance
func NewSendRawApiTool(client *device.Client, broadcaster *events.Broadcaster) *SendRawApiTool {
	return &SendRawApiTool{
		client:      client,
		broadcaster: broadcaster,
	}
}

// Definition returns the MCP tool definition for sendRawApi
func (t *SendRawApiTool) Definition() mcp.Tool {
	return mcp.Tool{
		Name:        "sendRawApi",
		Description: "Fire a raw query string exactly as typed in UFO web UI. Use this for custom commands or debugging. The query should not include the leading '?' or '/api' path - just the parameter string (e.g., 'effect=rainbow&dim=100').",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Raw query string to send to UFO /api endpoint (without leading ? or /)",
					"examples":    []string{"effect=rainbow", "dim=128", "logo=on", "top_init=1&top=ff0000"},
				},
			},
			Required: []string{"query"},
		},
	}
}

// Execute runs the sendRawApi tool
func (t *SendRawApiTool) Execute(ctx context.Context, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract query parameter
	queryArg, exists := arguments["query"]
	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'query' parameter is required",
				},
			},
			IsError: true,
		}, nil
	}

	query, ok := queryArg.(string)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: 'query' parameter must be a string",
				},
			},
			IsError: true,
		}, nil
	}

	// Basic validation - query should not contain suspicious characters
	if containsSuspiciousChars(query) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: "Error: Query contains potentially unsafe characters",
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the raw query
	result, err := t.client.SendRawQuery(ctx, query)
	if err != nil {
		// Publish the failed execution event
		t.broadcaster.PublishRawExecuted(query, fmt.Sprintf("ERROR: %v", err))

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("UFO communication error: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Publish the successful execution event
	t.broadcaster.PublishRawExecuted(query, result)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Raw API executed successfully.\nQuery: %s\nResponse: %s", query, result),
			},
		},
		IsError: false,
	}, nil
}

// containsSuspiciousChars performs basic validation on the query string
func containsSuspiciousChars(query string) bool {
	// Check for potentially dangerous characters or patterns
	dangerous := []string{
		"<script", "</script", "javascript:", "data:", "vbscript:",
		"../", "..\\", "file://", "ftp://",
		"\x00", // null byte
	}

	queryLower := query
	// Convert to lowercase for case-insensitive checking
	for i, c := range query {
		if c >= 'A' && c <= 'Z' {
			if i == 0 {
				queryLower = string(c+32) + query[1:]
			} else {
				queryLower = query[:i] + string(c+32) + query[i+1:]
			}
		}
	}

	for _, pattern := range dangerous {
		if contains(queryLower, pattern) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}
