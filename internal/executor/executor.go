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
