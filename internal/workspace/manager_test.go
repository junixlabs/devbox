package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
		got := FirstService(tt.services)
		if got != tt.want {
			t.Errorf("FirstService(%v) = %q, want %q", tt.services, got, tt.want)
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
