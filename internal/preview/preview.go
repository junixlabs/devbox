// Package preview builds the machine-readable output devbox emits (via the
// `--json` flag on `up`/`list`) so an orchestrator such as Forge can consume a
// workspace preview — connect URL, QR payload, status — without scraping
// human-readable logs. It is the mobile analog of Coolify's staging URL.
package preview

import (
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/workspace"
)

// defaultMetroPort is the Metro bundler port used when a host-runtime Expo
// workspace does not declare one explicitly (matches the expo builtin template).
const defaultMetroPort = 8081

// Result is the structured preview payload for a single workspace. Field
// order/tags are the stable contract Forge parses; keep additive.
type Result struct {
	// Status mirrors the workspace status ("running", "stopped", ...).
	Status string `json:"status"`
	// ConnectURL is the address a device opens to load the workspace bundle
	// (see ConnectURL). Empty when the tailnet address could not be resolved.
	ConnectURL string `json:"connect_url"`
	// QR is a scannable representation of ConnectURL: a base64 PNG data-URI
	// when a connect URL is available, otherwise empty.
	QR string `json:"qr"`
	// Logs is an optional human-readable hint; structured fields above are the
	// source of truth so clients never have to scrape logs.
	Logs string `json:"logs,omitempty"`
	// Mode is how the code is served: "fast-refresh" (Metro hot reload) or
	// "build" (native/EAS build). Empty for non-mobile workspaces.
	Mode string `json:"mode"`
}

// ConnectURL builds the address a device uses to reach the workspace. For a
// host-runtime (Expo/Metro) workspace it returns an `exp://<fqdn>:<metro>`
// URL that Expo Go opens directly over the tailnet (no Expo relay). For other
// runtimes it returns the HTTPS Tailscale-serve URL. Returns "" when fqdn is
// empty (Tailscale status unresolved) — the caller treats that as "off-tailnet,
// fall back to --tunnel".
func ConnectURL(ws workspace.Workspace, fqdn string) string {
	if fqdn == "" {
		return ""
	}
	if ws.Runtime == config.RuntimeHost {
		port := ws.Ports["metro"]
		if port == 0 {
			port = defaultMetroPort
		}
		return fmt.Sprintf("exp://%s:%d", fqdn, port)
	}
	return "https://" + fqdn
}

// QRTerminal renders text as a compact QR code drawn with Unicode half-blocks,
// scannable straight from a terminal. Empty text yields an empty string.
func QRTerminal(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	q, err := qrcode.New(text, qrcode.Low)
	if err != nil {
		return "", fmt.Errorf("encoding QR code: %w", err)
	}
	return q.ToSmallString(false), nil
}

// QRDataURI encodes text as a QR code and returns it as a base64 PNG data-URI
// (directly renderable by a client). Empty text yields an empty string.
func QRDataURI(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("encoding QR code: %w", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// Build assembles a Result for a workspace: connect URL from ws+fqdn, a QR
// data-URI of that URL, and the given serve mode.
func Build(ws workspace.Workspace, fqdn, mode string) (Result, error) {
	url := ConnectURL(ws, fqdn)
	qr, err := QRDataURI(url)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Status:     string(ws.Status),
		ConnectURL: url,
		QR:         qr,
		Mode:       mode,
	}, nil
}

// ListEntry is the per-workspace projection emitted by `devbox list --json`.
type ListEntry struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Server     string `json:"server"`
	Branch     string `json:"branch,omitempty"`
	Runtime    string `json:"runtime,omitempty"`
	ConnectURL string `json:"connect_url"`
}

// NewListEntry projects a workspace into a ListEntry, resolving its connect URL
// from the given tailnet fqdn.
func NewListEntry(ws workspace.Workspace, fqdn string) ListEntry {
	return ListEntry{
		Name:       ws.Name,
		Status:     string(ws.Status),
		Server:     ws.ServerHost,
		Branch:     ws.Branch,
		Runtime:    ws.Runtime,
		ConnectURL: ConnectURL(ws, fqdn),
	}
}
