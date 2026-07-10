package executor

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/junixlabs/devbox/internal/config"
)

// mockSSHExecutor implements ssh.Executor for testing.
type mockSSHExecutor struct {
	calls     []string
	runOut    string
	runErr    error
	copyErr   error
	streamErr error
	// runFunc, if set, overrides the default Run behavior per-command —
	// used by host_test.go to simulate different responses for
	// e.g. `cat serve.pid` vs `kill -0 <pid>`.
	runFunc func(command string) (string, string, error)
}

func (m *mockSSHExecutor) Run(_ context.Context, _ string, command string) (string, string, error) {
	m.calls = append(m.calls, command)
	if m.runFunc != nil {
		return m.runFunc(command)
	}
	return m.runOut, "", m.runErr
}

func (m *mockSSHExecutor) RunStream(_ context.Context, _ string, command string, _ io.Writer, _ io.Writer) error {
	m.calls = append(m.calls, command)
	return m.streamErr
}

func (m *mockSSHExecutor) CopyTo(_ context.Context, _ string, _ string, remotePath string) error {
	m.calls = append(m.calls, "copy:"+remotePath)
	return m.copyErr
}

func (m *mockSSHExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockSSHExecutor) Close() error { return nil }

func TestDockerExecutor_Deploy(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "devbox-vps", Services: []string{"mysql:8.0"}}

	ex, err := newDockerExecutor(mock, cfg, "devbox-vps", "test-ws")
	if err != nil {
		t.Fatalf("newDockerExecutor() error: %v", err)
	}

	if err := ex.Deploy(context.Background()); err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}

	// Docker adapter path: mkdir, copy compose, docker compose up -d.
	if len(mock.calls) != 3 {
		t.Fatalf("expected 3 SSH calls, got %d: %v", len(mock.calls), mock.calls)
	}
	if !strings.Contains(mock.calls[0], "mkdir -p") {
		t.Errorf("call 0 = %q, want mkdir", mock.calls[0])
	}
	if !strings.HasPrefix(mock.calls[1], "copy:") {
		t.Errorf("call 1 = %q, want copy", mock.calls[1])
	}
	if !strings.Contains(mock.calls[2], "up -d") {
		t.Errorf("call 2 = %q, want docker compose up -d", mock.calls[2])
	}
}

func TestDockerExecutor_UpDownDestroyDelegate(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "devbox-vps"}
	ex, err := newDockerExecutor(mock, cfg, "devbox-vps", "test-ws")
	if err != nil {
		t.Fatalf("newDockerExecutor() error: %v", err)
	}

	if err := ex.Up(context.Background()); err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	if !strings.Contains(mock.calls[len(mock.calls)-1], "up -d") {
		t.Errorf("Up() did not delegate to docker compose up -d")
	}

	if err := ex.Down(context.Background()); err != nil {
		t.Fatalf("Down() error: %v", err)
	}
	if !strings.Contains(mock.calls[len(mock.calls)-1], "down") {
		t.Errorf("Down() did not delegate to docker compose down")
	}

	if err := ex.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}
	if !strings.Contains(mock.calls[len(mock.calls)-1], "rm -rf") {
		t.Errorf("Destroy() did not delegate to rm -rf")
	}
}

func TestDockerExecutor_Logs(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "devbox-vps", Services: []string{"mysql:8.0"}}
	ex, err := newDockerExecutor(mock, cfg, "devbox-vps", "test-ws")
	if err != nil {
		t.Fatalf("newDockerExecutor() error: %v", err)
	}

	var stdout, stderr strings.Builder
	if err := ex.Logs(context.Background(), false, &stdout, &stderr); err != nil {
		t.Fatalf("Logs() error: %v", err)
	}
	if strings.Contains(mock.calls[0], "--follow") {
		t.Errorf("call = %q, want no --follow when follow=false", mock.calls[0])
	}
	if !strings.Contains(mock.calls[0], "logs mysql") {
		t.Errorf("call = %q, want logs mysql", mock.calls[0])
	}
}

func TestDockerExecutor_LogsFollow(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "devbox-vps", Services: []string{"mysql:8.0"}}
	ex, err := newDockerExecutor(mock, cfg, "devbox-vps", "test-ws")
	if err != nil {
		t.Fatalf("newDockerExecutor() error: %v", err)
	}

	var stdout, stderr strings.Builder
	if err := ex.Logs(context.Background(), true, &stdout, &stderr); err != nil {
		t.Fatalf("Logs() error: %v", err)
	}
	if !strings.Contains(mock.calls[0], "logs --follow mysql") {
		t.Errorf("call = %q, want logs --follow mysql", mock.calls[0])
	}
}

func TestDockerExecutor_DeployError(t *testing.T) {
	mock := &mockSSHExecutor{runErr: errors.New("permission denied")}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "devbox-vps"}
	ex, err := newDockerExecutor(mock, cfg, "devbox-vps", "test-ws")
	if err != nil {
		t.Fatalf("newDockerExecutor() error: %v", err)
	}

	if err := ex.Deploy(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFirstService(t *testing.T) {
	tests := []struct {
		services []string
		want     string
	}{
		{nil, "app"},
		{[]string{}, "app"},
		{[]string{"mysql:8.0"}, "mysql"},
		{[]string{"bitnami/redis:7"}, "redis"},
	}
	for _, tt := range tests {
		if got := firstService(tt.services); got != tt.want {
			t.Errorf("firstService(%v) = %q, want %q", tt.services, got, tt.want)
		}
	}
}
