// Package executor abstracts workspace provisioning and lifecycle over
// different runtimes (Docker Compose vs. a bare host shell) behind a single
// interface, so the workspace manager does not need to branch on runtime.
package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/ssh"
)

// PIDReporter is implemented by executors that track a supervised process
// PID (currently only hostExecutor). Callers can type-assert on it to
// surface the PID in workspace state for display purposes.
type PIDReporter interface {
	PID(ctx context.Context) (int, error)
}

// Refresher is implemented by executors that support refreshing an existing
// workspace in place — re-running setup and bouncing the serve process —
// without a full Destroy/Deploy cycle (currently only hostExecutor). Callers
// type-assert on it (like PIDReporter) and fall back to Down+Up when an
// executor doesn't support it.
type Refresher interface {
	// RunSetup re-runs the configured setup commands (e.g. a dependency
	// install) without starting the serve process.
	RunSetup(ctx context.Context) error

	// Restart stops (if running) then relaunches the serve process.
	Restart(ctx context.Context) error
}

// EASBuilder is implemented by executors that can produce a native app build
// via EAS (Expo Application Services) — currently only hostExecutor. Callers
// type-assert on it (like PIDReporter/Refresher) and treat its absence as
// "this runtime does not support builds".
type EASBuilder interface {
	// BuildAndroid runs an EAS Android build for the given profile and
	// returns the installable artifact URL of the produced app.
	BuildAndroid(ctx context.Context, profile string) (string, error)
}

// Executor defines the lifecycle operations a runtime must implement.
type Executor interface {
	// Deploy provisions the workspace for the first time and starts it.
	Deploy(ctx context.Context) error

	// Up starts an existing, stopped workspace.
	Up(ctx context.Context) error

	// Down stops the workspace without discarding its cache/data.
	Down(ctx context.Context) error

	// Logs streams (or dumps) the workspace's logs.
	Logs(ctx context.Context, follow bool, stdout, stderr io.Writer) error

	// Destroy stops the workspace and removes all of its data.
	Destroy(ctx context.Context) error
}

// New selects an Executor implementation based on cfg.Runtime.
// An empty runtime is treated as config.RuntimeDocker.
func New(sshExec ssh.Executor, cfg *config.DevboxConfig, host, name string) (Executor, error) {
	switch cfg.Runtime {
	case "", config.RuntimeDocker:
		return newDockerExecutor(sshExec, cfg, host, name)
	case config.RuntimeHost:
		return newHostExecutor(sshExec, cfg, host, name)
	default:
		return nil, fmt.Errorf("unknown runtime %q", cfg.Runtime)
	}
}
