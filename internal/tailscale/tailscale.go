package tailscale

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
