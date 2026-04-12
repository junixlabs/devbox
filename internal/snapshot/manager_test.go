package snapshot

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockExecutor records SSH commands and returns canned responses.
type mockExecutor struct {
	commands []string
	results  map[string]mockResult
}

type mockResult struct {
	stdout string
	stderr string
	err    error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		results: make(map[string]mockResult),
	}
}

func (m *mockExecutor) on(cmdPrefix string, stdout string, err error) {
	m.results[cmdPrefix] = mockResult{stdout: stdout, err: err}
}

func (m *mockExecutor) Run(_ context.Context, _ string, command string) (string, string, error) {
	m.commands = append(m.commands, command)
	for prefix, res := range m.results {
		if strings.HasPrefix(command, prefix) {
			return res.stdout, res.stderr, res.err
		}
	}
	return "", "", nil
}

func (m *mockExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}

func (m *mockExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error { return nil }
func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *mockExecutor) Close() error { return nil }

func (m *mockExecutor) hasCommandContaining(substr string) bool {
	for _, cmd := range m.commands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

func TestCreate(t *testing.T) {
	mock := newMockExecutor()
	mock.on("mkdir", "", nil)
	mock.on("docker ps", "myapp-web-1", nil)
	mock.on("docker ps -a", "myapp-web-1", nil)
	mock.on("docker inspect", "/var/lib/docker/volumes/myapp_data/_data ", nil)
	mock.on("for f in", "/workspaces/myapp/devbox.yaml ", nil)
	mock.on("tar czf", "", nil)
	mock.on("stat --printf", "1048576", nil)

	mgr := NewManager(mock)
	snap, err := mgr.Create("server1", "myapp", "backup1")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if snap.Name != "backup1" {
		t.Errorf("Name = %q, want %q", snap.Name, "backup1")
	}
	if snap.Workspace != "myapp" {
		t.Errorf("Workspace = %q, want %q", snap.Workspace, "myapp")
	}
	if snap.Size != 1048576 {
		t.Errorf("Size = %d, want %d", snap.Size, 1048576)
	}

	if !mock.hasCommandContaining("/workspaces/.snapshots/myapp") {
		t.Error("expected snapshot path to contain /workspaces/.snapshots/myapp")
	}
	if !mock.hasCommandContaining("tar czf") {
		t.Error("expected tar czf command")
	}
}

func TestCreateAutoName(t *testing.T) {
	mock := newMockExecutor()
	mock.on("mkdir", "", nil)
	mock.on("docker ps", "myapp-web-1", nil)
	mock.on("docker ps -a", "myapp-web-1", nil)
	mock.on("docker inspect", "/data ", nil)
	mock.on("for f in", "", nil)
	mock.on("tar czf", "", nil)
	mock.on("stat --printf", "512", nil)

	mgr := NewManager(mock)
	snap, err := mgr.Create("server1", "myapp", "")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if !strings.HasPrefix(snap.Name, "myapp-") {
		t.Errorf("auto-generated name %q should start with 'myapp-'", snap.Name)
	}
}

func TestCreateNoFiles(t *testing.T) {
	mock := newMockExecutor()
	mock.on("mkdir", "", nil)
	mock.on("docker ps", "", nil)
	mock.on("docker ps -a", "", nil)
	mock.on("docker inspect", "", nil)
	mock.on("for f in", "", nil)

	mgr := NewManager(mock)
	_, err := mgr.Create("server1", "myapp", "backup1")
	if err == nil {
		t.Fatal("Create() should fail when no files to snapshot")
	}
}

func TestCreateSSHError(t *testing.T) {
	mock := newMockExecutor()
	mock.on("mkdir", "", fmt.Errorf("connection refused"))

	mgr := NewManager(mock)
	_, err := mgr.Create("server1", "myapp", "backup1")
	if err == nil {
		t.Fatal("Create() should fail on SSH error")
	}
}

func TestRestore(t *testing.T) {
	mock := newMockExecutor()
	mock.on("test -f", "", nil)
	mock.on("docker ps -q", "", nil)
	mock.on("tar xzf", "", nil)
	mock.on("docker ps -aq", "", nil)

	mgr := NewManager(mock)
	err := mgr.Restore("server1", "myapp", "backup1")
	if err != nil {
		t.Fatalf("Restore() error: %v", err)
	}

	if !mock.hasCommandContaining("test -f") {
		t.Error("expected snapshot existence check")
	}
	if !mock.hasCommandContaining("docker stop") {
		t.Error("expected docker stop command")
	}
	if !mock.hasCommandContaining("tar xzf") {
		t.Error("expected tar extract command")
	}
	if !mock.hasCommandContaining("docker start") {
		t.Error("expected docker start command")
	}
}

func TestRestoreNotFound(t *testing.T) {
	mock := newMockExecutor()
	mock.on("test -f", "", fmt.Errorf("not found"))

	mgr := NewManager(mock)
	err := mgr.Restore("server1", "myapp", "nonexistent")
	if err == nil {
		t.Fatal("Restore() should fail when snapshot not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestList(t *testing.T) {
	mock := newMockExecutor()
	mock.on("test -d", "", nil)
	mock.on("find", `{"name":"backup1","size":1048576,"mtime":1712700000}
{"name":"backup2","size":2097152,"mtime":1712786400}
`, nil)

	mgr := NewManager(mock)
	snaps, err := mgr.List("server1", "myapp")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(snaps) != 2 {
		t.Fatalf("List() returned %d snapshots, want 2", len(snaps))
	}

	if snaps[0].Name != "backup1" {
		t.Errorf("snaps[0].Name = %q, want %q", snaps[0].Name, "backup1")
	}
	if snaps[0].Size != 1048576 {
		t.Errorf("snaps[0].Size = %d, want %d", snaps[0].Size, 1048576)
	}
	if snaps[1].Name != "backup2" {
		t.Errorf("snaps[1].Name = %q, want %q", snaps[1].Name, "backup2")
	}
}

func TestListNoDirectory(t *testing.T) {
	mock := newMockExecutor()
	mock.on("test -d", "", fmt.Errorf("not a directory"))

	mgr := NewManager(mock)
	snaps, err := mgr.List("server1", "myapp")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("List() returned %d snapshots, want 0", len(snaps))
	}
}

func TestListEmpty(t *testing.T) {
	mock := newMockExecutor()
	mock.on("test -d", "", nil)
	mock.on("find", "", nil)

	mgr := NewManager(mock)
	snaps, err := mgr.List("server1", "myapp")
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(snaps) != 0 {
		t.Errorf("List() returned %d snapshots, want 0", len(snaps))
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a\nb\nc\n", 3},
		{"a\nb\nc", 3},
		{"single", 1},
	}
	for _, tt := range tests {
		got := splitLines(tt.input)
		// Filter empty strings.
		var nonEmpty []string
		for _, s := range got {
			if s != "" {
				nonEmpty = append(nonEmpty, s)
			}
		}
		if len(nonEmpty) != tt.want {
			t.Errorf("splitLines(%q) = %d non-empty lines, want %d", tt.input, len(nonEmpty), tt.want)
		}
	}
}
