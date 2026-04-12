package metrics

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockExecutor implements ssh.Executor for testing.
type mockExecutor struct {
	// responses maps command substrings to (stdout, stderr, error).
	responses map[string]mockResponse
}

type mockResponse struct {
	stdout string
	stderr string
	err    error
}

func (m *mockExecutor) Run(_ context.Context, _ string, command string) (string, string, error) {
	for key, resp := range m.responses {
		if strings.Contains(command, key) {
			return resp.stdout, resp.stderr, resp.err
		}
	}
	return "", "", fmt.Errorf("no mock response for command: %s", command)
}

func (m *mockExecutor) RunStream(_ context.Context, _ string, _ string, _ io.Writer, _ io.Writer) error {
	return nil
}

func (m *mockExecutor) CopyTo(_ context.Context, _ string, _ string, _ string) error { return nil }
func (m *mockExecutor) CopyFrom(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *mockExecutor) Close() error { return nil }

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
	}{
		{"100B", 100},
		{"1kB", 1000},
		{"1.5kB", 1500},
		{"256MiB", 256 * 1024 * 1024},
		{"1.5GiB", uint64(1.5 * 1024 * 1024 * 1024)},
		{"2GB", 2_000_000_000},
		{"500MB", 500_000_000},
		{"1KiB", 1024},
		{"", 0},
		{"invalid", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseByteSize(tt.input)
			if got != tt.want {
				t.Errorf("ParseByteSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDockerStatsJSON(t *testing.T) {
	line := `{"Name":"myapp-web-1","CPUPerc":"1.23%","MemPerc":"15.50%","MemUsage":"256MiB / 2GiB","NetIO":"1.5kB / 2.3MB","BlockIO":"0B / 0B"}`

	wm, err := parseDockerStatsJSON(line)
	if err != nil {
		t.Fatalf("parseDockerStatsJSON failed: %v", err)
	}

	if wm.Container != "myapp-web-1" {
		t.Errorf("Container = %q, want %q", wm.Container, "myapp-web-1")
	}
	if wm.CPUPercent != 1.23 {
		t.Errorf("CPUPercent = %f, want 1.23", wm.CPUPercent)
	}
	if wm.MemUsage != 256*1024*1024 {
		t.Errorf("MemUsage = %d, want %d", wm.MemUsage, 256*1024*1024)
	}
	if wm.MemLimit != 2*1024*1024*1024 {
		t.Errorf("MemLimit = %d, want %d", wm.MemLimit, 2*1024*1024*1024)
	}
	if wm.NetIn != 1500 {
		t.Errorf("NetIn = %d, want 1500", wm.NetIn)
	}
	if wm.NetOut != 2_300_000 {
		t.Errorf("NetOut = %d, want 2300000", wm.NetOut)
	}
}

func TestCollectWorkspace(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"docker stats myapp-web-1": {
				stdout: `{"Name":"myapp-web-1","CPUPerc":"2.50%","MemPerc":"10.00%","MemUsage":"512MiB / 4GiB","NetIO":"100kB / 200kB","BlockIO":"0B / 0B"}`,
			},
			"docker exec myapp-web-1 df": {
				stdout: "/dev/sda1 107374182400 21474836480 85899345920 20% /",
			},
		},
	}

	c := NewCollector(mock)
	wm, err := c.CollectWorkspace(context.Background(), "server1", "myapp-web-1")
	if err != nil {
		t.Fatalf("CollectWorkspace failed: %v", err)
	}
	if wm.Stopped {
		t.Error("expected workspace to not be stopped")
	}
	if wm.CPUPercent != 2.5 {
		t.Errorf("CPUPercent = %f, want 2.5", wm.CPUPercent)
	}
	if wm.DiskUsage != 21474836480 {
		t.Errorf("DiskUsage = %d, want 21474836480", wm.DiskUsage)
	}
	if wm.DiskTotal != 107374182400 {
		t.Errorf("DiskTotal = %d, want 107374182400", wm.DiskTotal)
	}
}

func TestCollectWorkspace_Stopped(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"docker stats": {err: fmt.Errorf("container not running")},
		},
	}

	c := NewCollector(mock)
	wm, err := c.CollectWorkspace(context.Background(), "server1", "stopped-container")
	if err != nil {
		t.Fatalf("CollectWorkspace should not error for stopped container: %v", err)
	}
	if !wm.Stopped {
		t.Error("expected workspace to be stopped")
	}
	if wm.CPUPercent != 0 {
		t.Errorf("CPUPercent should be 0 for stopped container, got %f", wm.CPUPercent)
	}
}

func TestCollectServer(t *testing.T) {
	statsLine1 := `{"Name":"app-web-1","CPUPerc":"1.00%","MemPerc":"5.00%","MemUsage":"128MiB / 2GiB","NetIO":"10kB / 20kB","BlockIO":"0B / 0B"}`
	statsLine2 := `{"Name":"app-db-1","CPUPerc":"3.00%","MemPerc":"10.00%","MemUsage":"512MiB / 2GiB","NetIO":"50kB / 100kB","BlockIO":"0B / 0B"}`

	combined := fmt.Sprintf("%s\n%s\n===METRICS_SEP===\n4\n===METRICS_SEP===\nMemTotal:        8192000 kB\nMemAvailable:    4096000 kB\n===METRICS_SEP===\n/dev/sda1 107374182400 53687091200 53687091200 50%% /",
		statsLine1, statsLine2)

	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"docker stats": {stdout: combined},
		},
	}

	c := NewCollector(mock)
	sm, err := c.CollectServer(context.Background(), "server1")
	if err != nil {
		t.Fatalf("CollectServer failed: %v", err)
	}

	if sm.TotalCPUs != 4 {
		t.Errorf("TotalCPUs = %d, want 4", sm.TotalCPUs)
	}
	if sm.TotalMem != 8192000*1024 {
		t.Errorf("TotalMem = %d, want %d", sm.TotalMem, 8192000*1024)
	}
	if sm.UsedMem != (8192000-4096000)*1024 {
		t.Errorf("UsedMem = %d, want %d", sm.UsedMem, (8192000-4096000)*1024)
	}
	if sm.TotalDisk != 107374182400 {
		t.Errorf("TotalDisk = %d, want 107374182400", sm.TotalDisk)
	}
	if sm.UsedDisk != 53687091200 {
		t.Errorf("UsedDisk = %d, want 53687091200", sm.UsedDisk)
	}
	if len(sm.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(sm.Workspaces))
	}
	if sm.Workspaces[0].Container != "app-web-1" {
		t.Errorf("Workspace[0].Container = %q, want %q", sm.Workspaces[0].Container, "app-web-1")
	}
}

func TestCollectServer_SSHError(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"docker stats": {err: fmt.Errorf("ssh connection refused")},
		},
	}

	c := NewCollector(mock)
	_, err := c.CollectServer(context.Background(), "server1")
	if err == nil {
		t.Fatal("expected error from CollectServer with SSH failure")
	}
}

func TestParseMeminfo(t *testing.T) {
	input := `MemTotal:       16384000 kB
MemFree:         2048000 kB
MemAvailable:    8192000 kB
Buffers:          512000 kB
Cached:          4096000 kB`

	total, used := parseMeminfo(input)
	wantTotal := uint64(16384000 * 1024)
	wantUsed := uint64((16384000 - 8192000) * 1024)
	if total != wantTotal {
		t.Errorf("total = %d, want %d", total, wantTotal)
	}
	if used != wantUsed {
		t.Errorf("used = %d, want %d", used, wantUsed)
	}
}

func TestParseDfBytes(t *testing.T) {
	output := "/dev/sda1 107374182400 21474836480 85899345920 20% /"
	used, total := parseDfBytes(output)
	if total != 107374182400 {
		t.Errorf("total = %d, want 107374182400", total)
	}
	if used != 21474836480 {
		t.Errorf("used = %d, want 21474836480", used)
	}
}

func TestCollectWorkspace_InvalidContainerName(t *testing.T) {
	c := NewCollector(&mockExecutor{responses: map[string]mockResponse{}})

	tests := []string{"foo; rm -rf /", "$(whoami)", "a b", "", "x"}
	for _, name := range tests {
		_, err := c.CollectWorkspace(context.Background(), "server1", name)
		if err == nil {
			t.Errorf("expected error for invalid container name %q", name)
		}
	}
}

func TestCollectWorkspace_JSONParseError(t *testing.T) {
	mock := &mockExecutor{
		responses: map[string]mockResponse{
			"docker stats badjson": {stdout: "not-json-at-all"},
		},
	}

	c := NewCollector(mock)
	_, err := c.CollectWorkspace(context.Background(), "server1", "badjson-container")
	if err == nil {
		t.Fatal("expected error for corrupt docker stats output")
	}
}

func TestParseDfBytes_Empty(t *testing.T) {
	used, total := parseDfBytes("")
	if used != 0 || total != 0 {
		t.Errorf("expected (0, 0) for empty input, got (%d, %d)", used, total)
	}
}
