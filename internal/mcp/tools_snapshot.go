package mcp

import (
	"context"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/snapshot"
)

// snapshotEntry is the JSON response for a single snapshot.
type snapshotEntry struct {
	Name      string `json:"name"`
	Workspace string `json:"workspace"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

func toSnapshotEntry(s *snapshot.Snapshot) snapshotEntry {
	return snapshotEntry{
		Name:      s.Name,
		Workspace: s.Workspace,
		Size:      s.Size,
		CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// handleSnapshotCreate returns a handler for the devbox_snapshot_create tool.
func handleSnapshotCreate(mgr snapshot.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		server := getString(args, "server")
		if server == "" {
			return toolError(ErrInvalidInput, "server is required"), nil
		}
		workspace := getString(args, "workspace")
		if workspace == "" {
			return toolError(ErrInvalidInput, "workspace is required"), nil
		}
		name := getString(args, "name") // optional, auto-generated if empty

		snap, err := mgr.Create(ctx, server, workspace, name)
		if err != nil {
			return toolErrorf(ErrInternal, "snapshot create: %v", err), nil
		}

		return toolSuccess(toSnapshotEntry(snap))
	}
}

// handleSnapshotRestore returns a handler for the devbox_snapshot_restore tool.
func handleSnapshotRestore(mgr snapshot.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		server := getString(args, "server")
		if server == "" {
			return toolError(ErrInvalidInput, "server is required"), nil
		}
		workspace := getString(args, "workspace")
		if workspace == "" {
			return toolError(ErrInvalidInput, "workspace is required"), nil
		}
		name := getString(args, "name")
		if name == "" {
			return toolError(ErrInvalidInput, "name is required"), nil
		}

		if err := mgr.Restore(ctx, server, workspace, name); err != nil {
			return toolErrorf(ErrNotFound, "snapshot restore: %v", err), nil
		}

		return toolSuccess(map[string]any{
			"restored":  true,
			"workspace": workspace,
			"snapshot":  name,
		})
	}
}

// handleSnapshotList returns a handler for the devbox_snapshot_list tool.
func handleSnapshotList(mgr snapshot.Manager) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		server := getString(args, "server")
		if server == "" {
			return toolError(ErrInvalidInput, "server is required"), nil
		}
		workspace := getString(args, "workspace")
		if workspace == "" {
			return toolError(ErrInvalidInput, "workspace is required"), nil
		}

		snapshots, err := mgr.List(ctx, server, workspace)
		if err != nil {
			return toolErrorf(ErrInternal, "snapshot list: %v", err), nil
		}

		entries := make([]snapshotEntry, 0, len(snapshots))
		for i := range snapshots {
			entries = append(entries, toSnapshotEntry(&snapshots[i]))
		}

		return toolSuccess(entries)
	}
}
