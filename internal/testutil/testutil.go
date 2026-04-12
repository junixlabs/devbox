// Package testutil provides shared helpers for integration tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/workspace"
)

const defaultTestServer = "devbox-vps"

// TestServer returns the test server hostname from DEVBOX_TEST_SERVER env.
// Skips the test if set to "skip". Defaults to "devbox-vps".
func TestServer(t *testing.T) string {
	t.Helper()
	server := os.Getenv("DEVBOX_TEST_SERVER")
	if server == "skip" {
		t.Skip("DEVBOX_TEST_SERVER=skip — skipping integration test")
	}
	if server == "" {
		server = defaultTestServer
	}
	return server
}

// NewManager returns a workspace.Manager for integration tests.
func NewManager() workspace.Manager {
	return workspace.NewManager()
}

// CreateWorkspace creates a workspace and registers t.Cleanup to destroy it.
// Attempts to destroy any existing workspace with the same name first.
func CreateWorkspace(t *testing.T, mgr workspace.Manager, params workspace.CreateParams) *workspace.Workspace {
	t.Helper()
	// Best-effort cleanup of leftover workspace from a previous failed run.
	_ = mgr.Destroy(params.Name)

	ws, err := mgr.Create(params)
	if err != nil {
		t.Fatalf("CreateWorkspace(%s): %v", params.Name, err)
	}
	t.Cleanup(func() {
		if err := mgr.Destroy(ws.Name); err != nil {
			t.Logf("cleanup Destroy(%s): %v", ws.Name, err)
		}
	})
	return ws
}

// SSHRun executes a command on the remote host and returns stdout.
// Fatals the test on error.
func SSHRun(t *testing.T, host, cmd string) string {
	t.Helper()
	stdout, err := SSHRunE(host, cmd)
	if err != nil {
		t.Fatalf("SSHRun(%s, %q): %v", host, cmd, err)
	}
	return stdout
}

// SSHRunE executes a command on the remote host and returns stdout and error.
func SSHRunE(host, cmd string) (string, error) {
	sshExec, err := devboxssh.New()
	if err != nil {
		return "", fmt.Errorf("ssh.New: %w", err)
	}
	defer sshExec.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stdout, _, err := sshExec.Run(ctx, host, cmd)
	if err != nil {
		return "", fmt.Errorf("ssh run %q: %w", cmd, err)
	}
	return strings.TrimSpace(stdout), nil
}

// DockerInspect runs docker inspect with a Go template format on a container.
func DockerInspect(t *testing.T, host, container, format string) string {
	t.Helper()
	cmd := fmt.Sprintf("docker inspect --format '%s' %s", format, container)
	return SSHRun(t, host, cmd)
}

// WaitForContainer polls until a container matching the prefix is running,
// or the timeout expires.
func WaitForContainer(t *testing.T, host, containerName string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := SSHRunE(host, fmt.Sprintf("docker inspect --format '{{.State.Running}}' %s 2>/dev/null", containerName))
		if err == nil && strings.TrimSpace(out) == "true" {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("container %s not running after %v", containerName, timeout)
}

// portListenerCount returns the number of listeners on a port via ss.
func portListenerCount(host string, port int) (string, error) {
	cmd := fmt.Sprintf("ss -tlnp 2>/dev/null | grep -c ':%d ' || true", port)
	out, err := SSHRunE(host, cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// AssertPortListening verifies that a port is bound on the remote host.
func AssertPortListening(t *testing.T, host string, port int) {
	t.Helper()
	count, err := portListenerCount(host, port)
	if err != nil {
		t.Fatalf("AssertPortListening: %v", err)
	}
	if count == "0" {
		t.Errorf("expected port %d to be listening on %s", port, host)
	}
}

// AssertPortFree verifies that a port is NOT bound on the remote host.
func AssertPortFree(t *testing.T, host string, port int) {
	t.Helper()
	count, err := portListenerCount(host, port)
	if err != nil {
		t.Fatalf("AssertPortFree: %v", err)
	}
	if count != "0" {
		t.Errorf("expected port %d to be free on %s, but it's in use", port, host)
	}
}

// AssertDirExists verifies that a directory exists on the remote host.
func AssertDirExists(t *testing.T, host, path string) {
	t.Helper()
	cmd := fmt.Sprintf("test -d %s && echo exists || echo missing", path)
	out := SSHRun(t, host, cmd)
	if strings.TrimSpace(out) != "exists" {
		t.Errorf("expected directory %s to exist on %s", path, host)
	}
}

// AssertDirNotExists verifies that a directory does NOT exist on the remote host.
func AssertDirNotExists(t *testing.T, host, path string) {
	t.Helper()
	cmd := fmt.Sprintf("test -d %s && echo exists || echo missing", path)
	out := SSHRun(t, host, cmd)
	if strings.TrimSpace(out) != "missing" {
		t.Errorf("expected directory %s to NOT exist on %s", path, host)
	}
}
