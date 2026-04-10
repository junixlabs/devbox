package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/ssh"
)

// CheckResult holds the outcome of a single health check.
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
	Fix     string
}

// checkTimeout is the per-check timeout to prevent hangs.
const checkTimeout = 10 * time.Second

// Run executes all health checks and prints a formatted report.
// Returns true if all checks passed.
func Run(ctx context.Context, w io.Writer, sshExec ssh.Executor, host string) bool {
	checks := []CheckResult{
		checkGit(),
		checkTailscaleLocal(ctx),
		checkSSH(ctx, sshExec, host),
		checkDockerOnServer(ctx, sshExec, host),
		checkTailscaleOnServer(ctx, sshExec, host),
		checkDiskSpace(ctx, sshExec, host),
	}

	allPassed := true
	for _, c := range checks {
		if c.Passed {
			fmt.Fprintf(w, "  ✓ %s — %s\n", c.Name, c.Message)
		} else {
			fmt.Fprintf(w, "  ✗ %s — %s\n", c.Name, c.Message)
			fmt.Fprintf(w, "    Fix: %s\n", c.Fix)
			allPassed = false
		}
	}

	fmt.Fprintln(w)
	if allPassed {
		fmt.Fprintln(w, "All checks passed")
	} else {
		fmt.Fprintln(w, "Some checks failed — see fix suggestions above")
	}

	return allPassed
}

func checkGit() CheckResult {
	path, err := exec.LookPath("git")
	if err != nil {
		return CheckResult{
			Name:    "Git",
			Passed:  false,
			Message: "git not found in PATH",
			Fix:     "Install git: sudo apt install git",
		}
	}
	return CheckResult{
		Name:    "Git",
		Passed:  true,
		Message: fmt.Sprintf("found at %s", path),
	}
}

func checkTailscaleLocal(ctx context.Context) CheckResult {
	_, err := exec.LookPath("tailscale")
	if err != nil {
		return CheckResult{
			Name:    "Tailscale (local)",
			Passed:  false,
			Message: "tailscale not found in PATH",
			Fix:     "Install Tailscale: https://tailscale.com/download — then run: tailscale up",
		}
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "tailscale", "status", "--json").Output()
	if err != nil {
		return CheckResult{
			Name:    "Tailscale (local)",
			Passed:  false,
			Message: fmt.Sprintf("failed to get status: %v", err),
			Fix:     "Ensure Tailscale is running: sudo tailscale up",
		}
	}

	var status struct {
		BackendState string `json:"BackendState"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return CheckResult{
			Name:    "Tailscale (local)",
			Passed:  false,
			Message: fmt.Sprintf("failed to parse status: %v", err),
			Fix:     "Ensure Tailscale is running: sudo tailscale up",
		}
	}

	if status.BackendState != "Running" {
		return CheckResult{
			Name:    "Tailscale (local)",
			Passed:  false,
			Message: fmt.Sprintf("backend state: %s", status.BackendState),
			Fix:     "Connect Tailscale: sudo tailscale up",
		}
	}

	return CheckResult{
		Name:    "Tailscale (local)",
		Passed:  true,
		Message: "connected",
	}
}

func checkSSH(ctx context.Context, sshExec ssh.Executor, host string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	stdout, _, err := sshExec.Run(ctx, host, "echo ok")
	if err != nil {
		return CheckResult{
			Name:    "SSH connectivity",
			Passed:  false,
			Message: fmt.Sprintf("cannot reach %s: %v", host, err),
			Fix:     "Ensure SSH key is configured and server is reachable: ssh " + host,
		}
	}

	if strings.TrimSpace(stdout) != "ok" {
		return CheckResult{
			Name:    "SSH connectivity",
			Passed:  false,
			Message: fmt.Sprintf("unexpected response from %s", host),
			Fix:     "Ensure SSH key is configured and server is reachable: ssh " + host,
		}
	}

	return CheckResult{
		Name:    "SSH connectivity",
		Passed:  true,
		Message: fmt.Sprintf("connected to %s", host),
	}
}

func checkDockerOnServer(ctx context.Context, sshExec ssh.Executor, host string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	stdout, _, err := sshExec.Run(ctx, host, "docker info --format '{{.ServerVersion}}'")
	if err != nil {
		return CheckResult{
			Name:    "Docker (server)",
			Passed:  false,
			Message: fmt.Sprintf("docker not available on %s: %v", host, err),
			Fix:     "Install Docker on server: curl -fsSL https://get.docker.com | sh",
		}
	}

	version := strings.TrimSpace(stdout)
	if version == "" {
		return CheckResult{
			Name:    "Docker (server)",
			Passed:  false,
			Message: "docker returned empty version",
			Fix:     "Ensure Docker daemon is running: sudo systemctl start docker",
		}
	}

	return CheckResult{
		Name:    "Docker (server)",
		Passed:  true,
		Message: fmt.Sprintf("version %s on %s", version, host),
	}
}

func checkTailscaleOnServer(ctx context.Context, sshExec ssh.Executor, host string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	stdout, _, err := sshExec.Run(ctx, host, "tailscale status --json")
	if err != nil {
		return CheckResult{
			Name:    "Tailscale (server)",
			Passed:  false,
			Message: fmt.Sprintf("tailscale not available on %s: %v", host, err),
			Fix:     "Install Tailscale on server and run: tailscale up",
		}
	}

	var status struct {
		BackendState string `json:"BackendState"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		return CheckResult{
			Name:    "Tailscale (server)",
			Passed:  false,
			Message: fmt.Sprintf("failed to parse tailscale status on %s", host),
			Fix:     "Ensure Tailscale is running on server: sudo tailscale up",
		}
	}

	if status.BackendState != "Running" {
		return CheckResult{
			Name:    "Tailscale (server)",
			Passed:  false,
			Message: fmt.Sprintf("backend state on %s: %s", host, status.BackendState),
			Fix:     "Connect Tailscale on server: sudo tailscale up",
		}
	}

	return CheckResult{
		Name:    "Tailscale (server)",
		Passed:  true,
		Message: fmt.Sprintf("connected on %s", host),
	}
}

func checkDiskSpace(ctx context.Context, sshExec ssh.Executor, host string) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	stdout, _, err := sshExec.Run(ctx, host, "df / --output=pcent | tail -1")
	if err != nil {
		return CheckResult{
			Name:    "Disk space (server)",
			Passed:  false,
			Message: fmt.Sprintf("failed to check disk on %s: %v", host, err),
			Fix:     "Ensure server is reachable and df command is available",
		}
	}

	pctStr := strings.TrimSpace(stdout)
	pctStr = strings.TrimSuffix(pctStr, "%")
	pct, err := strconv.Atoi(strings.TrimSpace(pctStr))
	if err != nil {
		return CheckResult{
			Name:    "Disk space (server)",
			Passed:  false,
			Message: fmt.Sprintf("failed to parse disk usage on %s: %q", host, stdout),
			Fix:     "Check disk manually: ssh " + host + " df -h /",
		}
	}

	if pct > 90 {
		return CheckResult{
			Name:    "Disk space (server)",
			Passed:  false,
			Message: fmt.Sprintf("%d%% used on %s", pct, host),
			Fix:     "Free up disk space on server — currently above 90% usage",
		}
	}

	return CheckResult{
		Name:    "Disk space (server)",
		Passed:  true,
		Message: fmt.Sprintf("%d%% used on %s", pct, host),
	}
}
