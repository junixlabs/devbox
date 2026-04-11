package server

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

type mockExecutor struct {
	runFunc func(ctx context.Context, host string, command string) (string, string, error)
}

func (m *mockExecutor) Run(ctx context.Context, host string, command string) (string, string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, host, command)
	}
	return "ok", "", nil
}
func (m *mockExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}
func (m *mockExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error  { return nil }
func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error { return nil }
func (m *mockExecutor) Close() error                                                    { return nil }

func newTestPool(t *testing.T, exec devboxssh.Executor) Pool {
	t.Helper()
	path := filepath.Join(t.TempDir(), "servers.yaml")
	pool, err := NewFilePool(path, exec)
	if err != nil {
		t.Fatalf("NewFilePool: %v", err)
	}
	return pool
}

func TestAddAndList(t *testing.T) {
	pool := newTestPool(t, nil)
	srv, err := pool.Add("dev1", "10.0.0.1", WithUser("root"), WithPort(2222))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if srv.Name != "dev1" || srv.Host != "10.0.0.1" || srv.User != "root" || srv.Port != 2222 {
		t.Errorf("unexpected server: %+v", srv)
	}
	servers, err := pool.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(servers) != 1 || servers[0].Name != "dev1" {
		t.Fatalf("expected 1 server named dev1, got %+v", servers)
	}
}

func TestAddDuplicate(t *testing.T) {
	pool := newTestPool(t, nil)
	if _, err := pool.Add("dev1", "10.0.0.1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := pool.Add("dev1", "10.0.0.2"); err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestRemove(t *testing.T) {
	pool := newTestPool(t, nil)
	if _, err := pool.Add("dev1", "10.0.0.1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := pool.Remove("dev1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	servers, _ := pool.List()
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(servers))
	}
}

func TestRemoveNotFound(t *testing.T) {
	pool := newTestPool(t, nil)
	if err := pool.Remove("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestHealthCheckAllPass(t *testing.T) {
	exec := &mockExecutor{
		runFunc: func(_ context.Context, _ string, _ string) (string, string, error) {
			return "ok", "", nil
		},
	}
	pool := newTestPool(t, exec)
	if _, err := pool.Add("dev1", "10.0.0.1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	status, err := pool.HealthCheck("dev1")
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if !status.SSH || !status.Docker || !status.Tailscale {
		t.Errorf("expected all checks to pass: %+v", status)
	}
}

func TestHealthCheckNotFound(t *testing.T) {
	pool := newTestPool(t, nil)
	if _, err := pool.HealthCheck("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestHealthCheckNilExecutor(t *testing.T) {
	pool := newTestPool(t, nil)
	if _, err := pool.Add("dev1", "10.0.0.1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	status, err := pool.HealthCheck("dev1")
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if status.SSH || status.Docker || status.Tailscale {
		t.Errorf("expected all checks to fail with nil executor: %+v", status)
	}
}

func TestListEmpty(t *testing.T) {
	pool := newTestPool(t, nil)
	servers, _ := pool.List()
	if len(servers) != 0 {
		t.Fatalf("expected 0 servers, got %d", len(servers))
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "servers.yaml")
	pool1, _ := NewFilePool(path, nil)
	pool1.Add("dev1", "10.0.0.1")
	pool2, _ := NewFilePool(path, nil)
	servers, _ := pool2.List()
	if len(servers) != 1 || servers[0].Name != "dev1" {
		t.Errorf("server not persisted: %+v", servers)
	}
}
