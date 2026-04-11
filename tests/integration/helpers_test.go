//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// binaryPath holds the path to the compiled devbox binary.
var binaryPath string

// TestMain compiles the devbox binary and sets up the test environment.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "devbox-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(tmpDir, "devbox")

	// Build the binary from the repo root.
	repoRoot := findRepoRoot()
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/devbox/")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build devbox binary: %v\n", err)
		os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// findRepoRoot walks up from the current directory to find the go.mod file.
func findRepoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Fallback: assume two levels up from tests/integration/
			return filepath.Join(".", "..", "..")
		}
		dir = parent
	}
}

// runResult holds the output of a devbox CLI invocation.
type runResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runDevbox executes the devbox binary with the given arguments and working directory.
// It returns the captured output and exit code.
func runDevbox(t *testing.T, workDir string, args ...string) runResult {
	t.Helper()

	fullArgs := append([]string{"--no-color"}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run devbox: %v", err)
		}
	}

	return runResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// sshRun executes a command on the remote server via SSH and returns stdout.
// It fatally aborts on failure — do NOT use inside waitForCondition polling.
func sshRun(t *testing.T, server, command string) string {
	t.Helper()
	out, err := trySshRun(server, command)
	if err != nil {
		t.Fatalf("ssh %s %q failed: %v", server, command, err)
	}
	return out
}

// trySshRun executes a command on the remote server via SSH.
// Returns stdout and error without aborting — safe for use in polling loops.
func trySshRun(server, command string) (string, error) {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		server,
		command,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// requireEnv reads an environment variable or skips the test if unset.
func requireEnv(t *testing.T, key string) string {
	t.Helper()
	val := os.Getenv(key)
	if val == "" {
		t.Skipf("skipping: %s not set", key)
	}
	return val
}

// workspaceDir returns the remote workspace directory path.
func workspaceDir() string {
	dir := os.Getenv("DEVBOX_TEST_WORKSPACE_DIR")
	if dir == "" {
		return "/workspaces"
	}
	return dir
}

// prepareTestDir creates a temporary directory with a devbox.yaml config,
// substituting ${DEVBOX_TEST_SERVER} with the actual server value.
func prepareTestDir(t *testing.T, configFile string, server string) string {
	t.Helper()

	repoRoot := findRepoRoot()
	src := filepath.Join(repoRoot, "tests", "integration", "testdata", configFile)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read %s: %v", src, err)
	}

	content := strings.ReplaceAll(string(data), "${DEVBOX_TEST_SERVER}", server)

	tmpDir, err := os.MkdirTemp("", "devbox-test-project-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write devbox.yaml: %v", err)
	}

	return tmpDir
}

// cleanupWorkspace attempts to destroy a workspace and verify cleanup.
// Used in t.Cleanup() to ensure server state is clean even if a test fails.
func cleanupWorkspace(t *testing.T, workDir, name, server string) {
	t.Helper()

	// Best-effort destroy via CLI.
	cmd := exec.Command(binaryPath, "destroy", name, "--force")
	cmd.Dir = workDir
	_ = cmd.Run()

	// Best-effort remove workspace dir on server.
	wsDir := workspaceDir() + "/" + name
	sshCmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		server,
		fmt.Sprintf("rm -rf '%s'", wsDir),
	)
	_ = sshCmd.Run()
}

// waitForCondition polls a condition function until it returns true or timeout.
func waitForCondition(t *testing.T, timeout time.Duration, interval time.Duration, desc string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatalf("timed out waiting for: %s", desc)
}

// assertContains checks that haystack contains needle.
func assertContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected output to contain %q, got:\n%s", context, needle, haystack)
	}
}

// assertNotContains checks that haystack does not contain needle.
func assertNotContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("%s: expected output NOT to contain %q, got:\n%s", context, needle, haystack)
	}
}

// assertExitCode checks the exit code of a runResult.
func assertExitCode(t *testing.T, result runResult, expected int, context string) {
	t.Helper()
	if result.ExitCode != expected {
		t.Errorf("%s: exit code = %d, want %d\nstdout: %s\nstderr: %s",
			context, result.ExitCode, expected, result.Stdout, result.Stderr)
	}
}
