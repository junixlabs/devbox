package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/docker"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

// validName matches safe identifiers for workspace names.
var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// validBranch matches safe git branch names (no shell metacharacters).
var validBranch = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

// remoteManager implements Manager by orchestrating SSH and Docker Compose
// operations on a remote server, with local state persistence.
type remoteManager struct {
	state *stateStore
}

// NewManager creates a workspace Manager with local state at ~/.devbox/state.json.
func NewManager() Manager {
	state, err := newStateStore()
	if err != nil {
		// State dir creation failed — fall back to a temp path so the binary
		// still starts (errors will surface on first state operation).
		slog.Warn("failed to initialize state store, using temp dir", "error", err)
		state = newStateStoreAt(os.TempDir() + "/devbox-state.json")
	}
	return &remoteManager{state: state}
}

// newSSH creates a fresh SSH executor. Caller must Close() it.
func newSSH() (devboxssh.Executor, error) {
	sshExec, err := devboxssh.New()
	if err != nil {
		return nil, &WorkspaceError{
			Message:    "failed to create SSH executor",
			Suggestion: "Check that ssh is installed: which ssh",
			Err:        err,
		}
	}
	return sshExec, nil
}

func (m *remoteManager) Create(params CreateParams) (*Workspace, error) {
	// Validate inputs before using them in shell commands.
	if !validName.MatchString(params.Name) {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("invalid workspace name %q", params.Name),
			Suggestion: "Names must start with alphanumeric, then alphanumeric/hyphens/underscores/dots",
		}
	}
	if params.Branch != "" && !validBranch.MatchString(params.Branch) {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("invalid branch name %q", params.Branch),
			Suggestion: "Branch names must contain only alphanumeric characters, hyphens, underscores, dots, and slashes",
		}
	}

	// Check if workspace already exists.
	existing, err := m.state.Get(params.Name)
	if err != nil {
		return nil, fmt.Errorf("checking existing workspace: %w", err)
	}
	if existing != nil {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("workspace %q already exists", params.Name),
			Suggestion: fmt.Sprintf("Use 'devbox destroy %s' first, or choose a different name", params.Name),
		}
	}

	sshExec, err := newSSH()
	if err != nil {
		return nil, err
	}
	defer sshExec.Close()

	ctx := context.Background()
	wsDir := docker.WorkspaceBaseDir + "/" + params.Name

	// Clone repo if specified.
	if params.Repo != "" {
		branch := params.Branch
		if branch == "" {
			branch = "main"
		}
		cloneCmd := fmt.Sprintf("git clone --branch %s --single-branch -- %s %s/src",
			branch, params.Repo, wsDir)
		slog.Debug("cloning repo", "command", cloneCmd)
		if _, _, err := sshExec.Run(ctx, params.Server, cloneCmd); err != nil {
			// Clean up partial workspace on clone failure.
			sshExec.Run(ctx, params.Server, fmt.Sprintf("rm -rf %s", wsDir)) //nolint:errcheck
			return nil, &WorkspaceError{
				Message:    fmt.Sprintf("failed to clone repo on %s", params.Server),
				Suggestion: "Check repo URL and SSH key forwarding to the server",
				Err:        err,
			}
		}
	}

	// Generate compose YAML and deploy via docker manager.
	var res *config.Resources
	if !params.Resources.IsZero() {
		res = &params.Resources
	}
	cfg := &config.DevboxConfig{
		Name:      params.Name,
		Server:    params.Server,
		Repo:      params.Repo,
		Branch:    params.Branch,
		Services:  params.Services,
		Ports:     params.Ports,
		Env:       params.Env,
		Resources: res,
	}
	composeYAML, err := docker.GenerateCompose(params.Name, cfg)
	if err != nil {
		sshExec.Run(ctx, params.Server, fmt.Sprintf("rm -rf %s", wsDir)) //nolint:errcheck
		return nil, fmt.Errorf("generating compose file: %w", err)
	}

	dockerMgr, err := docker.NewManager(sshExec, params.Server, params.Name)
	if err != nil {
		sshExec.Run(ctx, params.Server, fmt.Sprintf("rm -rf %s", wsDir)) //nolint:errcheck
		return nil, fmt.Errorf("creating docker manager: %w", err)
	}

	if err := dockerMgr.Deploy(ctx, composeYAML); err != nil {
		sshExec.Run(ctx, params.Server, fmt.Sprintf("rm -rf %s", wsDir)) //nolint:errcheck
		return nil, fmt.Errorf("deploying workspace: %w", err)
	}

	now := time.Now()
	ws := &Workspace{
		Name:       params.Name,
		User:       params.User,
		Project:    params.Name,
		Branch:     params.Branch,
		Status:     StatusRunning,
		ServerHost: params.Server,
		Repo:       params.Repo,
		Ports:      params.Ports,
		Env:        params.Env,
		Services:   params.Services,
		Resources:  params.Resources,
		CreatedAt:  now,
		StartedAt:  &now,
	}

	if err := m.state.Put(ws); err != nil {
		return nil, fmt.Errorf("saving workspace state: %w", err)
	}

	return ws, nil
}

func (m *remoteManager) Start(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	if ws.Status == StatusRunning {
		return nil // already running
	}

	sshExec, err := newSSH()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	dockerMgr, err := docker.NewManager(sshExec, ws.ServerHost, ws.Name)
	if err != nil {
		return fmt.Errorf("creating docker manager: %w", err)
	}

	if err := dockerMgr.Up(context.Background()); err != nil {
		return fmt.Errorf("starting workspace containers: %w", err)
	}

	now := time.Now()
	ws.Status = StatusRunning
	ws.StartedAt = &now
	ws.StoppedAt = nil
	return m.state.Put(ws)
}

func (m *remoteManager) Stop(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	if ws.Status == StatusStopped {
		return nil // already stopped
	}

	sshExec, err := newSSH()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	dockerMgr, err := docker.NewManager(sshExec, ws.ServerHost, ws.Name)
	if err != nil {
		return fmt.Errorf("creating docker manager: %w", err)
	}

	if err := dockerMgr.Down(context.Background()); err != nil {
		return fmt.Errorf("stopping workspace containers: %w", err)
	}

	now := time.Now()
	ws.Status = StatusStopped
	ws.StoppedAt = &now
	return m.state.Put(ws)
}

func (m *remoteManager) Destroy(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	sshExec, err := newSSH()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	dockerMgr, err := docker.NewManager(sshExec, ws.ServerHost, ws.Name)
	if err != nil {
		return fmt.Errorf("creating docker manager: %w", err)
	}

	if err := dockerMgr.Destroy(context.Background()); err != nil {
		return fmt.Errorf("destroying workspace containers: %w", err)
	}

	return m.state.Delete(name)
}

func (m *remoteManager) List(opts ListOptions) ([]Workspace, error) {
	workspaces, err := m.state.Load()
	if err != nil {
		return nil, fmt.Errorf("loading workspace state: %w", err)
	}

	result := make([]Workspace, 0, len(workspaces))
	for _, ws := range workspaces {
		if !opts.All && opts.User != "" && ws.User != opts.User {
			continue
		}
		result = append(result, *ws)
	}
	return result, nil
}

func (m *remoteManager) Get(name string) (*Workspace, error) {
	return m.mustGet(name)
}

func (m *remoteManager) SSH(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	if ws.Status != StatusRunning {
		return &WorkspaceError{
			Message:    fmt.Sprintf("workspace %q is not running (status: %s)", name, ws.Status),
			Suggestion: fmt.Sprintf("Start it first: devbox up or devbox ssh %s (auto-starts)", name),
		}
	}

	// Use ssh -t to allocate a TTY, then docker exec into the first service container.
	containerName := ws.Name + "-" + firstService(ws.Services) + "-1"
	sshCmd := fmt.Sprintf("docker exec -it %s /bin/sh", containerName)
	slog.Debug("ssh into container", "host", ws.ServerHost, "container", containerName)

	cmd := exec.Command("ssh", "-t", ws.ServerHost, sshCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return &WorkspaceError{
			Message:    fmt.Sprintf("SSH session to %q failed", name),
			Suggestion: fmt.Sprintf("Check that the container is running: ssh %s docker ps", ws.ServerHost),
			Err:        err,
		}
	}
	return nil
}

func (m *remoteManager) Exec(name string, command string) (*ExecResult, error) {
	if command == "" {
		return nil, &WorkspaceError{
			Message:    "command must not be empty",
			Suggestion: "Provide a command to execute, e.g. devbox exec myws 'ls -la'",
		}
	}

	ws, err := m.mustGet(name)
	if err != nil {
		return nil, err
	}

	if ws.Status != StatusRunning {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("workspace %q is not running (status: %s)", name, ws.Status),
			Suggestion: fmt.Sprintf("Start it first: devbox up %s", name),
		}
	}

	sshExec, err := newSSH()
	if err != nil {
		return nil, err
	}
	defer sshExec.Close()

	containerName := ws.Name + "-" + firstService(ws.Services) + "-1"
	// Escape single quotes in the command for safe shell injection prevention.
	escaped := strings.ReplaceAll(command, "'", "'\\''")
	dockerCmd := fmt.Sprintf("docker exec %s sh -c '%s'", containerName, escaped)

	slog.Debug("exec in container", "host", ws.ServerHost, "container", containerName, "command", command)

	stdout, stderr, execErr := sshExec.Run(context.Background(), ws.ServerHost, dockerCmd)
	result := &ExecResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: 0,
	}

	if execErr != nil {
		// Try to extract exit code from the error.
		var exitErr *exec.ExitError
		if errors.As(execErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result, nil
}

func (m *remoteManager) DockerStats(host string) (map[string]*ResourceUsage, error) {
	sshExec, err := newSSH()
	if err != nil {
		return nil, err
	}
	defer sshExec.Close()

	cmd := `docker stats --no-stream --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"`
	stdout, _, err := sshExec.Run(context.Background(), host, cmd)
	if err != nil {
		return nil, fmt.Errorf("running docker stats on %s: %w", host, err)
	}
	return ParseDockerStats(stdout)
}

func (m *remoteManager) ServerResources(host string) (*ServerResourceInfo, error) {
	sshExec, err := newSSH()
	if err != nil {
		return nil, err
	}
	defer sshExec.Close()

	ctx := context.Background()
	combined, _, err := sshExec.Run(ctx, host, "nproc && echo '---SEPARATOR---' && cat /proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("querying server resources on %s: %w", host, err)
	}
	parts := strings.SplitN(combined, "---SEPARATOR---", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected server resources output from %s", host)
	}
	return ParseServerResources(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
}

// mustGet returns a workspace by name, or a WorkspaceError if not found.
func (m *remoteManager) mustGet(name string) (*Workspace, error) {
	ws, err := m.state.Get(name)
	if err != nil {
		return nil, fmt.Errorf("reading workspace state: %w", err)
	}
	if ws == nil {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("workspace %q not found", name),
			Suggestion: "Run 'devbox list' to see available workspaces",
		}
	}
	return ws, nil
}

// firstService returns the base name of the first service, or "app" as default.
func firstService(services []string) string {
	if len(services) == 0 {
		return "app"
	}
	// "mysql:8.0" → "mysql"
	svc := services[0]
	if i := strings.LastIndex(svc, ":"); i != -1 {
		svc = svc[:i]
	}
	if i := strings.LastIndex(svc, "/"); i != -1 {
		svc = svc[i+1:]
	}
	return svc
}
