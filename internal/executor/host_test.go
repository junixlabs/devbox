package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/junixlabs/devbox/internal/config"
)

// localExecSSH implements ssh.Executor by running commands through a real
// local bash instead of over SSH — used to verify actual shell quoting
// behavior (e.g. env values with spaces) rather than just inspecting the
// generated command string.
type localExecSSH struct{}

func (l *localExecSSH) Run(ctx context.Context, _ string, command string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (l *localExecSSH) RunStream(ctx context.Context, _ string, command string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (l *localExecSSH) CopyTo(context.Context, string, string, string) error   { return nil }
func (l *localExecSSH) CopyFrom(context.Context, string, string, string) error { return nil }
func (l *localExecSSH) Close() error                                          { return nil }

func TestHostExecutor_Deploy_RunsSetupThenDetachedServe(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{
		Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost,
		Setup: []string{"npm install"}, Serve: "npm start",
		Env:            map[string]string{"FOO": "bar"},
		WorkspacesRoot: "/workspaces",
	}

	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Deploy(context.Background()); err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}

	if len(mock.calls) != 3 {
		t.Fatalf("expected 3 SSH calls, got %d: %v", len(mock.calls), mock.calls)
	}
	if !strings.Contains(mock.calls[0], "mkdir -p /workspaces/test-ws/src") {
		t.Errorf("call 0 = %q, want mkdir src dir", mock.calls[0])
	}
	if !strings.Contains(mock.calls[1], "npm install") || !strings.Contains(mock.calls[1], "export FOO='bar'") {
		t.Errorf("call 1 = %q, want setup command with exported env", mock.calls[1])
	}
	if !strings.Contains(mock.calls[2], "setsid") || !strings.Contains(mock.calls[2], "npm start") ||
		!strings.Contains(mock.calls[2], "serve.log") || !strings.Contains(mock.calls[2], "serve.pid") {
		t.Errorf("call 2 = %q, want detached setsid serve launch with log+pid files", mock.calls[2])
	}
}

func TestHostExecutor_Deploy_NoServeConfigured(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Deploy(context.Background()); err == nil {
		t.Fatal("expected error when no serve command is configured")
	}
}

func TestHostExecutor_Deploy_SetupFailureAborts(t *testing.T) {
	mock := &mockSSHExecutor{
		runFunc: func(cmd string) (string, string, error) {
			if strings.Contains(cmd, "npm install") {
				return "", "npm ERR!", errors.New("exit 1")
			}
			return "", "", nil
		},
	}
	cfg := &config.DevboxConfig{
		Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost,
		Setup: []string{"npm install"}, Serve: "npm start", WorkspacesRoot: "/workspaces",
	}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Deploy(context.Background()); err == nil {
		t.Fatal("expected error when setup command fails")
	}
	// Should not have launched serve after a failed setup command.
	for _, c := range mock.calls {
		if strings.Contains(c, "setsid") {
			t.Errorf("serve should not launch after setup failure, but found: %q", c)
		}
	}
}

// TestHostExecutor_StartServe_EnvWithSpaceSurvives regression-tests the
// double-nested bash -c quoting bug: an env value containing a space used
// to corrupt the launch command so the serve process never actually ran.
// This drives startServe through a real local shell (not a mock) so it
// fails without the fix and passes with it.
func TestHostExecutor_StartServe_EnvWithSpaceSurvives(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "out.txt")

	h := &hostExecutor{
		ssh:     &localExecSSH{},
		host:    "localhost",
		name:    "test-ws",
		srcDir:  dir,
		logFile: filepath.Join(dir, "serve.log"),
		pidFile: filepath.Join(dir, "serve.pid"),
		serve:   fmt.Sprintf(`sh -c 'printf "%%s" "$FOO" > %s'`, outFile),
		env:     map[string]string{"FOO": "bar baz"},
	}

	if err := h.startServe(context.Background()); err != nil {
		t.Fatalf("startServe() error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var out []byte
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(outFile)
		if err == nil {
			out = b
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if string(out) != "bar baz" {
		t.Errorf("serve output = %q, want %q (env value with a space must survive the launch command)", out, "bar baz")
	}
}

func TestHostExecutor_Down_KillsProcessGroup(t *testing.T) {
	mock := &mockSSHExecutor{
		runFunc: func(cmd string) (string, string, error) {
			if strings.HasPrefix(cmd, "cat ") {
				return "1234", "", nil
			}
			return "", "", nil
		},
	}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Down(context.Background()); err != nil {
		t.Fatalf("Down() error: %v", err)
	}

	found := false
	for _, c := range mock.calls {
		if strings.Contains(c, "kill -TERM -- -1234") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a kill -TERM -- -1234 call, got: %v", mock.calls)
	}
}

func TestHostExecutor_Down_NoPidIsNoop(t *testing.T) {
	// Real remote behavior: `cat serve.pid 2>/dev/null || true` always exits 0,
	// with empty stdout when the PID file doesn't exist (never started / already stopped).
	mock := &mockSSHExecutor{runOut: ""}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Down(context.Background()); err != nil {
		t.Fatalf("Down() should be a no-op when no PID is recorded, got: %v", err)
	}
}

func TestHostExecutor_Down_PropagatesConnectionFailure(t *testing.T) {
	// A genuine SSH/transport failure must NOT be swallowed as "already stopped" —
	// otherwise `devbox stop` would falsely report success while unable to reach the host.
	mock := &mockSSHExecutor{runErr: errors.New("ssh: connect to host box1 port 22: connection refused")}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Down(context.Background()); err == nil {
		t.Fatal("Down() should propagate a genuine connection failure, got nil error")
	}
}

func TestHostExecutor_Destroy_RemovesWorkdir(t *testing.T) {
	mock := &mockSSHExecutor{
		runFunc: func(cmd string) (string, string, error) {
			if strings.HasPrefix(cmd, "cat ") {
				return "1234", "", nil
			}
			return "", "", nil
		},
	}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Destroy(context.Background()); err != nil {
		t.Fatalf("Destroy() error: %v", err)
	}

	found := false
	for _, c := range mock.calls {
		if strings.Contains(c, "rm -rf /workspaces/test-ws") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected rm -rf of workdir, got: %v", mock.calls)
	}
}

func TestHostExecutor_Logs_FollowVsDump(t *testing.T) {
	mock := &mockSSHExecutor{}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	var stdout, stderr strings.Builder
	if err := ex.Logs(context.Background(), true, &stdout, &stderr); err != nil {
		t.Fatalf("Logs(follow) error: %v", err)
	}
	if !strings.Contains(mock.calls[0], "tail -n +1 -f") {
		t.Errorf("follow call = %q, want tail -f", mock.calls[0])
	}

	mock2 := &mockSSHExecutor{}
	ex2, _ := newHostExecutor(mock2, cfg, "box1", "test-ws")
	if err := ex2.Logs(context.Background(), false, &stdout, &stderr); err != nil {
		t.Fatalf("Logs(dump) error: %v", err)
	}
	if !strings.Contains(mock2.calls[0], "cat ") || strings.Contains(mock2.calls[0], "tail") {
		t.Errorf("non-follow call = %q, want cat (no follow)", mock2.calls[0])
	}
}

func TestHostExecutor_Up_AlreadyAliveIsNoop(t *testing.T) {
	mock := &mockSSHExecutor{
		runFunc: func(cmd string) (string, string, error) {
			if strings.HasPrefix(cmd, "cat ") {
				return "1234", "", nil
			}
			if strings.HasPrefix(cmd, "kill -0") {
				return "", "", nil // process alive
			}
			return "", "", nil
		},
	}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, Serve: "npm start", WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Up(context.Background()); err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	for _, c := range mock.calls {
		if strings.Contains(c, "setsid") {
			t.Errorf("Up() should not relaunch serve when already alive, got: %v", mock.calls)
		}
	}
}

func TestHostExecutor_Up_DeadRelaunchesServe(t *testing.T) {
	mock := &mockSSHExecutor{
		runFunc: func(cmd string) (string, string, error) {
			if strings.HasPrefix(cmd, "cat ") {
				return "1234", "", nil
			}
			if strings.HasPrefix(cmd, "kill -0") {
				return "", "", errors.New("no such process")
			}
			return "", "", nil
		},
	}
	cfg := &config.DevboxConfig{Name: "test-ws", Server: "box1", Runtime: config.RuntimeHost, Serve: "npm start", WorkspacesRoot: "/workspaces"}
	ex, err := newHostExecutor(mock, cfg, "box1", "test-ws")
	if err != nil {
		t.Fatalf("newHostExecutor() error: %v", err)
	}

	if err := ex.Up(context.Background()); err != nil {
		t.Fatalf("Up() error: %v", err)
	}
	found := false
	for _, c := range mock.calls {
		if strings.Contains(c, "setsid") {
			found = true
		}
	}
	if !found {
		t.Errorf("Up() should relaunch serve when dead, got: %v", mock.calls)
	}
}

func TestShellQuote_EscapesSingleQuotes(t *testing.T) {
	got := shellQuote("it's a test")
	want := `'it'"'"'s a test'`
	if got != want {
		t.Errorf("shellQuote() = %q, want %q", got, want)
	}
}

func TestExportPrefix_RejectsUnsafeKeys(t *testing.T) {
	h := &hostExecutor{env: map[string]string{"good_KEY": "1", "bad;key": "2"}}
	prefix := h.exportPrefix()
	if !strings.Contains(prefix, "good_KEY") {
		t.Errorf("prefix = %q, want it to contain good_KEY", prefix)
	}
	if strings.Contains(prefix, "bad;key") {
		t.Errorf("prefix = %q, should not contain unsafe key bad;key", prefix)
	}
}
