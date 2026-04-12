package server

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

// ResourceInfo holds resource availability data for a server.
type ResourceInfo struct {
	TotalCPUs        int
	UsedCPUPercent   float64
	TotalMemoryBytes int64
	UsedMemoryBytes  int64
}

// QueryResources queries CPU and memory availability on a remote server via SSH.
// It runs nproc, free -b, and reads /proc/loadavg to determine resource state.
func QueryResources(ctx context.Context, exec devboxssh.Executor, host string) (*ResourceInfo, error) {
	cpuOut, _, err := exec.Run(ctx, host, "nproc")
	if err != nil {
		return nil, fmt.Errorf("querying CPU count on %s: %w", host, err)
	}

	memOut, _, err := exec.Run(ctx, host, "free -b")
	if err != nil {
		return nil, fmt.Errorf("querying memory on %s: %w", host, err)
	}

	cpuCount, err := strconv.Atoi(strings.TrimSpace(cpuOut))
	if err != nil {
		return nil, fmt.Errorf("parsing CPU count: %w", err)
	}

	totalMem, usedMem, err := parseFreeOutput(memOut)
	if err != nil {
		return nil, fmt.Errorf("parsing memory info: %w", err)
	}

	// Use /proc/loadavg for CPU usage estimate (1-minute load average).
	loadOut, _, err := exec.Run(ctx, host, "cat /proc/loadavg")
	if err != nil {
		// Best-effort: if we can't get load, assume 0% used.
		return &ResourceInfo{
			TotalCPUs:        cpuCount,
			TotalMemoryBytes: totalMem,
			UsedMemoryBytes:  usedMem,
		}, nil
	}

	cpuPercent := 0.0
	if cpuCount > 0 {
		loadAvg := parseLoadAvg(loadOut)
		cpuPercent = (loadAvg / float64(cpuCount)) * 100
		if cpuPercent > 100 {
			cpuPercent = 100
		}
	}

	return &ResourceInfo{
		TotalCPUs:        cpuCount,
		UsedCPUPercent:   cpuPercent,
		TotalMemoryBytes: totalMem,
		UsedMemoryBytes:  usedMem,
	}, nil
}

// AvailableScore returns a score from 0.0 to 1.0 representing how much
// capacity is available on the server. Higher means more available.
// Formula: 50% CPU-free-ratio + 50% memory-free-ratio.
func AvailableScore(info *ResourceInfo) float64 {
	if info == nil {
		return 0
	}

	cpuFreeRatio := 1.0
	if info.UsedCPUPercent > 0 {
		cpuFreeRatio = 1.0 - (info.UsedCPUPercent / 100.0)
		if cpuFreeRatio < 0 {
			cpuFreeRatio = 0
		}
	}

	memFreeRatio := 1.0
	if info.TotalMemoryBytes > 0 {
		memFreeRatio = 1.0 - (float64(info.UsedMemoryBytes) / float64(info.TotalMemoryBytes))
		if memFreeRatio < 0 {
			memFreeRatio = 0
		}
	}

	return 0.5*cpuFreeRatio + 0.5*memFreeRatio
}

// parseFreeOutput parses the output of `free -b` to extract total and used memory.
func parseFreeOutput(output string) (total int64, used int64, err error) {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return 0, 0, fmt.Errorf("unexpected free output format: %s", line)
			}
			total, err = strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parsing total memory: %w", err)
			}
			used, err = strconv.ParseInt(fields[2], 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("parsing used memory: %w", err)
			}
			return total, used, nil
		}
	}
	return 0, 0, fmt.Errorf("could not find Mem: line in free output")
}

// parseLoadAvg parses the 1-minute load average from /proc/loadavg.
func parseLoadAvg(output string) float64 {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) == 0 {
		return 0
	}
	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return load
}
