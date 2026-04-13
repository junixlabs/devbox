package mcp

import (
	"log/slog"

	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Deps holds dependencies for the MCP server.
type Deps struct {
	Manager  workspace.Manager
	Registry *AgentRegistry
}

// NewServer creates an MCP server with devbox workspace and agent tools registered.
// The session tracks the current connection's agent identity for isolation.
func NewServer(deps Deps, session *Session) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer("devbox", "1.0.0")
	mgr := deps.Manager

	// Workspace tools (ISS-61) — wrapped with isolation when agent is active.
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
		WithCreateBlock(handleCreate(mgr), session),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_list",
			mcp.WithDescription("List all development workspaces"),
			mcp.WithString("user", mcp.Description("Filter by user (all users if omitted)")),
		),
		WithListFilter(handleList(mgr), session, mgr),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_exec",
			mcp.WithDescription("Execute a command in a workspace container"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Workspace name")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Shell command to execute")),
		),
		WithIsolation(handleExec(mgr), session),
	)

	srv.AddTool(
		mcp.NewTool("devbox_workspace_destroy",
			mcp.WithDescription("Permanently destroy a workspace and its data"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Workspace name to destroy")),
		),
		WithIsolation(handleDestroy(mgr), session),
	)

	// Agent tools (ISS-64).
	srv.AddTool(
		mcp.NewTool("devbox_agent_register",
			mcp.WithDescription("Register an AI agent and auto-create an isolated workspace"),
			mcp.WithString("name", mcp.Required(), mcp.Description("Agent name")),
			mcp.WithArray("capabilities", mcp.Description("Agent capabilities"), mcp.WithStringItems()),
			mcp.WithNumber("cpus", mcp.Description("CPU limit for agent workspace (default: 1.0)")),
			mcp.WithString("memory", mcp.Description("Memory limit for agent workspace (default: 1g)")),
		),
		handleAgentRegister(deps.Registry, mgr, session),
	)

	srv.AddTool(
		mcp.NewTool("devbox_agent_list",
			mcp.WithDescription("List all active agents with their workspace assignments"),
		),
		handleAgentList(deps.Registry, mgr),
	)

	srv.AddTool(
		mcp.NewTool("devbox_agent_workspace",
			mcp.WithDescription("Get the current agent's workspace details"),
		),
		handleAgentWorkspace(deps.Registry, session, mgr),
	)

	return srv
}

// Serve starts the MCP server over stdio, blocking until stdin is closed.
// On return (agent disconnect), it cleans up the agent's workspace if one
// was registered during the session.
func Serve(deps Deps) error {
	session := &Session{}
	srv := NewServer(deps, session)

	err := mcpserver.ServeStdio(srv)

	// Cleanup: if an agent was registered, destroy its workspace.
	if agentID, _ := session.GetAgent(); agentID != "" {
		slog.Info("agent disconnected, cleaning up", "agent", agentID)
		if cleanupErr := deps.Registry.Cleanup(agentID, deps.Manager); cleanupErr != nil {
			slog.Warn("agent cleanup failed", "agent", agentID, "error", cleanupErr)
		}
	}

	return err
}
