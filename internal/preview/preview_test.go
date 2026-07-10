package preview

import (
	"strings"
	"testing"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/workspace"
)

func TestConnectURL_HostRuntimeExpo(t *testing.T) {
	ws := workspace.Workspace{
		Runtime: config.RuntimeHost,
		Ports:   map[string]int{"metro": 8081, "expo": 19000},
	}
	got := ConnectURL(ws, "devbox-vps.tailb5de5c.ts.net")
	want := "exp://devbox-vps.tailb5de5c.ts.net:8081"
	if got != want {
		t.Errorf("ConnectURL = %q, want %q", got, want)
	}
}

func TestConnectURL_HostRuntimeDefaultsMetroPort(t *testing.T) {
	ws := workspace.Workspace{Runtime: config.RuntimeHost, Ports: map[string]int{}}
	got := ConnectURL(ws, "box.ts.net")
	if got != "exp://box.ts.net:8081" {
		t.Errorf("ConnectURL = %q, want default metro port 8081", got)
	}
}

func TestConnectURL_NonHostFallsBackToHTTPS(t *testing.T) {
	ws := workspace.Workspace{Runtime: "docker"}
	got := ConnectURL(ws, "box.ts.net")
	if got != "https://box.ts.net" {
		t.Errorf("ConnectURL = %q, want https fallback", got)
	}
}

func TestConnectURL_EmptyFQDN(t *testing.T) {
	ws := workspace.Workspace{Runtime: config.RuntimeHost, Ports: map[string]int{"metro": 8081}}
	if got := ConnectURL(ws, ""); got != "" {
		t.Errorf("ConnectURL with empty fqdn = %q, want empty", got)
	}
}

func TestQRDataURI(t *testing.T) {
	got, err := QRDataURI("exp://box.ts.net:8081")
	if err != nil {
		t.Fatalf("QRDataURI: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("QRDataURI = %q, want data:image/png;base64, prefix", got[:min(40, len(got))])
	}
	if len(got) < 100 {
		t.Errorf("QRDataURI too short (%d bytes), expected a real PNG payload", len(got))
	}
}

func TestQRDataURI_Empty(t *testing.T) {
	got, err := QRDataURI("")
	if err != nil || got != "" {
		t.Errorf("QRDataURI(\"\") = %q, %v; want empty, nil", got, err)
	}
}

func TestBuild(t *testing.T) {
	ws := workspace.Workspace{
		Status:  workspace.StatusRunning,
		Runtime: config.RuntimeHost,
		Ports:   map[string]int{"metro": 8081},
	}
	res, err := Build(ws, "box.ts.net", "fast-refresh")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if res.Status != "running" || res.ConnectURL != "exp://box.ts.net:8081" || res.Mode != "fast-refresh" {
		t.Errorf("Build = %+v, unexpected fields", res)
	}
	if !strings.HasPrefix(res.QR, "data:image/png;base64,") {
		t.Errorf("Build QR = %q, want data-URI", res.QR[:min(40, len(res.QR))])
	}
}

func TestNewListEntry(t *testing.T) {
	ws := workspace.Workspace{
		Name:       "alice-app",
		Status:     workspace.StatusRunning,
		ServerHost: "vps1",
		Branch:     "feat/x",
		Runtime:    config.RuntimeHost,
		Ports:      map[string]int{"metro": 8081},
	}
	e := NewListEntry(ws, "box.ts.net")
	if e.Name != "alice-app" || e.Server != "vps1" || e.ConnectURL != "exp://box.ts.net:8081" {
		t.Errorf("NewListEntry = %+v, unexpected", e)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
