package workspace

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

// recordingSSH implements ssh.Executor, recording every command it is asked
// to run so tests can assert on what a reconstructed executor sends. When
// runFunc is set, it scripts the (stdout, stderr, err) per command — used to
// simulate git output (rev-parse/diff) and PID-file reads without a live host.
type recordingSSH struct {
	calls   []string
	runFunc func(command string) (string, string, error)
}

func (r *recordingSSH) Run(_ context.Context, _ string, command string) (string, string, error) {
	r.calls = append(r.calls, command)
	if r.runFunc != nil {
		return r.runFunc(command)
	}
	return "", "", nil
}

func (r *recordingSSH) RunStream(_ context.Context, _ string, command string, _, _ io.Writer) error {
	r.calls = append(r.calls, command)
	return nil
}

func (r *recordingSSH) CopyTo(context.Context, string, string, string) error   { return nil }
func (r *recordingSSH) CopyFrom(context.Context, string, string, string) error { return nil }
func (r *recordingSSH) Close() error                                           { return nil }

// --- State store tests ---

func tempStateStore(t *testing.T) *stateStore {
	t.Helper()
	dir := t.TempDir()
	return newStateStoreAt(filepath.Join(dir, "state.json"))
}

func TestStateStore_LoadEmpty(t *testing.T) {
	s := tempStateStore(t)
	ws, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(ws) != 0 {
		t.Errorf("expected empty map, got %d entries", len(ws))
	}
}

func TestStateStore_PutAndGet(t *testing.T) {
	s := tempStateStore(t)
	now := time.Now()

	ws := &Workspace{
		Name:       "test-ws",
		Project:    "test-ws",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		CreatedAt:  now,
	}
	if err := s.Put(ws); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	got, err := s.Get("test-ws")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.Name != "test-ws" {
		t.Errorf("Name = %q, want %q", got.Name, "test-ws")
	}
	if got.Status != StatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, StatusRunning)
	}
}

func TestStateStore_GetNotFound(t *testing.T) {
	s := tempStateStore(t)
	got, err := s.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for nonexistent workspace, got %+v", got)
	}
}

func TestStateStore_Delete(t *testing.T) {
	s := tempStateStore(t)
	ws := &Workspace{
		Name:       "to-delete",
		Project:    "to-delete",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	}
	if err := s.Put(ws); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	if err := s.Delete("to-delete"); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	got, err := s.Get("to-delete")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after delete, got %+v", got)
	}
}

func TestStateStore_LoadMultiple(t *testing.T) {
	s := tempStateStore(t)
	now := time.Now()

	for _, name := range []string{"ws-1", "ws-2", "ws-3"} {
		if err := s.Put(&Workspace{
			Name:       name,
			Project:    name,
			Status:     StatusRunning,
			ServerHost: "devbox-vps",
			CreatedAt:  now,
		}); err != nil {
			t.Fatalf("Put(%s) error: %v", name, err)
		}
	}

	all, err := s.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Load() returned %d workspaces, want 3", len(all))
	}
}

func TestStateStore_AtomicWrite(t *testing.T) {
	s := tempStateStore(t)
	ws := &Workspace{
		Name:       "atomic",
		Project:    "atomic",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	}
	if err := s.Put(ws); err != nil {
		t.Fatalf("Put() error: %v", err)
	}

	// Verify no .tmp file left behind.
	tmpPath := s.path + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %s should not exist after successful write", tmpPath)
	}
}

// --- Manager helper tests ---

func TestFirstService(t *testing.T) {
	tests := []struct {
		services []string
		want     string
	}{
		{nil, "app"},
		{[]string{}, "app"},
		{[]string{"mysql:8.0"}, "mysql"},
		{[]string{"redis:7-alpine", "mysql:8.0"}, "redis"},
		{[]string{"bitnami/redis:7"}, "redis"},
		{[]string{"postgres"}, "postgres"},
	}
	for _, tt := range tests {
		got := firstService(tt.services)
		if got != tt.want {
			t.Errorf("firstService(%v) = %q, want %q", tt.services, got, tt.want)
		}
	}
}

// --- Manager tests with state ---

func testManager(t *testing.T) *remoteManager {
	t.Helper()
	return &remoteManager{state: tempStateStore(t)}
}

func TestManager_GetNotFound(t *testing.T) {
	mgr := testManager(t)
	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workspace")
	}
	wsErr, ok := err.(*WorkspaceError)
	if !ok {
		t.Fatalf("expected *WorkspaceError, got %T", err)
	}
	if wsErr.Suggestion == "" {
		t.Error("expected suggestion in error")
	}
}

func TestManager_ListEmpty(t *testing.T) {
	mgr := testManager(t)
	workspaces, err := mgr.List(ListOptions{All: true})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(workspaces) != 0 {
		t.Errorf("expected empty list, got %d", len(workspaces))
	}
}

func TestManager_ListWithWorkspaces(t *testing.T) {
	mgr := testManager(t)
	now := time.Now()
	// Pre-populate state.
	for _, name := range []string{"ws-a", "ws-b"} {
		mgr.state.Put(&Workspace{
			Name:       name,
			Project:    name,
			Status:     StatusRunning,
			ServerHost: "devbox-vps",
			CreatedAt:  now,
		})
	}

	workspaces, err := mgr.List(ListOptions{All: true})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(workspaces) != 2 {
		t.Errorf("List() returned %d, want 2", len(workspaces))
	}
}

func TestManager_CreateInvalidName(t *testing.T) {
	mgr := testManager(t)
	badNames := []string{"bad name", "bad;name", "../escape", " leading-space"}
	for _, name := range badNames {
		_, err := mgr.Create(CreateParams{Name: name, Server: "devbox-vps"})
		if err == nil {
			t.Errorf("expected error for name %q, got nil", name)
			continue
		}
		wsErr, ok := err.(*WorkspaceError)
		if !ok {
			t.Errorf("expected *WorkspaceError for name %q, got %T", name, err)
			continue
		}
		if wsErr.Suggestion == "" {
			t.Errorf("expected suggestion for name %q", name)
		}
	}
}

func TestManager_CreateInvalidBranch(t *testing.T) {
	mgr := testManager(t)
	_, err := mgr.Create(CreateParams{
		Name:   "valid-name",
		Server: "devbox-vps",
		Branch: "branch; rm -rf /",
	})
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
	wsErr, ok := err.(*WorkspaceError)
	if !ok {
		t.Fatalf("expected *WorkspaceError, got %T", err)
	}
	if wsErr.Suggestion == "" {
		t.Error("expected suggestion in branch error")
	}
}

func TestManager_CreateValidBranch(t *testing.T) {
	mgr := testManager(t)
	// Valid branch names should pass validation (will fail at SSH step, but that's ok).
	validBranches := []string{"main", "feature/auth", "fix/bug-123", "release/v1.0.0"}
	for _, branch := range validBranches {
		_, err := mgr.Create(CreateParams{
			Name:   "test-" + strings.ReplaceAll(branch, "/", "-"),
			Server: "devbox-vps",
			Branch: branch,
		})
		// Should NOT get a validation error — it will fail at SSH, which is expected.
		if err != nil {
			wsErr, ok := err.(*WorkspaceError)
			if ok && strings.Contains(wsErr.Message, "invalid branch") {
				t.Errorf("branch %q should be valid but got: %v", branch, err)
			}
		}
	}
}

func TestManager_CreateDuplicate(t *testing.T) {
	mgr := testManager(t)
	mgr.state.Put(&Workspace{
		Name:       "existing",
		Project:    "existing",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	})

	_, err := mgr.Create(CreateParams{Name: "existing", Server: "devbox-vps"})
	if err == nil {
		t.Fatal("expected error for duplicate workspace")
	}
	wsErr, ok := err.(*WorkspaceError)
	if !ok {
		t.Fatalf("expected *WorkspaceError, got %T", err)
	}
	if wsErr.Suggestion == "" {
		t.Error("expected suggestion in duplicate error")
	}
}

func TestManager_SSHNotRunning(t *testing.T) {
	mgr := testManager(t)
	mgr.state.Put(&Workspace{
		Name:       "stopped-ws",
		Project:    "stopped-ws",
		Status:     StatusStopped,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	})

	err := mgr.SSH("stopped-ws")
	if err == nil {
		t.Fatal("expected error for SSH into stopped workspace")
	}
	wsErr, ok := err.(*WorkspaceError)
	if !ok {
		t.Fatalf("expected *WorkspaceError, got %T", err)
	}
	if wsErr.Suggestion == "" {
		t.Error("expected suggestion in SSH error")
	}
}

func TestManager_StopAlreadyStopped(t *testing.T) {
	mgr := testManager(t)
	mgr.state.Put(&Workspace{
		Name:       "stopped-ws",
		Project:    "stopped-ws",
		Status:     StatusStopped,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	})

	// Stop should be idempotent — no error, no SSH call needed.
	err := mgr.Stop("stopped-ws")
	if err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
}

func TestManager_StartAlreadyRunning(t *testing.T) {
	mgr := testManager(t)
	mgr.state.Put(&Workspace{
		Name:       "running-ws",
		Project:    "running-ws",
		Status:     StatusRunning,
		ServerHost: "devbox-vps",
		CreatedAt:  time.Now(),
	})

	// Start should be idempotent.
	err := mgr.Start("running-ws")
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
}

// TestNewExecutor_PreservesEnvForHostRuntime regression-tests that
// newExecutor (used to reconstruct a host-runtime executor on Start/Stop/
// Destroy/Logs from persisted state) carries Env through — previously it
// was dropped, so restarting a host workspace relaunched serve with no
// environment at all.
func TestNewExecutor_PreservesEnvForHostRuntime(t *testing.T) {
	mgr := testManager(t)
	ws := &Workspace{
		Name:       "host-ws",
		ServerHost: "box1",
		Runtime:    config.RuntimeHost,
		Serve:      "npm start",
		Env:        map[string]string{"FOO": "bar"},
	}

	mock := &recordingSSH{}
	ex, err := mgr.newExecutor(mock, ws)
	if err != nil {
		t.Fatalf("newExecutor() error: %v", err)
	}

	if err := ex.Up(context.Background()); err != nil {
		t.Fatalf("Up() error: %v", err)
	}

	found := false
	for _, c := range mock.calls {
		if strings.Contains(c, "export FOO=") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reconstructed host executor to include workspace Env, calls: %v", mock.calls)
	}
}

func TestManager_ListFilterByUser(t *testing.T) {
	mgr := testManager(t)
	now := time.Now()
	mgr.state.Put(&Workspace{
		Name: "alice-proj", User: "alice", Project: "proj",
		Status: StatusRunning, ServerHost: "devbox-vps", CreatedAt: now,
	})
	mgr.state.Put(&Workspace{
		Name: "bob-proj", User: "bob", Project: "proj",
		Status: StatusRunning, ServerHost: "devbox-vps", CreatedAt: now,
	})

	ws, err := mgr.List(ListOptions{User: "alice"})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(ws) != 1 {
		t.Errorf("List(alice) returned %d, want 1", len(ws))
	}
	if len(ws) > 0 && ws[0].User != "alice" {
		t.Errorf("expected alice's workspace, got user=%q", ws[0].User)
	}

	all, err := mgr.List(ListOptions{All: true})
	if err != nil {
		t.Fatalf("List(all) error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List(all) returned %d, want 2", len(all))
	}
}

func TestFormatName(t *testing.T) {
	tests := []struct {
		user, project, branch, want string
	}{
		{"alice", "myapp", "main", "alice-myapp-main"},
		{"bob", "proj", "", "bob-proj"},
		{"Alice", "MyApp", "feature/auth", "alice-myapp-feature-auth"},
	}
	for _, tt := range tests {
		got := FormatName(tt.user, tt.project, tt.branch)
		if got != tt.want {
			t.Errorf("FormatName(%q, %q, %q) = %q, want %q",
				tt.user, tt.project, tt.branch, got, tt.want)
		}
	}
}

// --- needsRebuild / git command builders (pure, no SSH needed) ---

func TestNeedsRebuild(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{"empty diff", nil, false},
		{"js only", []string{"src/App.js", "src/Home.js"}, false},
		{"package.json", []string{"package.json"}, true},
		{"npm lockfile", []string{"package-lock.json"}, true},
		{"yarn lockfile", []string{"yarn.lock"}, true},
		{"pnpm lockfile", []string{"pnpm-lock.yaml"}, true},
		{"app.json", []string{"app.json"}, true},
		{"app.config.js", []string{"app.config.js"}, true},
		{"app.config.ts", []string{"app.config.ts"}, true},
		{"android prefix", []string{"android/app/build.gradle"}, true},
		{"ios prefix", []string{"ios/Podfile"}, true},
		{"plugins prefix", []string{"plugins/my-plugin/index.js"}, true},
		{"unrelated filename containing package.json", []string{"src/package.json.bak"}, false},
		{"mixed js + native", []string{"src/App.js", "android/build.gradle"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsRebuild(tt.files); got != tt.want {
				t.Errorf("needsRebuild(%v) = %v, want %v", tt.files, got, tt.want)
			}
		})
	}
}

func TestGitFetchCheckoutCmd(t *testing.T) {
	got := gitFetchCheckoutCmd("/workspaces/ws/src", "feature/x")
	want := "git -C /workspaces/ws/src fetch origin feature/x && " +
		"git -C /workspaces/ws/src checkout -B feature/x FETCH_HEAD && " +
		"git -C /workspaces/ws/src reset --hard FETCH_HEAD"
	if got != want {
		t.Errorf("gitFetchCheckoutCmd() = %q, want %q", got, want)
	}
}

func TestGitRevParseHeadCmd(t *testing.T) {
	got := gitRevParseHeadCmd("/workspaces/ws/src")
	if !strings.Contains(got, "git -C /workspaces/ws/src rev-parse HEAD") {
		t.Errorf("gitRevParseHeadCmd() = %q, want it to rev-parse HEAD in srcDir", got)
	}
	if !strings.Contains(got, "|| echo ''") {
		t.Errorf("gitRevParseHeadCmd() = %q, want it to always exit 0", got)
	}
}

func TestGitDiffNamesCmd(t *testing.T) {
	got := gitDiffNamesCmd("/workspaces/ws/src", "abc123")
	want := "git -C /workspaces/ws/src diff --name-only abc123..HEAD"
	if got != want {
		t.Errorf("gitDiffNamesCmd() = %q, want %q", got, want)
	}
}

// --- Refresh ---

func refreshTestManager(t *testing.T, rec *recordingSSH) *remoteManager {
	t.Helper()
	mgr := testManager(t)
	mgr.sshFactory = func() (devboxssh.Executor, error) { return rec, nil }
	return mgr
}

// scriptedSSH fakes a host that reports its serve process as alive (`cat`
// returns a PID, `kill -0` succeeds) and returns oldRev/diffOut for the
// git rev-parse/diff commands Refresh issues, so a test can drive it through
// a specific fast-refresh vs. rebuild decision without a live host.
func scriptedSSH(oldRev, diffOut string) *recordingSSH {
	return &recordingSSH{
		runFunc: func(cmd string) (string, string, error) {
			switch {
			case strings.HasPrefix(cmd, "cat "):
				return "9999", "", nil
			case strings.HasPrefix(cmd, "kill -0"):
				return "", "", nil // process reports alive
			case strings.Contains(cmd, "rev-parse HEAD"):
				return oldRev, "", nil
			case strings.Contains(cmd, "diff --name-only"):
				return diffOut, "", nil
			default:
				return "", "", nil
			}
		},
	}
}

func TestManager_Refresh_NotFound(t *testing.T) {
	mgr := refreshTestManager(t, &recordingSSH{})
	_, err := mgr.Refresh(RefreshParams{Name: "nonexistent", Branch: "main"})
	if err == nil {
		t.Fatal("expected error refreshing a nonexistent workspace")
	}
}

func TestManager_Refresh_RejectsNonHostRuntime(t *testing.T) {
	mgr := refreshTestManager(t, &recordingSSH{})
	mgr.state.Put(&Workspace{
		Name: "docker-ws", ServerHost: "box1", Runtime: config.RuntimeDocker, Branch: "main",
	})

	_, err := mgr.Refresh(RefreshParams{Name: "docker-ws", Branch: "feature"})
	if err == nil {
		t.Fatal("expected error refreshing a docker-runtime workspace")
	}
}

func TestManager_Refresh_InvalidBranch(t *testing.T) {
	mgr := refreshTestManager(t, &recordingSSH{})
	mgr.state.Put(&Workspace{Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost, Branch: "main"})

	_, err := mgr.Refresh(RefreshParams{Name: "host-ws", Branch: "bad; rm -rf /"})
	if err == nil {
		t.Fatal("expected error for invalid branch name")
	}
}

func TestManager_Refresh_NewBranchJSOnlyFastRefresh(t *testing.T) {
	rec := scriptedSSH("oldsha123", "src/App.js\nsrc/Home.js\n")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	ws, err := mgr.Refresh(RefreshParams{
		Name: "host-ws", Branch: "feature", Setup: []string{"npm ci"}, Serve: "npm start",
	})
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if ws.Branch != "feature" {
		t.Errorf("Branch = %q, want %q", ws.Branch, "feature")
	}
	if ws.Status != StatusRunning {
		t.Errorf("Status = %q, want running", ws.Status)
	}

	var sawSetupRerun, sawForcedRestart bool
	for _, c := range rec.calls {
		if strings.Contains(c, "npm ci") {
			sawSetupRerun = true
		}
		if strings.Contains(c, "setsid") || strings.Contains(c, "kill -TERM") {
			sawForcedRestart = true
		}
	}
	if sawSetupRerun {
		t.Errorf("fast-refresh (JS-only diff) should not re-run setup, calls: %v", rec.calls)
	}
	if sawForcedRestart {
		t.Errorf("fast-refresh (JS-only diff) should not force a restart, calls: %v", rec.calls)
	}
}

func TestManager_Refresh_SameBranchEmptyDiffFastRefresh(t *testing.T) {
	rec := scriptedSSH("samesha", "")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	// No branch override — Refresh should default to the workspace's
	// existing branch rather than requiring the caller to repeat it.
	ws, err := mgr.Refresh(RefreshParams{Name: "host-ws", Setup: []string{"npm ci"}, Serve: "npm start"})
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if ws.Branch != "main" {
		t.Errorf("Branch = %q, want %q (unchanged)", ws.Branch, "main")
	}
	for _, c := range rec.calls {
		if strings.Contains(c, "npm ci") {
			t.Errorf("same-branch empty diff should not re-run setup, calls: %v", rec.calls)
		}
	}
}

// TestManager_Refresh_EmptyPersistedBranchDefaultsToMain regression-tests
// that a workspace created without an explicit branch (ws.Branch == "",
// mirroring Create's clone default of "main") can still be refreshed — not
// fail branch validation on an empty string.
func TestManager_Refresh_EmptyPersistedBranchDefaultsToMain(t *testing.T) {
	rec := scriptedSSH("oldsha", "")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	ws, err := mgr.Refresh(RefreshParams{Name: "host-ws", Setup: []string{"npm ci"}, Serve: "npm start"})
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if ws.Branch != "main" {
		t.Errorf("Branch = %q, want %q", ws.Branch, "main")
	}
}

func TestManager_Refresh_LockfileChangeTriggersRebuild(t *testing.T) {
	rec := scriptedSSH("oldsha", "package-lock.json\n")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	if _, err := mgr.Refresh(RefreshParams{Name: "host-ws", Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start"}); err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}

	var sawSetup, sawRestart bool
	for _, c := range rec.calls {
		if strings.Contains(c, "npm ci") {
			sawSetup = true
		}
		if strings.Contains(c, "setsid") {
			sawRestart = true
		}
	}
	if !sawSetup {
		t.Errorf("lockfile change should re-run setup, calls: %v", rec.calls)
	}
	if !sawRestart {
		t.Errorf("lockfile change should restart serve, calls: %v", rec.calls)
	}
}

func TestManager_Refresh_NativePathChangeTriggersRebuild(t *testing.T) {
	rec := scriptedSSH("oldsha", "android/app/build.gradle\n")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	if _, err := mgr.Refresh(RefreshParams{Name: "host-ws", Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start"}); err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}

	var sawSetup, sawRestart bool
	for _, c := range rec.calls {
		if strings.Contains(c, "npm ci") {
			sawSetup = true
		}
		if strings.Contains(c, "setsid") {
			sawRestart = true
		}
	}
	if !sawSetup || !sawRestart {
		t.Errorf("native-path change should re-run setup and restart serve, calls: %v", rec.calls)
	}
}

func TestManager_Refresh_UnknownOldRevForcesRebuild(t *testing.T) {
	// Empty rev-parse output (e.g. a corrupt/missing repo) means the diff is
	// unknown — Refresh must default to the safe superset (rebuild) rather
	// than risk skipping a native change it can't see.
	rec := scriptedSSH("", "")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start",
	})

	if _, err := mgr.Refresh(RefreshParams{Name: "host-ws", Branch: "main", Setup: []string{"npm ci"}, Serve: "npm start"}); err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}

	found := false
	for _, c := range rec.calls {
		if strings.Contains(c, "npm ci") {
			found = true
		}
	}
	if !found {
		t.Errorf("unknown old rev should force a rebuild, calls: %v", rec.calls)
	}
}

func TestManager_Refresh_PersistsFreshConfigNotStaleState(t *testing.T) {
	rec := scriptedSSH("oldsha", "")
	mgr := refreshTestManager(t, rec)
	mgr.state.Put(&Workspace{
		Name: "host-ws", ServerHost: "box1", Runtime: config.RuntimeHost,
		Branch: "main", Setup: []string{"old-setup"}, Serve: "old-serve",
		Env: map[string]string{"OLD": "1"},
	})

	ws, err := mgr.Refresh(RefreshParams{
		Name: "host-ws", Branch: "main",
		Setup: []string{"new-setup"}, Serve: "new-serve",
		Env: map[string]string{"NEW": "2"},
	})
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if len(ws.Setup) != 1 || ws.Setup[0] != "new-setup" {
		t.Errorf("Setup = %v, want fresh [new-setup]", ws.Setup)
	}
	if ws.Serve != "new-serve" {
		t.Errorf("Serve = %q, want %q", ws.Serve, "new-serve")
	}
	if ws.Env["NEW"] != "2" || ws.Env["OLD"] != "" {
		t.Errorf("Env = %v, want only the fresh NEW=2", ws.Env)
	}

	persisted, err := mgr.state.Get("host-ws")
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if persisted.Serve != "new-serve" {
		t.Errorf("persisted Serve = %q, want %q", persisted.Serve, "new-serve")
	}
}

func TestFormatPath(t *testing.T) {
	tests := []struct {
		root, user, project, branch, want string
	}{
		{"/workspaces", "alice", "myapp", "main", "/workspaces/alice/myapp-main"},
		{"/workspaces", "bob", "proj", "", "/workspaces/bob/proj"},
	}
	for _, tt := range tests {
		got := FormatPath(tt.root, tt.user, tt.project, tt.branch)
		if got != tt.want {
			t.Errorf("FormatPath(%q, %q, %q, %q) = %q, want %q",
				tt.root, tt.user, tt.project, tt.branch, got, tt.want)
		}
	}
}
