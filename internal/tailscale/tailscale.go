package tailscale

import "fmt"

// CommandRunner executes a command with arguments on the target machine.
// Returns the combined stdout output and any error.
type CommandRunner func(command string, args ...string) ([]byte, error)

// Manager defines the interface for Tailscale integration.
type Manager interface {
	// Serve exposes a local port via Tailscale serve (HTTPS with auto-cert).
	Serve(port int, hostname string) error

	// Unserve stops exposing a port via Tailscale serve.
	Unserve(port int) error

	// Status returns the current Tailscale connection status.
	Status() (*StatusInfo, error)
}

// StatusInfo holds basic Tailscale status information.
type StatusInfo struct {
	// Connected indicates whether Tailscale is connected to the network.
	Connected bool `json:"connected"`

	// Hostname is the Tailscale hostname of this machine.
	Hostname string `json:"hostname"`

	// TailnetName is the name of the tailnet this machine belongs to.
	TailnetName string `json:"tailnet_name"`

	// IP is the Tailscale IP address of this machine.
	IP string `json:"ip"`
}

// NewManager creates a Manager that executes Tailscale CLI commands via the given runner.
func NewManager(run CommandRunner) Manager {
	return &tsManager{run: run}
}

// WorkspaceURL returns the HTTPS URL for a workspace on Tailscale.
func WorkspaceURL(hostname, tailnet string) string {
	return fmt.Sprintf("https://%s.%s.ts.net", hostname, tailnet)
}
