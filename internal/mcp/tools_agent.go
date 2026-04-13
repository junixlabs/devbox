package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// defaultAgentCPUs is the default CPU limit for agent workspaces.
const defaultAgentCPUs = 1.0

// defaultAgentMemory is the default memory limit for agent workspaces.
const defaultAgentMemory = "1g"

// handleAgentRegister returns a tool handler that registers an agent,
// auto-creates an isolated workspace, and binds it to the current session.
func handleAgentRegister(reg *AgentRegistry, mgr workspace.Manager, session *Session) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Prevent double registration in a single session.
		if session.HasAgent() {
			return toolError(ErrInvalidInput, "agent already registered in this session"), nil
		}

		name, err := request.RequireString("name")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		// Validate agent name before using in workspace/container names.
		if err := ValidateAgentName(name); err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		// Parse optional capabilities.
		capabilities := request.GetStringSlice("capabilities", nil)

		// Parse optional resource limits with defaults.
		cpus := request.GetFloat("cpus", defaultAgentCPUs)
		memory := request.GetString("memory", defaultAgentMemory)

		// Generate unique agent ID.
		agentID, err := GenerateAgentID(name)
		if err != nil {
			return toolError(ErrInternal, fmt.Sprintf("generating agent ID: %v", err)), nil
		}

		// Auto-select server.
		serverHost, err := autoSelectServer()
		if err != nil {
			return toolError(ErrInternal, fmt.Sprintf("server auto-select failed: %v", err)), nil
		}

		// Create isolated workspace for this agent.
		ws, err := mgr.Create(workspace.CreateParams{
			Name:   agentID,
			User:   "agent-" + name,
			Server: serverHost,
			Resources: config.Resources{
				CPUs:   cpus,
				Memory: memory,
			},
		})
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		// Register agent in shared registry.
		agent := &Agent{
			ID:            agentID,
			Name:          name,
			Capabilities:  capabilities,
			WorkspaceName: agentID,
			ServerHost:    ws.ServerHost,
			PID:           os.Getpid(),
			RegisteredAt:  time.Now(),
		}
		if err := reg.Register(agent); err != nil {
			// Rollback: destroy workspace if registration fails.
			mgr.Destroy(agentID) //nolint:errcheck
			return toolError(ErrInternal, fmt.Sprintf("registering agent: %v", err)), nil
		}

		// Bind agent to this session for isolation enforcement.
		session.SetAgent(agentID, agentID)

		return toolSuccess(map[string]any{
			"agent_id":  agentID,
			"workspace": agentID,
			"server":    ws.ServerHost,
			"resources": map[string]any{
				"cpus":   cpus,
				"memory": memory,
			},
		}), nil
	}
}

// handleAgentList returns a tool handler that lists all active agents.
// Stale sessions (dead PIDs) are pruned before listing.
func handleAgentList(reg *AgentRegistry, mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Prune stale sessions from crashed processes.
		_ = reg.PruneStale(mgr)

		agents, err := reg.List()
		if err != nil {
			return toolError(ErrInternal, fmt.Sprintf("listing agents: %v", err)), nil
		}

		return toolSuccess(agents), nil
	}
}

// handleAgentWorkspace returns a tool handler that returns the calling agent's
// workspace details. Only works if an agent is registered in the current session.
func handleAgentWorkspace(reg *AgentRegistry, session *Session, mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID, wsName := session.GetAgent()
		if agentID == "" {
			return toolError(ErrInvalidInput, "no agent registered in this session — call devbox_agent_register first"), nil
		}

		agent, err := reg.Get(agentID)
		if err != nil {
			return toolError(ErrInternal, fmt.Sprintf("looking up agent: %v", err)), nil
		}
		if agent == nil {
			return toolError(ErrNotFound, "agent no longer registered"), nil
		}

		ws, err := mgr.Get(wsName)
		if err != nil {
			return mapWorkspaceError(err), nil
		}

		return toolSuccess(map[string]any{
			"agent":     agent,
			"workspace": ws,
		}), nil
	}
}
