package mcp

import (
	"context"
	"fmt"
	"os"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/server"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/workspace"
)

// Deps holds the dependencies for the MCP server.
type Deps struct {
	Manager   workspace.Manager
	Pool      server.Pool
	Collector metrics.Collector
	SSHExec   devboxssh.Executor
}

// NewServer creates an MCP server and registers all tools.
func NewServer(deps Deps, version string) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer("devbox", version)

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
