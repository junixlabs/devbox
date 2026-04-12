package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

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
	User       string            `json:"user"`
	Project    string            `json:"project"`
	Branch     string            `json:"branch"`
	Status     Status            `json:"status"`
	ServerHost string            `json:"server_host"`
	Repo       string            `json:"repo"`
	Ports      map[string]int    `json:"ports"`
	Env        map[string]string `json:"env"`
	Services   []string          `json:"services"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	StoppedAt  *time.Time        `json:"stopped_at,omitempty"`
}

// CreateParams bundles the inputs needed to create a workspace.
type CreateParams struct {
	Name     string
	User     string
	Server   string
	Repo     string
	Branch   string
	Services []string
	Ports    map[string]int
	Env      map[string]string
}

// ListOptions controls workspace list filtering.
type ListOptions struct {
	User string // Filter workspaces by user. Empty string shows all.
	All  bool   // When true, show all workspaces regardless of user filter.
}

// Manager defines the interface for workspace lifecycle management.
type Manager interface {
	// Create provisions a new workspace on the target server.
	Create(params CreateParams) (*Workspace, error)

	// Start starts a stopped workspace.
	Start(name string) error

	// Stop stops a running workspace without destroying it.
	Stop(name string) error

	// Destroy permanently removes a workspace and all its data.
	Destroy(name string) error

	// List returns workspaces, optionally filtered by user.
	List(opts ListOptions) ([]Workspace, error)

	// Get returns a single workspace by name.
	Get(name string) (*Workspace, error)

	// SSH opens an interactive SSH session into a workspace.
	SSH(name string) error
}

// WorkspaceError represents a workspace-related error with a suggestion.
type WorkspaceError struct {
	Message    string
	Suggestion string
	Err        error
}

func (e *WorkspaceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *WorkspaceError) Unwrap() error        { return e.Err }
func (e *WorkspaceError) GetSuggestion() string { return e.Suggestion }

// FormatName builds a workspace name from user, project, and branch.
// Returns {user}-{project}-{branch} with sanitization (lowercase, no slashes).
func FormatName(user, project, branch string) string {
	sanitize := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "/", "-")
		return s
	}
	parts := []string{sanitize(user), sanitize(project)}
	if branch != "" {
		parts = append(parts, sanitize(branch))
	}
	return strings.Join(parts, "-")
}

// FormatPath builds a workspace filesystem path.
// Returns {root}/{user}/{project}-{branch}/.
func FormatPath(root, user, project, branch string) string {
	dir := strings.ToLower(project)
	if branch != "" {
		dir += "-" + strings.ToLower(strings.ReplaceAll(branch, "/", "-"))
	}
	return filepath.Join(root, strings.ToLower(user), dir)
}
