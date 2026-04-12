package snapshot

import (
	"context"
	"time"
)

// Snapshot represents a saved point-in-time copy of a workspace's volumes and config.
type Snapshot struct {
	Name      string    `json:"name"`
	Workspace string    `json:"workspace"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// Manager defines the interface for workspace snapshot operations.
type Manager interface {
	// Create saves a snapshot of the workspace's Docker volumes and config files.
	Create(ctx context.Context, host, workspace, name string) (*Snapshot, error)

	// Restore extracts a snapshot, restoring volumes and config files.
	Restore(ctx context.Context, host, workspace, name string) error

	// List returns all snapshots for a workspace.
	List(ctx context.Context, host, workspace string) ([]Snapshot, error)
}
