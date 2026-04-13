package mcp

import (
	"context"
	"sync"

	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// Session tracks the current MCP connection's agent identity.
// Each devbox mcp serve process has exactly one Session.
type Session struct {
	agentID       string
	workspaceName string
	mu            sync.RWMutex
}

// SetAgent records the agent registered in this session.
func (s *Session) SetAgent(id, workspace string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentID = id
	s.workspaceName = workspace
}

// GetAgent returns the current session's agent ID and workspace name.
// Returns empty strings if no agent is registered.
func (s *Session) GetAgent() (id, workspace string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentID, s.workspaceName
}

// HasAgent returns true if an agent is registered in this session.
func (s *Session) HasAgent() bool {
	id, _ := s.GetAgent()
	return id != ""
}

// WithIsolation wraps a workspace tool handler to enforce agent boundaries.
// If no agent is registered in the session, the handler passes through unchanged
// (backwards-compatible with ISS-61 non-agent usage).
// If an agent is registered, only operations on the agent's own workspace are allowed.
func WithIsolation(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), session *Session) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		agentID, agentWorkspace := session.GetAgent()

		// No agent registered — pass through (non-agent mode).
		if agentID == "" {
			return handler(ctx, request)
		}

		// Agent mode: extract the target workspace name and enforce ownership.
		targetName, err := request.RequireString("name")
		if err != nil {
			return toolError(ErrInvalidInput, err.Error()), nil
		}

		if targetName != agentWorkspace {
			return toolError(ErrForbidden, "cannot access workspace owned by another agent"), nil
		}

		return handler(ctx, request)
	}
}

// WithCreateBlock wraps a create handler to block direct workspace creation
// when an agent is registered. Agents must use devbox_agent_register instead.
func WithCreateBlock(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), session *Session) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if session.HasAgent() {
			return toolError(ErrForbidden, "agents cannot create workspaces directly — use devbox_agent_register to get an isolated workspace"), nil
		}
		return handler(ctx, request)
	}
}

// WithListFilter wraps a list handler to only show the agent's own workspace
// when an agent is registered. In non-agent mode, shows all workspaces.
func WithListFilter(handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error), session *Session, mgr workspace.Manager) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !session.HasAgent() {
			return handler(ctx, request)
		}

		// Agent mode: only return the agent's own workspace.
		_, wsName := session.GetAgent()
		ws, err := mgr.Get(wsName)
		if err != nil {
			return toolError(ErrInternal, err.Error()), nil
		}
		if ws == nil {
			return toolSuccess([]workspace.Workspace{}), nil
		}
		return toolSuccess([]workspace.Workspace{*ws}), nil
	}
}
