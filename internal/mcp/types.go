package mcp

import (
	"encoding/json"
	"fmt"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// Error codes for MCP tool responses.
const (
	ErrNotFound     = "not_found"
	ErrInvalidInput = "invalid_input"
	ErrInternal     = "internal_error"
	ErrOffline      = "server_offline"
)

// toolError returns an MCP error result with structured error info.
func toolError(code, msg string) *gomcp.CallToolResult {
	data, _ := json.Marshal(map[string]string{
		"error_code": code,
		"message":    msg,
	})
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			gomcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
		IsError: true,
	}
}

// toolErrorf returns a formatted MCP error result.
func toolErrorf(code, format string, args ...any) *gomcp.CallToolResult {
	return toolError(code, fmt.Sprintf(format, args...))
}

// toolSuccess returns an MCP success result with JSON-encoded data.
func toolSuccess(data any) (*gomcp.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling tool result: %w", err)
	}
	return &gomcp.CallToolResult{
		Content: []gomcp.Content{
			gomcp.TextContent{
				Type: "text",
				Text: string(b),
			},
		},
	}, nil
}

// getString extracts an optional string argument from the request.
func getString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
