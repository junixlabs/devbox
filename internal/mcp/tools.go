package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/junixlabs/devbox/internal/server"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/tailscale"
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

		// Look up workspace before destroy to clean up Tailscale serve ports.
		ws, err := mgr.Get(name)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		unservePorts(ws)

		if err := mgr.Destroy(name); err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(map[string]string{"destroyed": name}), nil
	}
}

// mapWorkspaceError converts a workspace error to a structured MCP error
// with an appropriate error code based on the error message.
func mapWorkspaceError(err error) *mcp.CallToolResult {
	var wsErr *workspace.WorkspaceError
	if errors.As(err, &wsErr) {
		msg := wsErr.Error()
		if wsErr.GetSuggestion() != "" {
			msg += " (hint: " + wsErr.GetSuggestion() + ")"
		}
		code := classifyWorkspaceError(wsErr)
		return toolError(code, msg)
	}
	return toolError(ErrInternal, err.Error())
}

// classifyWorkspaceError maps a WorkspaceError to the appropriate MCP error code.
func classifyWorkspaceError(wsErr *workspace.WorkspaceError) string {
	msg := wsErr.Message
	switch {
	case strings.Contains(msg, "not running"):
		return ErrNotRunning
	case strings.Contains(msg, "not found"):
		return ErrNotFound
	case strings.Contains(msg, "invalid"),
		strings.Contains(msg, "already exists"),
		strings.Contains(msg, "must not be empty"):
		return ErrInvalidInput
	default:
		return ErrInternal
	}
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

// unservePorts tears down Tailscale serve entries for all workspace ports.
// Errors are logged as warnings but do not stop the operation.
func unservePorts(ws *workspace.Workspace) {
	if len(ws.Ports) == 0 {
		return
	}

	sshExec, err := devboxssh.New()
	if err != nil {
		slog.Warn("failed to connect for port cleanup", "error", err)
		return
	}
	defer sshExec.Close()

	runner := func(command string, args ...string) ([]byte, error) {
		parts := make([]string, 0, len(args)+1)
		parts = append(parts, command)
		parts = append(parts, args...)
		stdout, _, err := sshExec.Run(context.Background(), ws.ServerHost, strings.Join(parts, " "))
		return []byte(stdout), err
	}

	tm := tailscale.NewManager(runner)
	for _, port := range ws.Ports {
		if err := tm.Unserve(port); err != nil {
			slog.Warn("failed to unserve port", "port", port, "error", err)
		}
	}
}
