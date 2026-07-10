package workspace

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/docker"
	"github.com/junixlabs/devbox/internal/executor"
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

	// sshFactory overrides how SSH executors are created; nil means use
	// newSSH(). Tests set this to inject a recordingSSH without a live host.
	sshFactory func() (devboxssh.Executor, error)
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

// newSSHExecutor returns the manager's SSH executor factory, or newSSH if
// none was injected.
func (m *remoteManager) newSSHExecutor() (devboxssh.Executor, error) {
	if m.sshFactory != nil {
		return m.sshFactory()
	}
	return newSSH()
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

	sshExec, err := m.newSSHExecutor()
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

	// Build the runtime-neutral config and deploy via the selected executor.
	var res *config.Resources
	if !params.Resources.IsZero() {
		res = &params.Resources
	}
	runtime := params.Runtime
	if runtime == "" {
		runtime = config.RuntimeDocker
	}
	cfg := &config.DevboxConfig{
		Name:           params.Name,
		Server:         params.Server,
		Repo:           params.Repo,
		Branch:         params.Branch,
		Runtime:        runtime,
		Setup:          params.Setup,
		Serve:          params.Serve,
		Services:       params.Services,
		Ports:          params.Ports,
		Env:            params.Env,
		Resources:      res,
		WorkspacesRoot: docker.WorkspaceBaseDir,
	}

	ex, err := executor.New(sshExec, cfg, params.Server, params.Name)
	if err != nil {
		sshExec.Run(ctx, params.Server, fmt.Sprintf("rm -rf %s", wsDir)) //nolint:errcheck
		return nil, fmt.Errorf("creating executor: %w", err)
	}

	if err := ex.Deploy(ctx); err != nil {
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
		Runtime:    runtime,
		Setup:      params.Setup,
		Serve:      params.Serve,
		Ports:      params.Ports,
		Env:        params.Env,
		Services:   params.Services,
		Resources:  params.Resources,
		CreatedAt:  now,
		StartedAt:  &now,
	}
	if pr, ok := ex.(executor.PIDReporter); ok {
		if pid, err := pr.PID(ctx); err == nil {
			ws.ServePID = pid
		}
	}

	if err := m.state.Put(ws); err != nil {
		return nil, fmt.Errorf("saving workspace state: %w", err)
	}

	return ws, nil
}

// newExecutor reconstructs an Executor for an existing workspace from its
// persisted state (runtime, setup, serve).
func (m *remoteManager) newExecutor(sshExec devboxssh.Executor, ws *Workspace) (executor.Executor, error) {
	cfg := &config.DevboxConfig{
		Name:           ws.Name,
		Server:         ws.ServerHost,
		Runtime:        ws.Runtime,
		Setup:          ws.Setup,
		Serve:          ws.Serve,
		Services:       ws.Services,
		Env:            ws.Env,
		WorkspacesRoot: docker.WorkspaceBaseDir,
	}
	return executor.New(sshExec, cfg, ws.ServerHost, ws.Name)
}

func (m *remoteManager) Start(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	if ws.Status == StatusRunning {
		return nil // already running
	}

	sshExec, err := m.newSSHExecutor()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	ex, err := m.newExecutor(sshExec, ws)
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	if err := ex.Up(context.Background()); err != nil {
		return fmt.Errorf("starting workspace: %w", err)
	}

	if pr, ok := ex.(executor.PIDReporter); ok {
		if pid, err := pr.PID(context.Background()); err == nil {
			ws.ServePID = pid
		}
	}

	now := time.Now()
	ws.Status = StatusRunning
	ws.StartedAt = &now
	ws.StoppedAt = nil
	return m.state.Put(ws)
}

// nativeRebuildFiles are exact repo-relative paths whose change requires a
// full rebuild (re-run setup + restart serve) rather than a fast-refresh —
// per the parent epic's locked escalation rule (native deps / app config).
var nativeRebuildFiles = map[string]bool{
	"package.json":      true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":    true,
	"app.json":          true,
	"app.config.js":     true,
	"app.config.ts":     true,
}

// nativeRebuildPrefixes are repo-relative path prefixes whose change also
// requires a rebuild (native platform code / config plugins).
var nativeRebuildPrefixes = []string{"plugins/", "android/", "ios/"}

// needsRebuild reports whether any changed path requires a rebuild rather
// than a fast-refresh: Metro's own file watcher hot-reloads plain JS, but a
// native dependency, lockfile, or platform-code change needs setup re-run
// and a serve restart to take effect.
func needsRebuild(files []string) bool {
	for _, f := range files {
		if nativeRebuildFiles[f] {
			return true
		}
		for _, prefix := range nativeRebuildPrefixes {
			if strings.HasPrefix(f, prefix) {
				return true
			}
		}
	}
	return false
}

// gitFetchCheckoutCmd builds the command that fetches and force-checks-out
// the requested branch in srcDir. It fetches the branch's ref directly
// (rather than relying on a pre-existing local tracking branch) so it works
// even for a branch not present in the original --single-branch clone, then
// hard-resets to it so a dirty/diverged tree doesn't block the checkout.
func gitFetchCheckoutCmd(srcDir, branch string) string {
	return fmt.Sprintf(
		"git -C %s fetch origin %s && git -C %s checkout -B %s FETCH_HEAD && git -C %s reset --hard FETCH_HEAD",
		srcDir, branch, srcDir, branch, srcDir,
	)
}

// gitRevParseHeadCmd captures the current HEAD commit before checkout, so
// Refresh can later diff against it. It always exits 0 (`|| echo ”`) so a
// missing/corrupt repo surfaces as an empty rev rather than a transport
// error — callers must treat an empty rev as "diff unknown, assume rebuild".
func gitRevParseHeadCmd(srcDir string) string {
	return fmt.Sprintf("git -C %s rev-parse HEAD 2>/dev/null || echo ''", srcDir)
}

// gitDiffNamesCmd lists the paths changed between oldRev and the
// post-checkout HEAD, one per line.
func gitDiffNamesCmd(srcDir, oldRev string) string {
	return fmt.Sprintf("git -C %s diff --name-only %s..HEAD", srcDir, oldRev)
}

// Refresh syncs an existing host-runtime workspace to params.Branch in
// place: git fetch+checkout in its srcDir, then either a fast-refresh
// (just ensure the serve process is alive — Metro hot-reloads the checked
// out JS itself) or a rebuild (re-run setup + restart serve), depending on
// whether the diff touches a native/lockfile path. Setup/Serve/Env are read
// fresh from params (i.e. devbox.yaml), not from persisted workspace state,
// so config changes take effect on re-run.
func (m *remoteManager) Refresh(params RefreshParams) (*Workspace, error) {
	ws, err := m.mustGet(params.Name)
	if err != nil {
		return nil, err
	}
	if ws.Runtime != config.RuntimeHost {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("workspace %q is not a host-runtime workspace and cannot be refreshed in place", ws.Name),
			Suggestion: "Refresh only applies to runtime: host workspaces",
		}
	}
	branch := params.Branch
	if branch == "" {
		branch = ws.Branch
	}
	if branch == "" {
		// Mirrors Create's clone default: a workspace created with no
		// explicit branch has ws.Branch == "" persisted (even though its
		// initial clone used "main"), so Refresh must default the same way
		// or every no-branch workspace would fail branch validation here.
		branch = "main"
	}
	if !validBranch.MatchString(branch) {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("invalid branch name %q", branch),
			Suggestion: "Branch names must contain only alphanumeric characters, hyphens, underscores, dots, and slashes",
		}
	}

	sshExec, err := m.newSSHExecutor()
	if err != nil {
		return nil, err
	}
	defer sshExec.Close()

	ctx := context.Background()
	srcDir := docker.WorkspaceBaseDir + "/" + ws.Name + "/src"

	oldRevOut, _, _ := sshExec.Run(ctx, ws.ServerHost, gitRevParseHeadCmd(srcDir))
	oldRev := strings.TrimSpace(oldRevOut)

	if _, stderr, err := sshExec.Run(ctx, ws.ServerHost, gitFetchCheckoutCmd(srcDir, branch)); err != nil {
		return nil, &WorkspaceError{
			Message:    fmt.Sprintf("failed to sync branch %q for workspace %q on %s\nstderr: %s", branch, ws.Name, ws.ServerHost, stderr),
			Suggestion: "Check the branch exists on the remote and SSH access to the server",
			Err:        err,
		}
	}

	// An unknown pre-checkout rev (first-ever refresh against a corrupt/
	// missing repo) or a failed diff means we can't tell what changed —
	// rebuild is the safe superset in that case.
	rebuild := oldRev == ""
	if !rebuild {
		diffOut, _, diffErr := sshExec.Run(ctx, ws.ServerHost, gitDiffNamesCmd(srcDir, oldRev))
		if diffErr != nil {
			rebuild = true
		} else {
			var files []string
			for _, line := range strings.Split(diffOut, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					files = append(files, line)
				}
			}
			rebuild = needsRebuild(files)
		}
	}

	cfg := &config.DevboxConfig{
		Name:           ws.Name,
		Server:         ws.ServerHost,
		Runtime:        config.RuntimeHost,
		Setup:          params.Setup,
		Serve:          params.Serve,
		Env:            params.Env,
		WorkspacesRoot: docker.WorkspaceBaseDir,
	}
	ex, err := executor.New(sshExec, cfg, ws.ServerHost, ws.Name)
	if err != nil {
		return nil, fmt.Errorf("creating executor: %w", err)
	}

	refresher, isRefresher := ex.(executor.Refresher)
	switch {
	case isRefresher && rebuild:
		if err := refresher.RunSetup(ctx); err != nil {
			return nil, fmt.Errorf("re-running setup: %w", err)
		}
		if err := refresher.Restart(ctx); err != nil {
			return nil, fmt.Errorf("restarting serve process: %w", err)
		}
	case isRefresher:
		// Fast-refresh: Metro's own file watcher picks up the checked-out
		// JS changes; just make sure the serve process is still alive.
		if err := ex.Up(ctx); err != nil {
			return nil, fmt.Errorf("ensuring serve process is running: %w", err)
		}
	default:
		// Defensive fallback for an executor that doesn't support in-place
		// refresh (shouldn't happen for a host-runtime workspace).
		if err := ex.Down(ctx); err != nil {
			return nil, fmt.Errorf("stopping workspace: %w", err)
		}
		if err := ex.Up(ctx); err != nil {
			return nil, fmt.Errorf("starting workspace: %w", err)
		}
	}

	now := time.Now()
	ws.Branch = branch
	ws.Setup = params.Setup
	ws.Serve = params.Serve
	ws.Env = params.Env
	ws.Status = StatusRunning
	ws.StartedAt = &now
	ws.StoppedAt = nil
	if pr, ok := ex.(executor.PIDReporter); ok {
		if pid, err := pr.PID(ctx); err == nil {
			ws.ServePID = pid
		}
	}

	if err := m.state.Put(ws); err != nil {
		return nil, fmt.Errorf("saving workspace state: %w", err)
	}
	return ws, nil
}

func (m *remoteManager) Stop(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	if ws.Status == StatusStopped {
		return nil // already stopped
	}

	sshExec, err := m.newSSHExecutor()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	ex, err := m.newExecutor(sshExec, ws)
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	if err := ex.Down(context.Background()); err != nil {
		return fmt.Errorf("stopping workspace: %w", err)
	}

	now := time.Now()
	ws.Status = StatusStopped
	ws.StoppedAt = &now
	ws.ServePID = 0
	return m.state.Put(ws)
}

func (m *remoteManager) Destroy(name string) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	sshExec, err := m.newSSHExecutor()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	ex, err := m.newExecutor(sshExec, ws)
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	if err := ex.Destroy(context.Background()); err != nil {
		return fmt.Errorf("destroying workspace: %w", err)
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

func (m *remoteManager) Logs(name string, follow bool, stdout, stderr io.Writer) error {
	ws, err := m.mustGet(name)
	if err != nil {
		return err
	}

	sshExec, err := m.newSSHExecutor()
	if err != nil {
		return err
	}
	defer sshExec.Close()

	ex, err := m.newExecutor(sshExec, ws)
	if err != nil {
		return fmt.Errorf("creating executor: %w", err)
	}

	return ex.Logs(context.Background(), follow, stdout, stderr)
}

func (m *remoteManager) DockerStats(host string) (map[string]*ResourceUsage, error) {
	sshExec, err := m.newSSHExecutor()
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
	sshExec, err := m.newSSHExecutor()
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
