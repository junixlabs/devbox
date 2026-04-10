package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Executor defines the interface for running commands and transferring files over SSH.
type Executor interface {
	// Run executes a command on the remote host and returns captured output.
	Run(ctx context.Context, host string, command string) (stdout string, stderr string, err error)

	// RunStream executes a command on the remote host, streaming output in real-time.
	RunStream(ctx context.Context, host string, command string, stdout io.Writer, stderr io.Writer) error

	// CopyTo copies a local file to the remote host via scp.
	CopyTo(ctx context.Context, host string, localPath string, remotePath string) error

	// CopyFrom copies a remote file to the local machine via scp.
	CopyFrom(ctx context.Context, host string, remotePath string, localPath string) error

	// Close shuts down ControlMaster connections and cleans up the control socket directory.
	Close() error
}

// Option configures an executor.
type Option func(*executor)

// WithSSHBinary overrides the default ssh binary path.
func WithSSHBinary(path string) Option {
	return func(e *executor) {
		e.sshBinary = path
	}
}

// WithSCPBinary overrides the default scp binary path.
func WithSCPBinary(path string) Option {
	return func(e *executor) {
		e.scpBinary = path
	}
}

type executor struct {
	sshBinary  string
	scpBinary  string
	controlDir string
	hosts      map[string]struct{} // tracks hosts we've connected to
}

// New creates a new SSH Executor. Returns an error if the ssh or scp binaries
// cannot be found in PATH.
func New(opts ...Option) (Executor, error) {
	e := &executor{
		sshBinary: "ssh",
		scpBinary: "scp",
		hosts:     make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(e)
	}

	if _, err := exec.LookPath(e.sshBinary); err != nil {
		return nil, fmt.Errorf("ssh binary not found (%s): %w", e.sshBinary, err)
	}
	if _, err := exec.LookPath(e.scpBinary); err != nil {
		return nil, fmt.Errorf("scp binary not found (%s): %w", e.scpBinary, err)
	}

	dir, err := os.MkdirTemp("/tmp", "devbox-ssh-")
	if err != nil {
		return nil, fmt.Errorf("creating control socket dir: %w", err)
	}
	e.controlDir = dir

	return e, nil
}

// sshArgs returns the base SSH arguments including ControlMaster options.
func (e *executor) sshArgs(host string) []string {
	controlPath := filepath.Join(e.controlDir, "%r@%h:%p")
	return []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + controlPath,
		"-o", "ControlPersist=60",
		host,
	}
}

// scpArgs returns the base SCP arguments including ControlMaster options.
func (e *executor) scpArgs() []string {
	controlPath := filepath.Join(e.controlDir, "%r@%h:%p")
	return []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ControlMaster=auto",
		"-o", "ControlPath=" + controlPath,
		"-o", "ControlPersist=60",
	}
}

func (e *executor) Run(ctx context.Context, host string, command string) (string, string, error) {
	e.hosts[host] = struct{}{}
	args := append(e.sshArgs(host), command)
	cmd := exec.CommandContext(ctx, e.sshBinary, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(),
			fmt.Errorf("ssh command failed on %s: %w\nstderr: %s", host, err, stderrBuf.String())
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}

func (e *executor) RunStream(ctx context.Context, host string, command string, stdout io.Writer, stderr io.Writer) error {
	e.hosts[host] = struct{}{}
	args := append(e.sshArgs(host), command)
	cmd := exec.CommandContext(ctx, e.sshBinary, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh stream command failed on %s: %w", host, err)
	}
	return nil
}

func (e *executor) CopyTo(ctx context.Context, host string, localPath string, remotePath string) error {
	e.hosts[host] = struct{}{}
	args := append(e.scpArgs(), localPath, host+":"+remotePath)
	cmd := exec.CommandContext(ctx, e.scpBinary, args...)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp to %s failed: %w\nstderr: %s", host, err, stderrBuf.String())
	}
	return nil
}

func (e *executor) CopyFrom(ctx context.Context, host string, remotePath string, localPath string) error {
	e.hosts[host] = struct{}{}
	args := append(e.scpArgs(), host+":"+remotePath, localPath)
	cmd := exec.CommandContext(ctx, e.scpBinary, args...)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp from %s failed: %w\nstderr: %s", host, err, stderrBuf.String())
	}
	return nil
}

func (e *executor) Close() error {
	// Send ControlMaster exit signal to each host we've connected to.
	controlPath := filepath.Join(e.controlDir, "%r@%h:%p")
	for host := range e.hosts {
		// #nosec — host is tracked internally, not user input
		cmd := exec.Command(e.sshBinary,
			"-o", "ControlPath="+controlPath,
			"-O", "exit", host,
		)
		_ = cmd.Run() // best-effort; ignore errors (e.g. already closed)
	}
	return os.RemoveAll(e.controlDir)
}
