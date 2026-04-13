package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/workspace"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// handleCreate returns a tool handler that creates a workspace.
func handleCreate(deps Deps) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		name := getString(args, "name")
		if name == "" {
			return toolError(ErrInvalidInput, "name is required"), nil
		}

		serverHost := getString(args, "server")
		repo := getString(args, "repo")
		branch := getString(args, "branch")

		// Parse optional services array.
		var services []string
		if raw, ok := args["services"]; ok {
			if arr, ok := raw.([]any); ok {
				for _, v := range arr {
					if s, ok := v.(string); ok {
						services = append(services, s)
					}
				}
			}
		}

		// Parse optional env object.
		env := make(map[string]string)
		if envRaw, ok := args["env"]; ok {
			if envMap, ok := envRaw.(map[string]any); ok {
				for k, v := range envMap {
					if s, ok := v.(string); ok {
						env[k] = s
					}
				}
			}
		}

		// Auto-select server if not specified.
		if serverHost == "" {
			selected, err := autoSelectServer(ctx, deps.Pool, deps.SSHExec)
			if err != nil {
				return toolErrorf(ErrInternal, "server auto-select failed: %v", err), nil
			}
			serverHost = selected
		}

		params := workspace.CreateParams{
			Name:     name,
			User:     "mcp",
			Server:   serverHost,
			Repo:     repo,
			Branch:   branch,
			Services: services,
			Env:      env,
		}

		ws, err := deps.Manager.Create(params)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(ws)
	}
}

// handleList returns a tool handler that lists workspaces.
func handleList(mgr workspace.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		user := getString(args, "user")

		opts := workspace.ListOptions{
			User: user,
			All:  user == "",
		}
		workspaces, err := mgr.List(opts)
		if err != nil {
			return toolErrorf(ErrInternal, "failed to list workspaces: %v", err), nil
		}

		return toolSuccess(workspaces)
	}
}

// handleExec returns a tool handler that executes a command in a workspace.
func handleExec(mgr workspace.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		name := getString(args, "name")
		if name == "" {
			return toolError(ErrInvalidInput, "name is required"), nil
		}

		command := getString(args, "command")
		if command == "" {
			return toolError(ErrInvalidInput, "command is required"), nil
		}

		result, err := mgr.Exec(name, command)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(result)
	}
}

// handleDestroy returns a tool handler that destroys a workspace.
func handleDestroy(mgr workspace.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		name := getString(args, "name")
		if name == "" {
			return toolError(ErrInvalidInput, "name is required"), nil
		}

		if err := mgr.Destroy(name); err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(map[string]string{"destroyed": name})
	}
}

// mapWorkspaceError converts a workspace error to a structured MCP error.
func mapWorkspaceError(err error) *gomcp.CallToolResult {
	var wsErr *workspace.WorkspaceError
	if errors.As(err, &wsErr) {
		msg := wsErr.Error()
		if wsErr.GetSuggestion() != "" {
			msg += " (hint: " + wsErr.GetSuggestion() + ")"
		}
		return toolError(ErrNotFound, msg)
	}
	return toolError(ErrInternal, err.Error())
}

// autoSelectServer picks the least-loaded server from the pool.
func autoSelectServer(ctx context.Context, pool server.Pool, exec interface{ Close() error }) (string, error) {
	servers, err := pool.List()
	if err != nil {
		return "", fmt.Errorf("listing servers: %w", err)
	}
	if len(servers) == 0 {
		return "", fmt.Errorf("no servers configured — run 'devbox server add' first")
	}

	// Return first server for simplicity; full least-loaded selection uses SSH metrics.
	return server.SSHHost(&servers[0]), nil
}
