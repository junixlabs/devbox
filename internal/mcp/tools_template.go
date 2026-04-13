package mcp

import (
	"context"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/registry"
	"github.com/junixlabs/devbox/internal/template"
)

// templateEntry is the JSON response for a single template.
type templateEntry struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Services    []string       `json:"services,omitempty"`
	Ports       map[string]int `json:"ports,omitempty"`
}

// registryEntry is the JSON response for a community registry search result.
type registryEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// handleTemplateList returns a handler for the devbox_template_list tool.
func handleTemplateList(reg *template.LocalRegistry) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		templates, err := reg.List()
		if err != nil {
			return toolErrorf(ErrInternal, "template list: %v", err), nil
		}

		entries := make([]templateEntry, 0, len(templates))
		for _, t := range templates {
			entries = append(entries, templateEntry{
				Name:        t.Name,
				Description: t.Description,
				Services:    t.Services,
				Ports:       t.Ports,
			})
		}

		return toolSuccess(entries)
	}
}

// handleTemplateSearch returns a handler for the devbox_template_search tool.
func handleTemplateSearch(remote *registry.RemoteRegistry) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		query := getString(args, "query")
		if query == "" {
			return toolError(ErrInvalidInput, "query is required"), nil
		}

		results, err := remote.Search(query)
		if err != nil {
			return toolErrorf(ErrInternal, "template search: %v", err), nil
		}

		entries := make([]registryEntry, 0, len(results))
		for _, r := range results {
			entries = append(entries, registryEntry{
				Name:        r.Name,
				Version:     r.Version,
				Description: r.Description,
			})
		}

		return toolSuccess(entries)
	}
}
