package mcp

import (
	"encoding/json"
	"testing"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// extractText extracts the text content from a CallToolResult.
func extractText(t *testing.T, result *gomcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(gomcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestToolSuccess(t *testing.T) {
	tests := []struct {
		name string
		data any
	}{
		{"string", "hello"},
		{"struct", struct {
			Name string `json:"name"`
		}{"test"}},
		{"slice", []string{"a", "b"}},
		{"map", map[string]int{"x": 1}},
		{"nil", nil},
		{"empty_slice", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolSuccess(tt.data)
			if err != nil {
				t.Fatalf("toolSuccess() error: %v", err)
			}
			if result.IsError {
				t.Error("toolSuccess() returned IsError=true")
			}
			text := extractText(t, result)
			if !json.Valid([]byte(text)) {
				t.Errorf("toolSuccess() content is not valid JSON: %s", text)
			}
		})
	}
}

func TestToolError(t *testing.T) {
	tests := []struct {
		code string
		msg  string
	}{
		{ErrNotFound, "workspace not found"},
		{ErrInvalidInput, "name is required"},
		{ErrInternal, "unexpected error"},
		{ErrOffline, "server is offline"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := toolError(tt.code, tt.msg)
			if !result.IsError {
				t.Error("toolError() returned IsError=false")
			}
			text := extractText(t, result)
			var body map[string]string
			if err := json.Unmarshal([]byte(text), &body); err != nil {
				t.Fatalf("toolError() content is not valid JSON: %v", err)
			}
			if body["error_code"] != tt.code {
				t.Errorf("error_code = %q, want %q", body["error_code"], tt.code)
			}
			if body["message"] != tt.msg {
				t.Errorf("message = %q, want %q", body["message"], tt.msg)
			}
		})
	}
}

func TestToolErrorf(t *testing.T) {
	result := toolErrorf(ErrNotFound, "workspace %q not found", "test-ws")
	if !result.IsError {
		t.Error("toolErrorf() returned IsError=false")
	}
	text := extractText(t, result)
	var body map[string]string
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("content is not valid JSON: %v", err)
	}
	want := `workspace "test-ws" not found`
	if body["message"] != want {
		t.Errorf("message = %q, want %q", body["message"], want)
	}
}

func TestGetString(t *testing.T) {
	args := map[string]any{
		"name":  "test",
		"count": 42,
		"empty": "",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"name", "test"},
		{"count", ""},
		{"empty", ""},
		{"missing", ""},
	}

	for _, tt := range tests {
		got := getString(args, tt.key)
		if got != tt.want {
			t.Errorf("getString(args, %q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestNewServer(t *testing.T) {
	deps := Deps{}
	srv := NewServer(deps, "1.0.0-test")
	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}
}
