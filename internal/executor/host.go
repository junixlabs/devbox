package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/junixlabs/devbox/internal/config"
	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"github.com/junixlabs/devbox/internal/ssh"
)

// validEnvKey matches safe shell environment variable names.
var validEnvKey = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// errNoPID indicates the serve process has no PID file recorded (e.g. it was
// never started, or was already stopped) — as opposed to an error reaching
// the host at all, which callers must not treat the same way.
var errNoPID = errors.New("no serve PID recorded")

// hostExecutor runs setup + a long-lived serve process directly on the
// remote host's shell (no container). Because ssh.Executor.Run is one-shot,
// the serve process is launched detached (setsid + redirected log file) so
// it survives the SSH session ending, with its PID tracked in a PID file.
type hostExecutor struct {
	ssh     ssh.Executor
	host    string
	name    string
	workdir string
	srcDir  string
	logFile string
	pidFile string
	setup   []string
	serve   string
	env     map[string]string
	appDir  string
}

// validAppDir matches a safe relative subdirectory (no shell metachars). The
// no-`..` check is enforced separately.
var validAppDir = regexp.MustCompile(`^[a-zA-Z0-9._][a-zA-Z0-9._/-]*$`)

func newHostExecutor(sshExec ssh.Executor, cfg *config.DevboxConfig, host, name string) (Executor, error) {
	if cfg.AppDir != "" && (!validAppDir.MatchString(cfg.AppDir) || strings.Contains(cfg.AppDir, "..")) {
		return nil, devboxerr.NewConfigError(
			fmt.Sprintf("invalid appDir %q", cfg.AppDir),
			"appDir must be a relative path (letters, digits, ., _, -, /) with no '..'",
			nil,
		)
	}
	workdir := cfg.WorkspacesRoot + "/" + name
	return &hostExecutor{
		ssh:     sshExec,
		host:    host,
		name:    name,
		workdir: workdir,
		srcDir:  workdir + "/src",
		logFile: workdir + "/serve.log",
		pidFile: workdir + "/serve.pid",
		setup:   cfg.Setup,
		serve:   cfg.Serve,
		env:     cfg.Env,
		appDir:  cfg.AppDir,
	}, nil
}

// runDir is the directory host commands (setup, serve, build) execute in: the
// cloned src root, or a subdirectory of it when appDir is set (monorepo apps).
func (h *hostExecutor) runDir() string {
	if h.appDir != "" {
		return h.srcDir + "/" + h.appDir
	}
	return h.srcDir
}

// PID reads the PID of the running serve process from its PID file. Callers
// (workspace.Manager) can track it in workspace state for display purposes.
//
// The read command always exits 0 (`|| true`) so a missing PID file — the
// normal case when the workspace was never started or is already stopped —
// surfaces as errNoPID, not a transport error. A non-nil error that is NOT
// errNoPID means the host could not be reached at all, and callers must
// propagate it rather than treating it as "not running".
func (h *hostExecutor) PID(ctx context.Context) (int, error) {
	stdout, _, err := h.ssh.Run(ctx, h.host, fmt.Sprintf("cat %s 2>/dev/null || true", h.pidFile))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		return 0, errNoPID
	}
	return pid, nil
}

func (h *hostExecutor) Deploy(ctx context.Context) error {
	if h.serve == "" {
		return devboxerr.NewConfigError(
			fmt.Sprintf("workspace %q has no 'serve' command configured", h.name),
			"Add 'serve: <long-running command>' to devbox.yaml",
			nil,
		)
	}

	if _, _, err := h.ssh.Run(ctx, h.host, fmt.Sprintf("mkdir -p %s", h.srcDir)); err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to create workspace directory %s on %s", h.workdir, h.host),
			fmt.Sprintf("Check SSH access: ssh %s", h.host),
			err,
		)
	}

	if err := h.RunSetup(ctx); err != nil {
		return err
	}

	return h.startServe(ctx)
}

// RunSetup (re-)runs the configured setup commands (e.g. a dependency
// install) in srcDir, without starting the serve process. It is safe to call
// again on an already-deployed workspace — the setup commands themselves
// (npm ci, expo install, etc.) are idempotent — which is what lets Refresh
// re-run it after a lockfile change without a full Destroy+Deploy cycle.
func (h *hostExecutor) RunSetup(ctx context.Context) error {
	exports := h.exportPrefix()
	for _, cmd := range h.setup {
		full := fmt.Sprintf("cd %s && %s%s", h.runDir(), exports, cmd)
		if _, stderr, err := h.ssh.Run(ctx, h.host, full); err != nil {
			return devboxerr.NewConnectionError(
				fmt.Sprintf("setup command %q failed on %s\nstderr: %s", cmd, h.host, stderr),
				"Check the setup command in devbox.yaml runs cleanly on the target host",
				err,
			)
		}
	}
	return nil
}

func (h *hostExecutor) Up(ctx context.Context) error {
	alive, err := h.isAlive(ctx)
	if err != nil {
		return err
	}
	if alive {
		return nil // already running
	}
	return h.startServe(ctx)
}

// Restart stops the serve process (if running) then relaunches it. Down is
// already idempotent (a no-op when no PID is recorded), so Restart works
// whether or not the process is currently alive.
func (h *hostExecutor) Restart(ctx context.Context) error {
	if err := h.Down(ctx); err != nil {
		return err
	}
	return h.startServe(ctx)
}

// validProfile guards the EAS build profile name, which is interpolated into
// a shell command.
var validProfile = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// BuildAndroid runs an EAS Android build in the workspace's checked-out source
// and returns the installable artifact URL. It runs `eas build` non-interactively
// (authenticated by EAS_TOKEN from the workspace env) with --wait so the cloud
// build completes, and --json so the artifact URL is machine-readable.
//
// NOTE: live EAS execution is not exercised in unit tests (it needs a real EAS
// account, token, and eas.json profile); the artifact-URL parsing is covered by
// parseEASBuildURL's tests. iOS builds require a macOS host and are out of scope.
func (h *hostExecutor) BuildAndroid(ctx context.Context, profile string) (string, error) {
	if profile == "" {
		profile = "preview"
	}
	if !validProfile.MatchString(profile) {
		return "", devboxerr.NewConfigError(
			fmt.Sprintf("invalid EAS build profile %q", profile),
			"Profile names may contain only letters, digits, dots, underscores, and hyphens",
			nil,
		)
	}
	exports := h.exportPrefix()
	cmd := fmt.Sprintf(
		"cd %s && %snpx --yes eas-cli build --platform android --profile %s --non-interactive --json --wait",
		h.runDir(), exports, profile,
	)
	stdout, stderr, err := h.ssh.Run(ctx, h.host, cmd)
	if err != nil {
		return "", devboxerr.NewConnectionError(
			fmt.Sprintf("eas build failed for %s on %s\nstderr: %s", h.name, h.host, stderr),
			"Ensure EAS_TOKEN is set in the workspace env and eas.json defines the build profile",
			err,
		)
	}
	url := parseEASBuildURL([]byte(stdout))
	if url == "" {
		return "", devboxerr.NewConfigError(
			fmt.Sprintf("eas build for %s produced no installable artifact URL", h.name),
			"Check the EAS build profile produces an Android artifact (internal distribution)",
			nil,
		)
	}
	return url, nil
}

// startServe launches the serve command detached via setsid, redirecting
// output to logFile and recording the PID in pidFile.
//
// exports is applied in the outer shell — same as the setup commands in
// Deploy — rather than nested inside the inner `bash -c` string. Nesting
// shellQuote'd exports inside a second single-quoted bash -c would corrupt
// the invocation for any env value containing a space/quote/metachar,
// because the outer single quotes end at the first embedded quote. The
// inner bash -c only ever wraps the fixed "exec <serve>" string, which is
// itself shell-quoted, so it is safe regardless of what serve contains.
func (h *hostExecutor) startServe(ctx context.Context) error {
	exports := h.exportPrefix()
	launch := fmt.Sprintf(
		"cd %s && %ssetsid bash -c %s >%s 2>&1 </dev/null & echo $! >%s",
		h.runDir(), exports, shellQuote("exec "+h.serve), h.logFile, h.pidFile,
	)
	if _, stderr, err := h.ssh.Run(ctx, h.host, launch); err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to start serve process for %s on %s\nstderr: %s", h.name, h.host, stderr),
			fmt.Sprintf("Check the 'serve' command in devbox.yaml runs on %s", h.host),
			err,
		)
	}

	// Confirm the process actually stayed up — a serve that exits right after
	// launching (e.g. Expo hitting a port conflict and skipping in
	// non-interactive mode) must be reported as a failure, not "running".
	check := fmt.Sprintf("sleep 2; if kill -0 $(cat %s 2>/dev/null) 2>/dev/null; then echo alive; fi", h.pidFile)
	out, _, _ := h.ssh.Run(ctx, h.host, check)
	if strings.TrimSpace(out) != "alive" {
		logTail, _, _ := h.ssh.Run(ctx, h.host, fmt.Sprintf("tail -n 15 %s 2>/dev/null", h.logFile))
		return devboxerr.NewConfigError(
			fmt.Sprintf("serve process for %s exited right after starting on %s\nrecent logs:\n%s", h.name, h.host, strings.TrimSpace(logTail)),
			"Check the serve command and its port in devbox.yaml (another process may already hold the port)",
			nil,
		)
	}
	return nil
}

func (h *hostExecutor) Down(ctx context.Context) error {
	pid, err := h.PID(ctx)
	if errors.Is(err, errNoPID) {
		return nil // not running — nothing to stop
	}
	if err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to check serve process state for %s on %s", h.name, h.host),
			fmt.Sprintf("Check SSH connectivity: ssh %s", h.host),
			err,
		)
	}

	// setsid makes the serve process its own process group leader (PID == PGID),
	// so killing the negative PID terminates the whole process tree.
	killCmd := fmt.Sprintf("kill -TERM -- -%d 2>/dev/null; rm -f %s", pid, h.pidFile)
	if _, _, err := h.ssh.Run(ctx, h.host, killCmd); err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to stop serve process (pid %d) for %s on %s", pid, h.name, h.host),
			fmt.Sprintf("Check SSH access: ssh %s", h.host),
			err,
		)
	}
	return nil
}

func (h *hostExecutor) Logs(ctx context.Context, follow bool, stdout, stderr io.Writer) error {
	var cmd string
	if follow {
		cmd = fmt.Sprintf("tail -n +1 -f %s", h.logFile)
	} else {
		cmd = fmt.Sprintf("cat %s", h.logFile)
	}
	if err := h.ssh.RunStream(ctx, h.host, cmd, stdout, stderr); err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to read serve logs for %s on %s", h.name, h.host),
			"Check that the workspace has been deployed: devbox up",
			err,
		)
	}
	return nil
}

func (h *hostExecutor) Destroy(ctx context.Context) error {
	if err := h.Down(ctx); err != nil {
		return err
	}
	if _, _, err := h.ssh.Run(ctx, h.host, fmt.Sprintf("rm -rf %s", h.workdir)); err != nil {
		return devboxerr.NewConnectionError(
			fmt.Sprintf("failed to remove workspace directory %s on %s", h.workdir, h.host),
			fmt.Sprintf("Check SSH access: ssh %s", h.host),
			err,
		)
	}
	return nil
}

// isAlive checks whether the serve process is still running via kill -0.
func (h *hostExecutor) isAlive(ctx context.Context) (bool, error) {
	pid, err := h.PID(ctx)
	if errors.Is(err, errNoPID) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_, _, err = h.ssh.Run(ctx, h.host, fmt.Sprintf("kill -0 %d 2>/dev/null", pid))
	return err == nil, nil
}

// exportPrefix builds a shell-safe "export K=V; ..." prefix for setup/serve
// commands, sorted for deterministic output. Keys are validated; values are
// single-quote shell-escaped.
func (h *hostExecutor) exportPrefix() string {
	if len(h.env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h.env))
	for k := range h.env {
		if validEnvKey.MatchString(k) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString("export ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(shellQuote(h.env[k]))
		b.WriteString("; ")
	}
	return b.String()
}

// shellQuote wraps a string in single quotes, safely escaping any embedded
// single quotes for use in a POSIX shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
