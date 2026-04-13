package mcp

import (
	"context"
	"fmt"
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/registry"
	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/snapshot"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/template"
	"github.com/junixlabs/devbox/internal/workspace"
)

// Deps holds the dependencies for the MCP server.
type Deps struct {
	Manager     workspace.Manager
	Pool        server.Pool
	Collector   metrics.Collector
	SSHExec     devboxssh.Executor
	SnapshotMgr snapshot.Manager
	TemplateReg *template.LocalRegistry
	RemoteReg   *registry.RemoteRegistry
}

// NewServer creates an MCP server and registers all tools.
func NewServer(deps Deps, version string) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer("devbox", version)

	// Workspace lifecycle tools.
	srv.AddTool(
		gomcp.NewTool("devbox_workspace_create",
			gomcp.WithDescription("Create a new development workspace on a remote server"),
			gomcp.WithString("name", gomcp.Required(), gomcp.Description("Workspace name")),
			gomcp.WithString("server", gomcp.Description("Server host (auto-selected if omitted)")),
			gomcp.WithString("template", gomcp.Description("Template name to use")),
			gomcp.WithString("repo", gomcp.Description("Git repository URL to clone")),
			gomcp.WithString("branch", gomcp.Description("Git branch to checkout")),
			gomcp.WithArray("services", gomcp.Description("Docker services to run"), gomcp.WithStringItems()),
			gomcp.WithObject("env", gomcp.Description("Environment variables")),
		),
		handleCreate(deps),
	)

	srv.AddTool(
		gomcp.NewTool("devbox_workspace_list",
			gomcp.WithDescription("List all development workspaces"),
			gomcp.WithString("user", gomcp.Description("Filter by user (all users if omitted)")),
		),
		handleList(deps.Manager),
	)

	srv.AddTool(
		gomcp.NewTool("devbox_workspace_exec",
			gomcp.WithDescription("Execute a command in a workspace container"),
			gomcp.WithString("name", gomcp.Required(), gomcp.Description("Workspace name")),
			gomcp.WithString("command", gomcp.Required(), gomcp.Description("Shell command to execute")),
		),
		handleExec(deps.Manager),
	)

	srv.AddTool(
		gomcp.NewTool("devbox_workspace_destroy",
			gomcp.WithDescription("Permanently destroy a workspace and its data"),
			gomcp.WithString("name", gomcp.Required(), gomcp.Description("Workspace name to destroy")),
		),
		handleDestroy(deps.Manager),
	)

	// Server pool tools.
	srv.AddTool(
		gomcp.NewTool("devbox_server_list",
			gomcp.WithDescription("List all servers in the pool with health status and available resources"),
		),
		handleServerList(deps.Pool, deps.SSHExec),
	)

	srv.AddTool(
		gomcp.NewTool("devbox_server_status",
			gomcp.WithDescription("Get detailed status for a single server including resource usage and workspace breakdown"),
			gomcp.WithString("name", gomcp.Required(), gomcp.Description("Server name")),
		),
		handleServerStatus(deps.Pool, deps.Collector, deps.SSHExec),
	)

	// Metrics tool.
	srv.AddTool(
		gomcp.NewTool("devbox_metrics",
			gomcp.WithDescription("Get resource metrics (CPU, memory, disk, network I/O) for a workspace or server"),
			gomcp.WithString("workspace", gomcp.Description("Workspace name — returns per-workspace metrics")),
			gomcp.WithString("server", gomcp.Description("Server name or host — returns server-level metrics with workspace breakdown")),
		),
		handleMetrics(deps.Manager, deps.Pool, deps.Collector),
	)

	// Snapshot tools.
	if deps.SnapshotMgr != nil {
		srv.AddTool(
			gomcp.NewTool("devbox_snapshot_create",
				gomcp.WithDescription("Save a snapshot of a workspace's Docker volumes and config files"),
				gomcp.WithString("server", gomcp.Required(), gomcp.Description("Server host where the workspace runs")),
				gomcp.WithString("workspace", gomcp.Required(), gomcp.Description("Workspace name")),
				gomcp.WithString("name", gomcp.Description("Snapshot name (auto-generated if omitted)")),
			),
			handleSnapshotCreate(deps.SnapshotMgr),
		)

		srv.AddTool(
			gomcp.NewTool("devbox_snapshot_restore",
				gomcp.WithDescription("Restore a workspace from a previously saved snapshot"),
				gomcp.WithString("server", gomcp.Required(), gomcp.Description("Server host where the workspace runs")),
				gomcp.WithString("workspace", gomcp.Required(), gomcp.Description("Workspace name")),
				gomcp.WithString("name", gomcp.Required(), gomcp.Description("Snapshot name to restore")),
			),
			handleSnapshotRestore(deps.SnapshotMgr),
		)

		srv.AddTool(
			gomcp.NewTool("devbox_snapshot_list",
				gomcp.WithDescription("List all snapshots for a workspace"),
				gomcp.WithString("server", gomcp.Required(), gomcp.Description("Server host where the workspace runs")),
				gomcp.WithString("workspace", gomcp.Required(), gomcp.Description("Workspace name")),
			),
			handleSnapshotList(deps.SnapshotMgr),
		)
	}

	// Template tools.
	if deps.TemplateReg != nil {
		srv.AddTool(
			gomcp.NewTool("devbox_template_list",
				gomcp.WithDescription("List available workspace templates (built-in and custom)"),
			),
			handleTemplateList(deps.TemplateReg),
		)
	}

	if deps.RemoteReg != nil {
		srv.AddTool(
			gomcp.NewTool("devbox_template_search",
				gomcp.WithDescription("Search the community template registry"),
				gomcp.WithString("query", gomcp.Required(), gomcp.Description("Search query (matches template name and description)")),
			),
			handleTemplateSearch(deps.RemoteReg),
		)
	}

	return srv
}

// Serve starts the MCP server over stdio. Blocks until stdin closes.
func Serve(deps Deps, version string) error {
	srv := NewServer(deps, version)
	stdio := mcpserver.NewStdioServer(srv)
	return stdio.Listen(context.Background(), os.Stdin, os.Stdout)
}

// resolveServerHost looks up a server by name in the pool and returns its SSH host.
func resolveServerHost(pool server.Pool, name string) (*server.Server, error) {
	servers, err := pool.List()
	if err != nil {
		return nil, fmt.Errorf("listing servers: %w", err)
	}
	for i := range servers {
		if servers[i].Name == name {
			return &servers[i], nil
		}
	}
	return nil, fmt.Errorf("server %q not found", name)
}
