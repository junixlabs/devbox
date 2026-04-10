package doctor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockExecutor implements ssh.Executor for testing.
type mockExecutor struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	stdout string
	stderr string
	err    error
}

func (m *mockExecutor) Run(_ context.Context, host string, command string) (string, string, error) {
	key := host + ":" + command
	if r, ok := m.responses[key]; ok {
		return r.stdout, r.stderr, r.err
	}
	return "", "", fmt.Errorf("unexpected command: %s", command)
}

func (m *mockExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}

func (m *mockExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockExecutor) Close() error {
	return nil
}

func TestCheckSSH_Pass(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:echo ok": {stdout: "ok\n"},
		},
	}

	result := checkSSH(context.Background(), mock, "server1")
	if !result.Passed {
		t.Errorf("expected SSH check to pass, got: %s", result.Message)
	}
}

func TestCheckSSH_Fail(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:echo ok": {err: fmt.Errorf("connection refused")},
		},
	}

	result := checkSSH(context.Background(), mock, "server1")
	if result.Passed {
		t.Error("expected SSH check to fail")
	}
	if result.Fix == "" {
		t.Error("expected fix suggestion")
	}
}

func TestCheckDockerOnServer_Pass(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:docker info --format '{{.ServerVersion}}'": {stdout: "24.0.7\n"},
		},
	}

	result := checkDockerOnServer(context.Background(), mock, "server1")
	if !result.Passed {
		t.Errorf("expected Docker check to pass, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "24.0.7") {
		t.Errorf("expected version in message, got: %s", result.Message)
	}
}

func TestCheckDockerOnServer_Fail(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:docker info --format '{{.ServerVersion}}'": {err: fmt.Errorf("command not found: docker")},
		},
	}

	result := checkDockerOnServer(context.Background(), mock, "server1")
	if result.Passed {
		t.Error("expected Docker check to fail")
	}
}

func TestCheckTailscaleOnServer_Pass(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:tailscale status --json": {stdout: `{"BackendState":"Running"}`},
		},
	}

	result := checkTailscaleOnServer(context.Background(), mock, "server1")
	if !result.Passed {
		t.Errorf("expected Tailscale server check to pass, got: %s", result.Message)
	}
}

func TestCheckTailscaleOnServer_NotRunning(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:tailscale status --json": {stdout: `{"BackendState":"Stopped"}`},
		},
	}

	result := checkTailscaleOnServer(context.Background(), mock, "server1")
	if result.Passed {
		t.Error("expected Tailscale server check to fail when stopped")
	}
}

func TestCheckDiskSpace_Pass(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:df / --output=pcent | tail -1": {stdout: " 42%\n"},
		},
	}

	result := checkDiskSpace(context.Background(), mock, "server1")
	if !result.Passed {
		t.Errorf("expected disk check to pass, got: %s", result.Message)
	}
	if !strings.Contains(result.Message, "42%") {
		t.Errorf("expected percentage in message, got: %s", result.Message)
	}
}

func TestCheckDiskSpace_Fail(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:df / --output=pcent | tail -1": {stdout: " 95%\n"},
		},
	}

	result := checkDiskSpace(context.Background(), mock, "server1")
	if result.Passed {
		t.Error("expected disk check to fail at 95%")
	}
	if !strings.Contains(result.Message, "95%") {
		t.Errorf("expected percentage in message, got: %s", result.Message)
	}
}

func TestRun_AllPass(t *testing.T) {
	// Note: Run also calls checkGit and checkTailscaleLocal which use real exec.LookPath.
	// We test the remote checks individually above. Here we just verify Run doesn't panic
	// and returns a bool.
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:echo ok":                                   {stdout: "ok\n"},
			"server1:docker info --format '{{.ServerVersion}}'": {stdout: "24.0.7\n"},
			"server1:tailscale status --json":                   {stdout: `{"BackendState":"Running"}`},
			"server1:df / --output=pcent | tail -1":             {stdout: " 42%\n"},
		},
	}

	var buf bytes.Buffer
	_ = Run(context.Background(), &buf, mock, "server1")

	output := buf.String()
	if !strings.Contains(output, "Git") {
		t.Error("expected Git check in output")
	}
	if !strings.Contains(output, "SSH connectivity") {
		t.Error("expected SSH check in output")
	}
	if !strings.Contains(output, "Docker") {
		t.Error("expected Docker check in output")
	}
}

func TestRun_OutputFormat(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"server1:echo ok":                                   {err: fmt.Errorf("connection refused")},
			"server1:docker info --format '{{.ServerVersion}}'": {err: fmt.Errorf("not found")},
			"server1:tailscale status --json":                   {err: fmt.Errorf("not found")},
			"server1:df / --output=pcent | tail -1":             {err: fmt.Errorf("not found")},
		},
	}

	var buf bytes.Buffer
	allPassed := Run(context.Background(), &buf, mock, "server1")

	if allPassed {
		t.Error("expected allPassed to be false when checks fail")
	}

	output := buf.String()
	if !strings.Contains(output, "✗") {
		t.Error("expected ✗ in output for failed checks")
	}
	if !strings.Contains(output, "Fix:") {
		t.Error("expected fix suggestions in output")
	}
	if !strings.Contains(output, "Some checks failed") {
		t.Error("expected failure summary in output")
	}
}
