package workspace

import "time"

// Status represents the current state of a workspace.
type Status string

const (
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
	StatusCreating Status = "creating"
	StatusError    Status = "error"
)

// Workspace represents a remote development workspace.
type Workspace struct {
	Name       string            `json:"name"`
	Project    string            `json:"project"`
	Branch     string            `json:"branch"`
	Status     Status            `json:"status"`
	ServerHost string            `json:"server_host"`
	Ports      map[string]int    `json:"ports"`
	Env        map[string]string `json:"env"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	StoppedAt  *time.Time        `json:"stopped_at,omitempty"`
}

// Manager defines the interface for workspace lifecycle management.
type Manager interface {
	// Create provisions a new workspace on the target server.
	Create(name string, project string, branch string) (*Workspace, error)

	// Start starts a stopped workspace.
	Start(name string) error

	// Stop stops a running workspace without destroying it.
	Stop(name string) error

	// Destroy permanently removes a workspace and all its data.
	Destroy(name string) error

	// List returns all workspaces across configured servers.
	List() ([]Workspace, error)

	// Get returns a single workspace by name.
	Get(name string) (*Workspace, error)

	// SSH opens an interactive SSH session into a workspace.
	SSH(name string) error
}
