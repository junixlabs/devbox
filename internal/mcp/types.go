package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// Error code constants for structured MCP tool responses.
const (
	ErrNotFound     = "NOT_FOUND"
	ErrInvalidInput = "INVALID_INPUT"
	ErrInternal     = "INTERNAL"
	ErrNotRunning   = "NOT_RUNNING"
)

// toolError returns an MCP error result with a structured JSON body.
func toolError(code string, msg string) *mcp.CallToolResult {
	body, _ := json.Marshal(map[string]string{
		"error_code": code,
		"message":    msg,
	})
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(body),
			},
		},
	}
}

// toolSuccess returns an MCP success result with the data JSON-marshaled.
func toolSuccess(data any) *mcp.CallToolResult {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return toolError(ErrInternal, fmt.Sprintf("failed to marshal response: %v", err))
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(body),
			},
		},
	}
}
