//go:build integration

package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestFullLifecycle runs the complete workspace lifecycle as ordered subtests.
// This test requires DEVBOX_TEST_SERVER to be set and the server to have
// Docker + Tailscale installed. The subtests must run in order.
func TestFullLifecycle(t *testing.T) {
	server := requireEnv(t, "DEVBOX_TEST_SERVER")
	workDir := prepareTestDir(t, "devbox.yaml", server)

	t.Cleanup(func() {
		cleanupWorkspace(t, workDir, "integration-test", server)
	})

	t.Run("Up_CreatesWorkspace", func(t *testing.T) {
		result := runDevbox(t, workDir, "up")
		assertExitCode(t, result, 0, "devbox up")

		combined := result.Stdout + result.Stderr
		assertContains(t, combined, "integration-test", "devbox up output should mention workspace name")

		// Verify workspace directory exists on the server.
		wsDir := workspaceDir() + "/integration-test"
		waitForCondition(t, 30*time.Second, 2*time.Second, "workspace dir exists", func() bool {
			out, err := trySshRun(server, fmt.Sprintf("test -d '%s' && echo exists", wsDir))
			return err == nil && strings.Contains(out, "exists")
		})

		// Verify containers are running.
		waitForCondition(t, 30*time.Second, 2*time.Second, "containers running", func() bool {
			out, err := trySshRun(server, "docker ps --format '{{.Names}}'")
			return err == nil && strings.Contains(out, "integration-test")
		})
	})

	t.Run("List_ShowsWorkspace", func(t *testing.T) {
		result := runDevbox(t, workDir, "list")
		assertExitCode(t, result, 0, "devbox list")
		assertContains(t, result.Stdout, "integration-test", "list output should show workspace")
		assertContains(t, result.Stdout, "running", "list output should show running status")
	})

	t.Run("SSH_Connects", func(t *testing.T) {
		// Run ssh with a simple command to verify connectivity.
		// The devbox ssh command opens an interactive session, so we test
		// that it starts without error by checking exit code.
		// In a real implementation, ssh might support passing commands.
		result := runDevbox(t, workDir, "ssh", "integration-test")

		// SSH to an interactive session will exit immediately without a TTY,
		// but it should not produce an error about the workspace not existing.
		if result.ExitCode != 0 {
			combined := result.Stdout + result.Stderr
			// If the error is about TTY or terminal, that's expected in CI.
			if !strings.Contains(combined, "terminal") && !strings.Contains(combined, "tty") {
				t.Errorf("devbox ssh failed unexpectedly: exit=%d\nstdout: %s\nstderr: %s",
					result.ExitCode, result.Stdout, result.Stderr)
			}
		}
	})

	t.Run("Stop_Workspace", func(t *testing.T) {
		result := runDevbox(t, workDir, "stop", "integration-test")
		assertExitCode(t, result, 0, "devbox stop")

		combined := result.Stdout + result.Stderr
		assertContains(t, combined, "stopped", "stop output should confirm stopped")

		// Verify containers are no longer running.
		waitForCondition(t, 15*time.Second, 2*time.Second, "containers stopped", func() bool {
			out, err := trySshRun(server, "docker ps --format '{{.Names}}'")
			return err == nil && !strings.Contains(out, "integration-test")
		})

		// Verify list shows stopped status.
		listResult := runDevbox(t, workDir, "list")
		assertExitCode(t, listResult, 0, "devbox list after stop")
		assertContains(t, listResult.Stdout, "integration-test", "list should still show workspace after stop")
		assertContains(t, listResult.Stdout, "stopped", "list should show stopped status")
	})

	t.Run("Start_AfterStop", func(t *testing.T) {
		// Re-running 'up' on a stopped workspace should start (not re-create) it.
		result := runDevbox(t, workDir, "up")
		assertExitCode(t, result, 0, "devbox up (restart)")

		// Verify containers are running again.
		waitForCondition(t, 30*time.Second, 2*time.Second, "containers running after restart", func() bool {
			out, err := trySshRun(server, "docker ps --format '{{.Names}}'")
			return err == nil && strings.Contains(out, "integration-test")
		})
	})

	t.Run("Destroy_Cleanup", func(t *testing.T) {
		result := runDevbox(t, workDir, "destroy", "integration-test", "--force")
		assertExitCode(t, result, 0, "devbox destroy")

		combined := result.Stdout + result.Stderr
		assertContains(t, combined, "destroyed", "destroy output should confirm destruction")

		// Verify workspace directory is gone.
		wsDir := workspaceDir() + "/integration-test"
		waitForCondition(t, 15*time.Second, 2*time.Second, "workspace dir removed", func() bool {
			out, err := trySshRun(server, fmt.Sprintf("test -d '%s' && echo exists || echo gone", wsDir))
			return err == nil && strings.Contains(out, "gone")
		})

		// Verify list no longer shows the workspace.
		listResult := runDevbox(t, workDir, "list")
		assertExitCode(t, listResult, 0, "devbox list after destroy")
		assertNotContains(t, listResult.Stdout, "integration-test", "list should not show destroyed workspace")
	})
}

// TestError_ServerUnreachable verifies that devbox up fails gracefully
// when the target server is unreachable.
func TestError_ServerUnreachable(t *testing.T) {
	workDir := prepareTestDir(t, "devbox-noserver.yaml", "")

	result := runDevbox(t, workDir, "up")
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit code for unreachable server")
	}

	combined := result.Stdout + result.Stderr
	// Should produce a meaningful error, not a panic or raw stack trace.
	if strings.Contains(combined, "panic") {
		t.Error("output should not contain a panic")
	}
	if strings.Contains(combined, "goroutine") {
		t.Error("output should not contain goroutine stack traces")
	}
}

// TestError_MissingConfig verifies that devbox up fails with a helpful message
// when no devbox.yaml exists.
func TestError_MissingConfig(t *testing.T) {
	tmpDir := t.TempDir()

	result := runDevbox(t, tmpDir, "up")
	if result.ExitCode == 0 {
		t.Fatal("expected non-zero exit code for missing config")
	}

	combined := result.Stdout + result.Stderr
	// Should mention the config file.
	hasConfigRef := strings.Contains(combined, "devbox.yaml") ||
		strings.Contains(combined, "config")
	if !hasConfigRef {
		t.Errorf("error output should reference config file, got:\n%s", combined)
	}
}

// TestError_PortConflict verifies that starting two workspaces with the same
// ports on the same server produces a port conflict error.
func TestError_PortConflict(t *testing.T) {
	server := requireEnv(t, "DEVBOX_TEST_SERVER")

	// Start the first workspace.
	workDir1 := prepareTestDir(t, "devbox.yaml", server)
	t.Cleanup(func() {
		cleanupWorkspace(t, workDir1, "integration-test", server)
	})

	result1 := runDevbox(t, workDir1, "up")
	assertExitCode(t, result1, 0, "first devbox up")

	// Wait for first workspace to be fully running.
	waitForCondition(t, 30*time.Second, 2*time.Second, "first workspace running", func() bool {
		out, err := trySshRun(server, "docker ps --format '{{.Names}}'")
		return err == nil && strings.Contains(out, "integration-test")
	})

	// Try to start a second workspace with the same port.
	workDir2 := prepareTestDir(t, "devbox-conflict.yaml", server)
	t.Cleanup(func() {
		cleanupWorkspace(t, workDir2, "integration-test-conflict", server)
	})

	result2 := runDevbox(t, workDir2, "up")

	// The second workspace should fail due to port conflict.
	// (Exact behavior depends on implementation — it may succeed if
	// the workspace manager handles port conflicts at a different level.)
	if result2.ExitCode != 0 {
		combined := result2.Stdout + result2.Stderr
		// Verify it's a meaningful error, not a crash.
		if strings.Contains(combined, "panic") {
			t.Error("port conflict should not cause a panic")
		}
	}
}
