package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/junixlabs/devbox/internal/ssh"
)

// validContainerName matches valid Docker container names.
var validContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

// dockerStatsJSON mirrors the JSON output of docker stats --format '{{json .}}'.
type dockerStatsJSON struct {
	Name    string `json:"Name"`
	CPUPerc string `json:"CPUPerc"`
	MemPerc string `json:"MemPerc"`
	MemUsage string `json:"MemUsage"`
	NetIO   string `json:"NetIO"`
	BlockIO string `json:"BlockIO"`
}

// sshCollector implements Collector using an ssh.Executor.
type sshCollector struct {
	exec ssh.Executor
}

// NewCollector creates a Collector that runs commands on remote hosts via SSH.
func NewCollector(exec ssh.Executor) Collector {
	return &sshCollector{exec: exec}
}

func (c *sshCollector) CollectWorkspace(ctx context.Context, host, container string) (*WorkspaceMetrics, error) {
	if !validContainerName.MatchString(container) {
		return nil, fmt.Errorf("invalid container name: %q", container)
	}

	// Collect docker stats for the single container.
	cmd := fmt.Sprintf("docker stats %s --no-stream --format '{{json .}}'", container)
	stdout, _, err := c.exec.Run(ctx, host, cmd)
	if err != nil {
		// Container may be stopped — return zero metrics.
		return &WorkspaceMetrics{Container: container, Stopped: true}, nil
	}

	wm, err := parseDockerStatsJSON(strings.TrimSpace(stdout))
	if err != nil {
		return nil, fmt.Errorf("parsing stats for container %s: %w", container, err)
	}

	// Collect disk usage inside the container.
	diskCmd := fmt.Sprintf("docker exec %s df -B1 / 2>/dev/null | tail -1", container)
	diskOut, _, diskErr := c.exec.Run(ctx, host, diskCmd)
	if diskErr == nil {
		wm.DiskUsage, wm.DiskTotal = parseDfBytes(diskOut)
	}

	return wm, nil
}

func (c *sshCollector) CollectServer(ctx context.Context, host string) (*ServerMetrics, error) {
	// Single SSH command for all container stats + server info.
	cmd := "docker stats --no-stream --format '{{json .}}' 2>/dev/null; " +
		"echo '===METRICS_SEP==='; " +
		"nproc; " +
		"echo '===METRICS_SEP==='; " +
		"cat /proc/meminfo; " +
		"echo '===METRICS_SEP==='; " +
		"df -B1 / | tail -1"
	stdout, _, err := c.exec.Run(ctx, host, cmd)
	if err != nil {
		return nil, fmt.Errorf("collecting server metrics on %s: %w", host, err)
	}

	parts := strings.Split(stdout, "===METRICS_SEP===")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected metrics output from %s (got %d parts)", host, len(parts))
	}

	dockerOut := strings.TrimSpace(parts[0])
	cpuOut := strings.TrimSpace(parts[1])
	memOut := strings.TrimSpace(parts[2])
	diskOut := strings.TrimSpace(parts[3])

	sm := &ServerMetrics{}

	// Parse container stats.
	if dockerOut != "" {
		for _, line := range strings.Split(dockerOut, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			wm, err := parseDockerStatsJSON(line)
			if err != nil {
				continue
			}
			sm.Workspaces = append(sm.Workspaces, *wm)
		}
	}

	// Parse CPU count.
	cpuCount, err := strconv.Atoi(cpuOut)
	if err == nil {
		sm.TotalCPUs = cpuCount
	}

	// Parse memory from /proc/meminfo.
	sm.TotalMem, sm.UsedMem = parseMeminfo(memOut)

	// Parse server disk.
	sm.UsedDisk, sm.TotalDisk = parseDfBytes(diskOut)

	return sm, nil
}

// parseDockerStatsJSON parses a single JSON line from docker stats.
func parseDockerStatsJSON(line string) (*WorkspaceMetrics, error) {
	var ds dockerStatsJSON
	if err := json.Unmarshal([]byte(line), &ds); err != nil {
		return nil, fmt.Errorf("parsing docker stats JSON: %w", err)
	}

	wm := &WorkspaceMetrics{Container: ds.Name}
	wm.CPUPercent = parsePercent(ds.CPUPerc)

	// Parse memory: "123MiB / 4GiB"
	memParts := strings.Split(ds.MemUsage, "/")
	if len(memParts) == 2 {
		wm.MemUsage = ParseByteSize(strings.TrimSpace(memParts[0]))
		wm.MemLimit = ParseByteSize(strings.TrimSpace(memParts[1]))
	}

	// Parse network: "1.5kB / 2.3MB"
	netParts := strings.Split(ds.NetIO, "/")
	if len(netParts) == 2 {
		wm.NetIn = ParseByteSize(strings.TrimSpace(netParts[0]))
		wm.NetOut = ParseByteSize(strings.TrimSpace(netParts[1]))
	}

	return wm, nil
}

// parsePercent parses "1.23%" → 1.23.
func parsePercent(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ParseByteSize parses human-readable byte strings like "1.23GiB", "456MiB",
// "789kB", "1.5MB", "100B" into bytes.
func ParseByteSize(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	type suffix struct {
		name string
		mult float64
	}
	// Order matters: check longer suffixes first.
	suffixes := []suffix{
		{"GiB", 1024 * 1024 * 1024},
		{"MiB", 1024 * 1024},
		{"KiB", 1024},
		{"GB", 1e9},
		{"MB", 1e6},
		{"kB", 1e3},
		{"B", 1},
	}

	for _, sf := range suffixes {
		if strings.HasSuffix(s, sf.name) {
			numStr := strings.TrimSpace(strings.TrimSuffix(s, sf.name))
			v, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0
			}
			return uint64(v * sf.mult)
		}
	}
	return 0
}

// parseDfBytes parses the output of `df -B1 / | tail -1`.
// Format: "filesystem  total  used  avail  use%  mount"
func parseDfBytes(output string) (used, total uint64) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) < 4 {
		return 0, 0
	}
	t, _ := strconv.ParseUint(fields[1], 10, 64)
	u, _ := strconv.ParseUint(fields[2], 10, 64)
	return u, t
}

// parseMeminfo extracts MemTotal and calculates used memory from /proc/meminfo.
func parseMeminfo(output string) (total, used uint64) {
	var memTotal, memAvailable uint64
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		valBytes := val * 1024 // /proc/meminfo values are in kB
		switch fields[0] {
		case "MemTotal:":
			memTotal = valBytes
		case "MemAvailable:":
			memAvailable = valBytes
		}
	}
	if memTotal > 0 && memAvailable <= memTotal {
		return memTotal, memTotal - memAvailable
	}
	return memTotal, 0
}
