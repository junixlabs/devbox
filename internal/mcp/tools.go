package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/server"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleCreate returns a tool handler that creates a workspace.
func handleCreate(mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		serverHost := request.GetString("server", "")
		template := request.GetString("template", "")
		repo := request.GetString("repo", "")
		branch := request.GetString("branch", "")

		// Parse optional services array.
		services := request.GetStringSlice("services", nil)

		// Parse optional env object.
		env := make(map[string]string)
		if args := request.GetArguments(); args != nil {
			if envRaw, ok := args["env"]; ok {
				if envMap, ok := envRaw.(map[string]any); ok {
					for k, v := range envMap {
						if s, ok := v.(string); ok {
							env[k] = s
						}
					}
				}
			}
		}

		// Auto-select server if not specified.
		if serverHost == "" {
			selected, err := autoSelectServer()
			if err != nil {
				return toolError(ErrInternal, fmt.Sprintf("server auto-select failed: %v", err)), nil
			}
			serverHost = selected
		}

		_ = template // TODO: template support in future issue

		params := workspace.CreateParams{
			Name:     name,
			User:     "mcp",
			Server:   serverHost,
			Repo:     repo,
			Branch:   branch,
			Services: services,
			Env:      env,
		}

		ws, err := mgr.Create(params)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(ws), nil
	}
}

// handleList returns a tool handler that lists workspaces.
func handleList(mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		user := request.GetString("user", "")

		opts := workspace.ListOptions{
			User: user,
			All:  user == "",
		}
		workspaces, err := mgr.List(opts)
		if err != nil {
			return toolError(ErrInternal, fmt.Sprintf("failed to list workspaces: %v", err)), nil
		}

		return toolSuccess(workspaces), nil
	}
}

// handleExec returns a tool handler that executes a command in a workspace.
func handleExec(mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		command, err := request.RequireString("command")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		result, err := mgr.Exec(name, command)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(result), nil
	}
}

// handleDestroy returns a tool handler that destroys a workspace.
func handleDestroy(mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		if err := mgr.Destroy(name); err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(map[string]string{"destroyed": name}), nil
	}
}

// mapWorkspaceError converts a workspace error to a structured MCP error.
func mapWorkspaceError(err error) *mcp.CallToolResult {
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
func autoSelectServer() (string, error) {
	sshExec, err := devboxssh.New()
	if err != nil {
		return "", fmt.Errorf("creating SSH executor: %w", err)
	}
	defer sshExec.Close()

	configPath, err := server.DefaultConfigPath()
	if err != nil {
		return "", fmt.Errorf("getting server config path: %w", err)
	}

	pool, err := server.NewFilePool(configPath, sshExec)
	if err != nil {
		return "", fmt.Errorf("loading server pool: %w", err)
	}

	servers, err := pool.List()
	if err != nil {
		return "", fmt.Errorf("listing servers: %w", err)
	}
	if len(servers) == 0 {
		return "", fmt.Errorf("no servers configured — run 'devbox server add' first")
	}

	selector := server.NewLeastLoaded(sshExec)
	selected, err := selector.Select(context.Background(), servers)
	if err != nil {
		return "", err
	}

	return server.SSHHost(selected), nil
}

// resourcesFromArgs extracts optional cpus/memory resource limits from request args.
func resourcesFromArgs(request mcp.CallToolRequest) config.Resources {
	return config.Resources{
		CPUs:   request.GetFloat("cpus", 0),
		Memory: request.GetString("memory", ""),
	}
}
