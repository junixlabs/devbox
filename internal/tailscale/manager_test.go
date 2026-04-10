package tailscale

import (
	"errors"
	"testing"
)

func mockRunner(out []byte, err error) CommandRunner {
	return func(command string, args ...string) ([]byte, error) {
		return out, err
	}
}

func TestServe_Success(t *testing.T) {
	mgr := NewManager(mockRunner(nil, nil))
	if err := mgr.Serve(8080, "myhost"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestServe_Error(t *testing.T) {
	mgr := NewManager(mockRunner(nil, errors.New("connection refused")))
	err := mgr.Serve(8080, "myhost")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "tailscale serve port 8080: connection refused" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestUnserve_Success(t *testing.T) {
	mgr := NewManager(mockRunner(nil, nil))
	if err := mgr.Unserve(8080); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnserve_Error(t *testing.T) {
	mgr := NewManager(mockRunner(nil, errors.New("not serving")))
	err := mgr.Unserve(3000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "tailscale unserve port 3000: not serving" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

const validStatusJSON = `{
	"BackendState": "Running",
	"MagicDNSSuffix": "example.com",
	"Self": {
		"HostName": "devbox-vps",
		"TailscaleIPs": ["100.117.246.55", "fd7a:115c:a1e0::1"]
	}
}`

func TestStatus_Success(t *testing.T) {
	mgr := NewManager(mockRunner([]byte(validStatusJSON), nil))
	info, err := mgr.Status()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !info.Connected {
		t.Error("expected Connected=true")
	}
	if info.Hostname != "devbox-vps" {
		t.Errorf("expected Hostname=devbox-vps, got %s", info.Hostname)
	}
	if info.TailnetName != "example.com" {
		t.Errorf("expected TailnetName=example.com, got %s", info.TailnetName)
	}
	if info.IP != "100.117.246.55" {
		t.Errorf("expected IP=100.117.246.55, got %s", info.IP)
	}
}

func TestStatus_NotRunning(t *testing.T) {
	stoppedJSON := `{
		"BackendState": "Stopped",
		"MagicDNSSuffix": "example.com",
		"Self": {
			"HostName": "devbox-vps",
			"TailscaleIPs": ["100.117.246.55"]
		}
	}`
	mgr := NewManager(mockRunner([]byte(stoppedJSON), nil))
	info, err := mgr.Status()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.Connected {
		t.Error("expected Connected=false for Stopped state")
	}
}

func TestStatus_CommandError(t *testing.T) {
	mgr := NewManager(mockRunner(nil, errors.New("tailscale not found")))
	_, err := mgr.Status()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "tailscale status: tailscale not found" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestStatus_InvalidJSON(t *testing.T) {
	mgr := NewManager(mockRunner([]byte("not json"), nil))
	_, err := mgr.Status()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStatus_NoIPs(t *testing.T) {
	noIPJSON := `{
		"BackendState": "Running",
		"MagicDNSSuffix": "example.com",
		"Self": {
			"HostName": "devbox-vps",
			"TailscaleIPs": []
		}
	}`
	mgr := NewManager(mockRunner([]byte(noIPJSON), nil))
	_, err := mgr.Status()
	if err == nil {
		t.Fatal("expected error for empty TailscaleIPs")
	}
	if got := err.Error(); got != "tailscale status: no IP addresses reported" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestWorkspaceURL(t *testing.T) {
	tests := []struct {
		hostname, tailnet, want string
	}{
		{"myhost", "tailb5de5c.ts.net", "https://myhost.tailb5de5c.ts.net"},
		{"devbox-vps", "example.com", "https://devbox-vps.example.com"},
	}
	for _, tt := range tests {
		got := WorkspaceURL(tt.hostname, tt.tailnet)
		if got != tt.want {
			t.Errorf("WorkspaceURL(%q, %q) = %q, want %q", tt.hostname, tt.tailnet, got, tt.want)
		}
	}
}
