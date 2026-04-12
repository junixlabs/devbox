package workspace

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/config"
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
	Resources  config.Resources  `json:"resources"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	StoppedAt  *time.Time        `json:"stopped_at,omitempty"`
}

// CreateParams bundles the inputs needed to create a workspace.
type CreateParams struct {
	Name      string
	User      string
	Server    string
	Repo      string
	Branch    string
	Services  []string
	Ports     map[string]int
	Env       map[string]string
	Resources config.Resources
}

// ListOptions controls filtering for workspace listing.
type ListOptions struct {
	User string // filter by user; empty means no filter
	All  bool   // if true, show all users' workspaces
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

	// List returns all workspaces across configured servers.
	List(opts ListOptions) ([]Workspace, error)

	// Get returns a single workspace by name.
	Get(name string) (*Workspace, error)

	// SSH opens an interactive SSH session into a workspace.
	SSH(name string) error

	// DockerStats returns live resource usage for all containers on a host.
	DockerStats(host string) (map[string]*ResourceUsage, error)

	// ServerResources returns total CPU and memory for a host.
	ServerResources(host string) (*ServerResourceInfo, error)
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

// FormatName returns a workspace name in the format {user}-{project}-{branch}.
// If branch is empty, returns {user}-{project}.
func FormatName(user, project, branch string) string {
	parts := []string{sanitizePart(user), sanitizePart(project)}
	if branch != "" {
		parts = append(parts, sanitizePart(branch))
	}
	return strings.Join(parts, "-")
}

// FormatPath returns the filesystem path for a workspace: {root}/{user}/{project}-{branch}/.
func FormatPath(root, user, project, branch string) string {
	dirName := sanitizePart(project)
	if branch != "" {
		dirName += "-" + sanitizePart(branch)
	}
	return filepath.Join(root, sanitizePart(user), dirName)
}

// sanitizePart normalizes a name part: lowercase, replace slashes with hyphens.
func sanitizePart(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "/", "-")
	return s
}
