package ssh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDefaults(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	impl := e.(*executor)
	if impl.sshBinary != "ssh" {
		t.Errorf("sshBinary = %q, want %q", impl.sshBinary, "ssh")
	}
	if impl.scpBinary != "scp" {
		t.Errorf("scpBinary = %q, want %q", impl.scpBinary, "scp")
	}
	if impl.controlDir == "" {
		t.Error("controlDir should not be empty")
	}
	if _, err := os.Stat(impl.controlDir); os.IsNotExist(err) {
		t.Errorf("controlDir %q does not exist", impl.controlDir)
	}
}

func TestNewWithOptions(t *testing.T) {
	// Use actual ssh/scp paths to avoid LookPath errors.
	e, err := New(WithSSHBinary("ssh"), WithSCPBinary("scp"))
	if err != nil {
		t.Fatalf("New() with options failed: %v", err)
	}
	defer e.Close()

	impl := e.(*executor)
	if impl.sshBinary != "ssh" {
		t.Errorf("sshBinary = %q, want %q", impl.sshBinary, "ssh")
	}
	if impl.scpBinary != "scp" {
		t.Errorf("scpBinary = %q, want %q", impl.scpBinary, "scp")
	}
}

func TestNewInvalidBinary(t *testing.T) {
	_, err := New(WithSSHBinary("nonexistent-ssh-binary-12345"))
	if err == nil {
		t.Fatal("expected error for nonexistent ssh binary")
	}
	if !strings.Contains(err.Error(), "ssh binary not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "ssh binary not found")
	}

	_, err = New(WithSCPBinary("nonexistent-scp-binary-12345"))
	if err == nil {
		t.Fatal("expected error for nonexistent scp binary")
	}
	if !strings.Contains(err.Error(), "scp binary not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "scp binary not found")
	}
}

func TestSSHArgs(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	impl := e.(*executor)
	args := impl.sshArgs("devbox-vps")

	// Should contain host as last element.
	if args[len(args)-1] != "devbox-vps" {
		t.Errorf("last arg = %q, want %q", args[len(args)-1], "devbox-vps")
	}

	// Should contain ControlMaster options.
	joined := strings.Join(args, " ")
	for _, want := range []string{"StrictHostKeyChecking=accept-new", "ControlMaster=auto", "ControlPath=", "ControlPersist=60"} {
		if !strings.Contains(joined, want) {
			t.Errorf("sshArgs missing %q in: %s", want, joined)
		}
	}

	// ControlPath should use the controlDir.
	if !strings.Contains(joined, impl.controlDir) {
		t.Errorf("sshArgs ControlPath should use controlDir %q, got: %s", impl.controlDir, joined)
	}
}

func TestSCPArgs(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	impl := e.(*executor)
	args := impl.scpArgs()

	joined := strings.Join(args, " ")
	for _, want := range []string{"StrictHostKeyChecking=accept-new", "ControlMaster=auto", "ControlPath=", "ControlPersist=60"} {
		if !strings.Contains(joined, want) {
			t.Errorf("scpArgs missing %q in: %s", want, joined)
		}
	}
}

func TestRunErrorNonexistentHost(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9) // 5 seconds
	defer cancel()

	_, _, err = e.Run(ctx, "nonexistent-host-12345.invalid", "echo hello")
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if !strings.Contains(err.Error(), "ssh command failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "ssh command failed")
	}
}

func TestRunStreamErrorNonexistentHost(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9)
	defer cancel()

	var stdout, stderr strings.Builder
	err = e.RunStream(ctx, "nonexistent-host-12345.invalid", "echo hello", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if !strings.Contains(err.Error(), "ssh stream command failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "ssh stream command failed")
	}
}

func TestCopyToErrorNonexistentHost(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9)
	defer cancel()

	err = e.CopyTo(ctx, "nonexistent-host-12345.invalid", "/dev/null", "/tmp/test")
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if !strings.Contains(err.Error(), "scp to") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "scp to")
	}
}

func TestCopyFromErrorNonexistentHost(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer e.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*1e9)
	defer cancel()

	err = e.CopyFrom(ctx, "nonexistent-host-12345.invalid", "/tmp/test", filepath.Join(t.TempDir(), "out"))
	if err == nil {
		t.Fatal("expected error for nonexistent host")
	}
	if !strings.Contains(err.Error(), "scp from") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "scp from")
	}
}

func TestClose(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	impl := e.(*executor)
	dir := impl.controlDir

	// Verify dir exists before close.
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("controlDir %q should exist before Close", dir)
	}

	// Create a dummy file to simulate a socket.
	dummySocket := filepath.Join(dir, "test@host:22")
	if err := os.WriteFile(dummySocket, []byte(""), 0600); err != nil {
		t.Fatalf("failed to create dummy socket: %v", err)
	}

	if err := e.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	// Verify dir is removed after close.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("controlDir %q should be removed after Close", dir)
	}
}
