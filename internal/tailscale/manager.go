package tailscale

import (
	"encoding/json"
	"fmt"
	"strings"
)

// tsManager implements Manager by shelling out to the Tailscale CLI via a CommandRunner.
type tsManager struct {
	run CommandRunner
}

func (m *tsManager) Serve(port int, hostname string) error {
	_, err := m.run("tailscale", "serve", "--bg", fmt.Sprintf("%d", port))
	if err != nil {
		return fmt.Errorf("tailscale serve port %d: %w", port, err)
	}
	return nil
}

func (m *tsManager) Unserve(port int) error {
	_, err := m.run("tailscale", "serve", "--remove", fmt.Sprintf("%d", port))
	if err != nil {
		return fmt.Errorf("tailscale unserve port %d: %w", port, err)
	}
	return nil
}

// tsStatusJSON mirrors the relevant fields from `tailscale status --json`.
// Only fields we consume are declared — resilient to Tailscale version additions.
type tsStatusJSON struct {
	BackendState   string `json:"BackendState"`
	MagicDNSSuffix string `json:"MagicDNSSuffix"`
	Self           struct {
		HostName     string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
	} `json:"Self"`
}

func (m *tsManager) Status() (*StatusInfo, error) {
	out, err := m.run("tailscale", "status", "--json")
	if err != nil {
		return nil, fmt.Errorf("tailscale status: %w", err)
	}

	var raw tsStatusJSON
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("tailscale status: failed to parse JSON: %w", err)
	}

	if len(raw.Self.TailscaleIPs) == 0 {
		return nil, fmt.Errorf("tailscale status: no IP addresses reported")
	}

	tailnet := strings.TrimPrefix(raw.MagicDNSSuffix, ".")

	return &StatusInfo{
		Connected:   raw.BackendState == "Running",
		Hostname:    raw.Self.HostName,
		TailnetName: tailnet,
		IP:          raw.Self.TailscaleIPs[0],
	}, nil
}
