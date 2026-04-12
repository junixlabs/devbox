package identity

import (
	"errors"
	"testing"

	"github.com/junixlabs/devbox/internal/tailscale"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"alice", "alice"},
		{"Alice", "alice"},
		{"user.name", "user-name"},
		{"user_name", "user-name"},
		{"user@name", "username"},
		{"user--name", "user-name"},
		{"-leading-", "leading"},
		{"UPPER.Case_Mix", "upper-case-mix"},
		{"a!b#c$d", "abcd"},
		{"", ""},
	}
	for _, tt := range tests {
		got := Sanitize(tt.input)
		if got != tt.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestUsernameFromLogin(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"dev@example.com", "dev"},
		{"user.name@company.co", "user-name"},
		{"plainuser", "plainuser"},
		{"UPPER@DOMAIN.COM", "upper"},
	}
	for _, tt := range tests {
		got := UsernameFromLogin(tt.input)
		if got != tt.want {
			t.Errorf("UsernameFromLogin(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

type mockTSManager struct {
	status *tailscale.StatusInfo
	err    error
}

func (m *mockTSManager) Serve(port int, hostname string) error { return nil }
func (m *mockTSManager) Unserve(port int) error                { return nil }
func (m *mockTSManager) Status() (*tailscale.StatusInfo, error) {
	return m.status, m.err
}

func TestResolver_Tailscale(t *testing.T) {
	ts := &mockTSManager{
		status: &tailscale.StatusInfo{UserLogin: "dev@example.com"},
	}
	r := NewResolver(ts)
	id, err := r.Current()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "dev" {
		t.Errorf("Username = %q, want %q", id.Username, "dev")
	}
	if id.Source != SourceTailscale {
		t.Errorf("Source = %q, want %q", id.Source, SourceTailscale)
	}
}

func TestResolver_EnvFallback(t *testing.T) {
	ts := &mockTSManager{err: errors.New("not connected")}
	r := NewResolver(ts)
	t.Setenv("DEVBOX_USER", "envuser")
	id, err := r.Current()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "envuser" {
		t.Errorf("Username = %q, want %q", id.Username, "envuser")
	}
	if id.Source != SourceEnv {
		t.Errorf("Source = %q, want %q", id.Source, SourceEnv)
	}
}

func TestResolver_NilTailscale_EnvFallback(t *testing.T) {
	r := NewResolver(nil)
	t.Setenv("DEVBOX_USER", "testuser")
	id, err := r.Current()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Username != "testuser" {
		t.Errorf("Username = %q, want %q", id.Username, "testuser")
	}
	if id.Source != SourceEnv {
		t.Errorf("Source = %q, want %q", id.Source, SourceEnv)
	}
}

func TestResolver_NoSource(t *testing.T) {
	ts := &mockTSManager{err: errors.New("not connected")}
	r := NewResolver(ts)
	t.Setenv("DEVBOX_USER", "")
	_, err := r.Current()
	if err == nil {
		t.Fatal("expected error when no identity source available")
	}
}
