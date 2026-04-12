package workspace

import (
	"fmt"
	"strconv"
	"strings"
)

// ResourceUsage holds live resource consumption for a single container.
type ResourceUsage struct {
	CPUPercent    float64
	MemoryUsed   int64
	MemoryLimit  int64
	MemoryPercent float64
}

// ServerResourceInfo holds total and used resources for a server.
type ServerResourceInfo struct {
	TotalCPUs        int
	TotalMemoryBytes int64
	UsedCPUs         float64
	UsedMemoryBytes  int64
}

// LowResourceThreshold is the default percentage above which warnings are emitted.
const LowResourceThreshold = 85.0

// ParseDockerStats parses the output of:
//
//	docker stats --no-stream --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
//
// Returns a map of container name → ResourceUsage.
func ParseDockerStats(output string) (map[string]*ResourceUsage, error) {
	result := make(map[string]*ResourceUsage)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		cpuPct := strings.TrimSuffix(parts[1], "%")
		memPct := strings.TrimSuffix(parts[3], "%")

		cpu, _ := strconv.ParseFloat(cpuPct, 64)
		mem, _ := strconv.ParseFloat(memPct, 64)

		// Parse memory usage: "123MiB / 4GiB"
		memUsed, memLimit := parseMemUsage(parts[2])

		result[name] = &ResourceUsage{
			CPUPercent:    cpu,
			MemoryUsed:   memUsed,
			MemoryLimit:  memLimit,
			MemoryPercent: mem,
		}
	}
	return result, nil
}

// parseMemUsage parses "123.4MiB / 7.775GiB" into bytes.
func parseMemUsage(s string) (used int64, limit int64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	used = parseMemValue(strings.TrimSpace(parts[0]))
	limit = parseMemValue(strings.TrimSpace(parts[1]))
	return used, limit
}

// parseMemValue parses values like "123.4MiB", "7.775GiB", "512KiB" into bytes.
func parseMemValue(s string) int64 {
	s = strings.TrimSpace(s)
	var multiplier int64
	switch {
	case strings.HasSuffix(s, "GiB"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GiB")
	case strings.HasSuffix(s, "MiB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MiB")
	case strings.HasSuffix(s, "KiB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KiB")
	case strings.HasSuffix(s, "B"):
		multiplier = 1
		s = strings.TrimSuffix(s, "B")
	default:
		return 0
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multiplier))
}

// ParseServerResources parses nproc output and /proc/meminfo to determine
// total server resources.
func ParseServerResources(cpuOut, memOut string) (*ServerResourceInfo, error) {
	cpuCount, err := strconv.Atoi(strings.TrimSpace(cpuOut))
	if err != nil {
		return nil, fmt.Errorf("parsing CPU count from nproc: %w", err)
	}

	var totalMemKB int64
	for _, line := range strings.Split(memOut, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				totalMemKB, _ = strconv.ParseInt(fields[1], 10, 64)
			}
			break
		}
	}
	if totalMemKB == 0 {
		return nil, fmt.Errorf("could not parse MemTotal from /proc/meminfo")
	}

	return &ServerResourceInfo{
		TotalCPUs:        cpuCount,
		TotalMemoryBytes: totalMemKB * 1024,
	}, nil
}

// CheckLowResources returns warning messages if server resource usage exceeds
// the given threshold percentage.
func CheckLowResources(info *ServerResourceInfo, threshold float64) []string {
	if info == nil {
		return nil
	}
	var warnings []string
	if info.TotalCPUs > 0 && info.UsedCPUs > 0 {
		cpuPct := (info.UsedCPUs / float64(info.TotalCPUs)) * 100
		if cpuPct >= threshold {
			warnings = append(warnings, fmt.Sprintf(
				"Server CPU usage at %.0f%% (%.1f/%d cores)",
				cpuPct, info.UsedCPUs, info.TotalCPUs,
			))
		}
	}
	if info.TotalMemoryBytes > 0 && info.UsedMemoryBytes > 0 {
		memPct := (float64(info.UsedMemoryBytes) / float64(info.TotalMemoryBytes)) * 100
		if memPct >= threshold {
			warnings = append(warnings, fmt.Sprintf(
				"Server memory usage at %.0f%% (%s/%s)",
				memPct, formatBytes(info.UsedMemoryBytes), formatBytes(info.TotalMemoryBytes),
			))
		}
	}
	return warnings
}

// FormatResourceUsage formats a ResourceUsage for table display.
func FormatResourceUsage(ru *ResourceUsage) (cpuStr, memStr string) {
	if ru == nil {
		return "-", "-"
	}
	cpuStr = fmt.Sprintf("%.1f%%", ru.CPUPercent)
	memStr = fmt.Sprintf("%s / %s (%.1f%%)", formatBytes(ru.MemoryUsed), formatBytes(ru.MemoryLimit), ru.MemoryPercent)
	return cpuStr, memStr
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024*1024:
		return fmt.Sprintf("%.1fGi", float64(b)/(1024*1024*1024))
	case b >= 1024*1024:
		return fmt.Sprintf("%.0fMi", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.0fKi", float64(b)/1024)
	default:
		return fmt.Sprintf("%dB", b)
	}
}
