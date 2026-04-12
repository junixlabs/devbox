package server

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockSSHExecutor simulates SSH commands for testing.
type mockSSHExecutor struct {
	// responses maps "host:command-prefix" to stdout output.
	responses map[string]string
	// errors maps "host" to an error (simulates offline servers).
	errors map[string]error
}

func (m *mockSSHExecutor) Run(_ context.Context, host string, command string) (string, string, error) {
	if err, ok := m.errors[host]; ok {
		return "", "", err
	}
	// Match by host + command prefix.
	for key, resp := range m.responses {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 && parts[0] == host && strings.HasPrefix(command, parts[1]) {
			return resp, "", nil
		}
	}
	return "", "", fmt.Errorf("no mock response for %s: %s", host, command)
}

func (m *mockSSHExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}

func (m *mockSSHExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockSSHExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (m *mockSSHExecutor) Close() error { return nil }

func freeOutput(totalBytes, usedBytes int64) string {
	free := totalBytes - usedBytes
	return fmt.Sprintf(`              total        used        free      shared  buff/cache   available
Mem:     %d  %d  %d   0  0  %d
Swap:    0           0  0`, totalBytes, usedBytes, free, free)
}

func TestLeastLoadedSelector_SelectsLeastLoaded(t *testing.T) {
	mock := &mockSSHExecutor{
		responses: map[string]string{
			// srv1: 4 CPUs, 80% CPU load, 8GB total / 6.4GB used (80%)
			"srv1.example.com:nproc":           "4\n",
			"srv1.example.com:free -b":         freeOutput(8_000_000_000, 6_400_000_000),
			"srv1.example.com:cat /proc/loadavg": "3.20 2.50 2.00 1/200 1234\n", // 3.2/4 = 80%
			// srv2: 4 CPUs, 30% CPU load, 8GB total / 2.4GB used (30%)
			"srv2.example.com:nproc":           "4\n",
			"srv2.example.com:free -b":         freeOutput(8_000_000_000, 2_400_000_000),
			"srv2.example.com:cat /proc/loadavg": "1.20 1.00 0.80 1/200 1234\n", // 1.2/4 = 30%
		},
		errors: map[string]error{},
	}

	servers := []Server{
		{Name: "srv1", Host: "srv1.example.com"},
		{Name: "srv2", Host: "srv2.example.com"},
	}

	selector := NewLeastLoaded(mock)
	selected, err := selector.Select(context.Background(), servers)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected.Name != "srv2" {
		t.Errorf("Select() = %q, want %q (least loaded)", selected.Name, "srv2")
	}
}

func TestLeastLoadedSelector_SkipsOfflineServer(t *testing.T) {
	mock := &mockSSHExecutor{
		responses: map[string]string{
			"srv2.example.com:nproc":           "4\n",
			"srv2.example.com:free -b":         freeOutput(8_000_000_000, 2_000_000_000),
			"srv2.example.com:cat /proc/loadavg": "0.50 0.40 0.30 1/100 1234\n",
		},
		errors: map[string]error{
			"srv1.example.com": fmt.Errorf("connection refused"),
		},
	}

	servers := []Server{
		{Name: "srv1", Host: "srv1.example.com"},
		{Name: "srv2", Host: "srv2.example.com"},
	}

	selector := NewLeastLoaded(mock)
	selected, err := selector.Select(context.Background(), servers)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected.Name != "srv2" {
		t.Errorf("Select() = %q, want %q (only online server)", selected.Name, "srv2")
	}
}

func TestLeastLoadedSelector_AllOffline(t *testing.T) {
	mock := &mockSSHExecutor{
		responses: map[string]string{},
		errors: map[string]error{
			"srv1.example.com": fmt.Errorf("connection refused"),
			"srv2.example.com": fmt.Errorf("timeout"),
		},
	}

	servers := []Server{
		{Name: "srv1", Host: "srv1.example.com"},
		{Name: "srv2", Host: "srv2.example.com"},
	}

	selector := NewLeastLoaded(mock)
	_, err := selector.Select(context.Background(), servers)
	if err == nil {
		t.Fatal("Select() should return error when all servers offline")
	}
	if !strings.Contains(err.Error(), "all servers are offline") {
		t.Errorf("Select() error = %q, want to contain 'all servers are offline'", err.Error())
	}
}

func TestLeastLoadedSelector_EmptyServers(t *testing.T) {
	mock := &mockSSHExecutor{
		responses: map[string]string{},
		errors:    map[string]error{},
	}

	selector := NewLeastLoaded(mock)
	_, err := selector.Select(context.Background(), nil)
	if err == nil {
		t.Fatal("Select() should return error for empty server list")
	}
}

func TestLeastLoadedSelector_DeterministicTieBreak(t *testing.T) {
	// Both servers have identical load — should pick alphabetically first.
	mock := &mockSSHExecutor{
		responses: map[string]string{
			"alpha.example.com:nproc":           "4\n",
			"alpha.example.com:free -b":         freeOutput(8_000_000_000, 4_000_000_000),
			"alpha.example.com:cat /proc/loadavg": "2.00 1.50 1.00 1/200 1234\n",
			"beta.example.com:nproc":            "4\n",
			"beta.example.com:free -b":          freeOutput(8_000_000_000, 4_000_000_000),
			"beta.example.com:cat /proc/loadavg":  "2.00 1.50 1.00 1/200 1234\n",
		},
		errors: map[string]error{},
	}

	servers := []Server{
		{Name: "beta", Host: "beta.example.com"},
		{Name: "alpha", Host: "alpha.example.com"},
	}

	selector := NewLeastLoaded(mock)
	selected, err := selector.Select(context.Background(), servers)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected.Name != "alpha" {
		t.Errorf("Select() = %q, want %q (alphabetical tie-break)", selected.Name, "alpha")
	}
}
