package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/junixlabs/devbox/internal/workspace"
)

// validAgentName matches safe agent names: alphanumeric start, then alphanumeric/hyphens/underscores.
var validAgentName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// Agent represents a registered AI agent session.
type Agent struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Capabilities  []string  `json:"capabilities,omitempty"`
	WorkspaceName string    `json:"workspace_name"`
	ServerHost    string    `json:"server_host"`
	PID           int       `json:"pid"`
	RegisteredAt  time.Time `json:"registered_at"`
}

// agentFile is the on-disk JSON structure for the agent registry.
type agentFile struct {
	Agents map[string]*Agent `json:"agents"`
}

// AgentRegistry manages agent sessions in a shared JSON file.
// Thread-safe within a single process; uses file locking for cross-process safety.
type AgentRegistry struct {
	path string
	mu   sync.Mutex
}

// NewAgentRegistry creates an AgentRegistry at ~/.devbox/agents.json.
func NewAgentRegistry() (*AgentRegistry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}
	dir := filepath.Join(home, ".devbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating state directory %s: %w", dir, err)
	}
	return &AgentRegistry{path: filepath.Join(dir, "agents.json")}, nil
}

// load reads agents from disk. Caller must hold both mu and file lock.
func (r *AgentRegistry) load() (map[string]*Agent, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Agent), nil
		}
		return nil, fmt.Errorf("reading agent registry: %w", err)
	}

	var af agentFile
	if err := json.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("parsing agent registry: %w", err)
	}
	if af.Agents == nil {
		return make(map[string]*Agent), nil
	}
	return af.Agents, nil
}

// save writes agents to disk atomically. Caller must hold both mu and file lock.
func (r *AgentRegistry) save(agents map[string]*Agent) error {
	af := agentFile{Agents: agents}
	data, err := json.MarshalIndent(af, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling agent registry: %w", err)
	}

	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing agent registry: %w", err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming agent registry: %w", err)
	}
	return nil
}

// withFileLock runs fn while holding an exclusive flock on the registry file.
// This provides cross-process safety for concurrent devbox mcp serve processes.
func (r *AgentRegistry) withFileLock(fn func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	lockPath := r.path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquiring file lock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN) //nolint:errcheck

	return fn()
}

// Register adds an agent to the registry.
func (r *AgentRegistry) Register(agent *Agent) error {
	return r.withFileLock(func() error {
		agents, err := r.load()
		if err != nil {
			return err
		}
		if _, exists := agents[agent.ID]; exists {
			return fmt.Errorf("agent %q already registered", agent.ID)
		}
		agents[agent.ID] = agent
		return r.save(agents)
	})
}

// Deregister removes an agent from the registry.
func (r *AgentRegistry) Deregister(id string) error {
	return r.withFileLock(func() error {
		agents, err := r.load()
		if err != nil {
			return err
		}
		delete(agents, id)
		return r.save(agents)
	})
}

// Get returns an agent by ID, or nil if not found.
func (r *AgentRegistry) Get(id string) (*Agent, error) {
	var agent *Agent
	err := r.withFileLock(func() error {
		agents, loadErr := r.load()
		if loadErr != nil {
			return loadErr
		}
		agent = agents[id]
		return nil
	})
	return agent, err
}

// List returns all registered agents.
func (r *AgentRegistry) List() ([]*Agent, error) {
	var result []*Agent
	err := r.withFileLock(func() error {
		agents, loadErr := r.load()
		if loadErr != nil {
			return loadErr
		}
		result = make([]*Agent, 0, len(agents))
		for _, a := range agents {
			result = append(result, a)
		}
		return nil
	})
	return result, err
}

// Cleanup destroys the agent's workspace and removes it from the registry.
// Errors from workspace destruction are logged but not returned — the agent
// is deregistered regardless (workspace may already be gone).
func (r *AgentRegistry) Cleanup(id string, mgr workspace.Manager) error {
	// Atomically look up and deregister the agent under one file lock
	// to prevent TOCTOU races with concurrent processes.
	var wsName string
	err := r.withFileLock(func() error {
		agents, loadErr := r.load()
		if loadErr != nil {
			return loadErr
		}
		agent, exists := agents[id]
		if !exists {
			return nil // already gone
		}
		wsName = agent.WorkspaceName
		delete(agents, id)
		return r.save(agents)
	})
	if err != nil {
		return fmt.Errorf("deregistering agent for cleanup: %w", err)
	}

	// Destroy workspace outside the lock (may take time over SSH).
	if wsName != "" {
		if err := mgr.Destroy(wsName); err != nil {
			slog.Warn("failed to destroy agent workspace during cleanup",
				"agent", id, "workspace", wsName, "error", err)
		}
	}

	return nil
}

// PruneStale removes agents whose process (PID) is no longer running.
// This handles crash recovery — if a devbox mcp serve process was killed
// with SIGKILL, its cleanup never ran.
func (r *AgentRegistry) PruneStale(mgr workspace.Manager) error {
	// Collect stale agent IDs and deregister atomically under file lock.
	var staleAgents []struct {
		id        string
		workspace string
	}
	err := r.withFileLock(func() error {
		agents, loadErr := r.load()
		if loadErr != nil {
			return loadErr
		}

		changed := false
		for id, agent := range agents {
			if !isProcessAlive(agent.PID) {
				slog.Info("pruning stale agent session", "agent", id)
				staleAgents = append(staleAgents, struct {
					id        string
					workspace string
				}{id: id, workspace: agent.WorkspaceName})
				delete(agents, id)
				changed = true
			}
		}
		if changed {
			return r.save(agents)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("pruning stale agents: %w", err)
	}

	// Destroy workspaces outside the lock (may take time over SSH).
	for _, stale := range staleAgents {
		if err := mgr.Destroy(stale.workspace); err != nil {
			slog.Warn("failed to destroy stale agent workspace",
				"agent", stale.id, "workspace", stale.workspace, "error", err)
		}
	}
	return nil
}

// isProcessAlive checks if a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't send a signal but checks if the process exists.
	return proc.Signal(syscall.Signal(0)) == nil
}

// ValidateAgentName checks that an agent name is safe for use in workspace names.
func ValidateAgentName(name string) error {
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("agent name too long (max 64 characters)")
	}
	if !validAgentName.MatchString(name) {
		return fmt.Errorf("agent name %q is invalid: must start with alphanumeric, then alphanumeric/hyphens/underscores", name)
	}
	return nil
}

// GenerateAgentID creates a unique agent ID: agent-{name}-{hex8}.
func GenerateAgentID(name string) (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return fmt.Sprintf("agent-%s-%s", name, hex.EncodeToString(b)), nil
}
