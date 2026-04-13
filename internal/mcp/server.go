package mcp

import (
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewServer creates an MCP server with devbox workspace tools registered.
func NewServer(mgr workspace.Manager) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer("devbox", "1.0.0")

	srv.AddTool(
		mcp.NewTool("devbox_workspace_create",
			mcp.WithDescription("Create a new development workspace on a remote server"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Workspace name")),
			mcp.WithString("server", mcp.Description("Server host (auto-selected if omitted)")),
			mcp.WithString("template", mcp.Description("Template name to use")),
			mcp.WithString("repo", mcp.Description("Git repository URL to clone")),
			mcp.WithString("branch", mcp.Description("Git branch to checkout")),
			mcp.WithArray("services", mcp.Description("Docker services to run"), mcp.WithStringItems()),
			mcp.WithObject("env", mcp.Description("Environment variables")),
		),
		handleCreate(mgr),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_list",
			mcp.WithDescription("List all development workspaces"),
			mcp.WithString("user", mcp.Description("Filter by user (all users if omitted)")),
		),
		handleList(mgr),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_exec",
			mcp.WithDescription("Execute a command in a workspace container"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Workspace name")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Shell command to execute")),
		),
		handleExec(mgr),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_destroy",
			mcp.WithDescription("Permanently destroy a workspace and its data"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Workspace name to destroy")),
		),
		handleDestroy(mgr),
	)

	return srv
}

// Serve starts the MCP server over stdio, blocking until stdin is closed.
func Serve(mgr workspace.Manager) error {
	srv := NewServer(mgr)
	return mcpserver.ServeStdio(srv)
}
