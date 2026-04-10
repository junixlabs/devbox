package ui

import (
	"testing"

	"github.com/fatih/color"
	"github.com/junixlabs/devbox/internal/workspace"
)

func TestStatusColor_NoColor(t *testing.T) {
	SetNoColor(true)
	defer SetNoColor(false)

	tests := []struct {
		status workspace.Status
		want   string
	}{
		{workspace.StatusRunning, "running"},
		{workspace.StatusStopped, "stopped"},
		{workspace.StatusCreating, "creating"},
		{workspace.StatusError, "error"},
		{workspace.Status("unknown"), "unknown"},
	}

	for _, tt := range tests {
		got := StatusColor(tt.status)
		if got != tt.want {
			t.Errorf("StatusColor(%q) with NoColor = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusColor_WithColor(t *testing.T) {
	SetNoColor(false)
	defer SetNoColor(true)

	// With color enabled, output should contain ANSI escape codes
	got := StatusColor(workspace.StatusRunning)
	if got == "running" {
		t.Error("StatusColor(running) with color should contain ANSI codes")
	}
	if len(got) <= len("running") {
		t.Error("StatusColor(running) with color should be longer than plain text")
	}
}

func TestSetNoColor(t *testing.T) {
	SetNoColor(true)
	if !color.NoColor {
		t.Error("SetNoColor(true) should set color.NoColor to true")
	}

	SetNoColor(false)
	if color.NoColor {
		t.Error("SetNoColor(false) should set color.NoColor to false")
	}
}

func TestPrintTable(t *testing.T) {
	// Smoke test — just ensure it doesn't panic
	SetNoColor(true)
	defer SetNoColor(false)

	headers := []string{"NAME", "STATUS", "SERVER"}
	rows := [][]string{
		{"my-ws", StatusColor(workspace.StatusRunning), "server1"},
		{"other-ws", StatusColor(workspace.StatusStopped), "server2"},
	}
	PrintTable(headers, rows)
}

func TestPrintUpSuccess(t *testing.T) {
	// Smoke test — just ensure it doesn't panic
	SetNoColor(true)
	defer SetNoColor(false)

	ports := map[string]int{"app": 8080, "db": 3306}
	PrintUpSuccess("test-ws", "my-server", "https://example.com", ports)
}
