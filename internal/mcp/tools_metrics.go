package mcp

import (
	"context"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/workspace"
)

// workspaceMetricsResponse is the JSON response for per-workspace metrics.
type workspaceMetricsResponse struct {
	Container  string  `json:"container"`
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   uint64  `json:"mem_usage_bytes"`
	MemLimit   uint64  `json:"mem_limit_bytes"`
	DiskUsage  uint64  `json:"disk_usage_bytes,omitempty"`
	DiskTotal  uint64  `json:"disk_total_bytes,omitempty"`
	NetIn      uint64  `json:"net_in_bytes"`
	NetOut     uint64  `json:"net_out_bytes"`
	Stopped    bool    `json:"stopped,omitempty"`
}

// serverMetricsResponse is the JSON response for server-level metrics.
type serverMetricsResponse struct {
	TotalCPUs  int                     `json:"total_cpus"`
	TotalMem   uint64                  `json:"total_memory_bytes"`
	UsedMem    uint64                  `json:"used_memory_bytes"`
	TotalDisk  uint64                  `json:"total_disk_bytes"`
	UsedDisk   uint64                  `json:"used_disk_bytes"`
	Workspaces []workspaceMetricsEntry `json:"workspaces"`
}

// handleMetrics returns a handler for the devbox_metrics tool.
func handleMetrics(mgr workspace.Manager, pool server.Pool, collector metrics.Collector) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		wsName := getString(args, "workspace")
		serverName := getString(args, "server")

		if wsName == "" && serverName == "" {
			return toolError(ErrInvalidInput, "provide either workspace or server parameter"), nil
		}

		// Per-workspace metrics.
		if wsName != "" {
			return collectWorkspaceMetrics(ctx, mgr, collector, wsName)
		}

		// Server-level metrics.
		return collectServerMetrics(ctx, pool, collector, serverName)
	}
}

// collectWorkspaceMetrics returns metrics for a single workspace.
func collectWorkspaceMetrics(ctx context.Context, mgr workspace.Manager, collector metrics.Collector, name string) (*gomcp.CallToolResult, error) {
	ws, err := mgr.Get(name)
	if err != nil {
		return toolErrorf(ErrNotFound, "workspace %q not found: %v", name, err), nil
	}

	wm, err := collector.CollectWorkspace(ctx, ws.ServerHost, name)
	if err != nil {
		return toolErrorf(ErrInternal, "collecting metrics for workspace %q: %v", name, err), nil
	}

	resp := workspaceMetricsResponse{
		Container:  wm.Container,
		CPUPercent: wm.CPUPercent,
		MemUsage:   wm.MemUsage,
		MemLimit:   wm.MemLimit,
		DiskUsage:  wm.DiskUsage,
		DiskTotal:  wm.DiskTotal,
		NetIn:      wm.NetIn,
		NetOut:     wm.NetOut,
		Stopped:    wm.Stopped,
	}
	return toolSuccess(resp)
}

// collectServerMetrics returns metrics for a server with workspace breakdown.
func collectServerMetrics(ctx context.Context, pool server.Pool, collector metrics.Collector, name string) (*gomcp.CallToolResult, error) {
	srv, err := resolveServerHost(pool, name)
	if err != nil {
		return toolErrorf(ErrNotFound, "server %q not found", name), nil
	}

	host := server.SSHHost(srv)
	sm, err := collector.CollectServer(ctx, host)
	if err != nil {
		return toolErrorf(ErrInternal, "collecting server metrics: %v", err), nil
	}

	resp := serverMetricsResponse{
		TotalCPUs:  sm.TotalCPUs,
		TotalMem:   sm.TotalMem,
		UsedMem:    sm.UsedMem,
		TotalDisk:  sm.TotalDisk,
		UsedDisk:   sm.UsedDisk,
		Workspaces: make([]workspaceMetricsEntry, 0, len(sm.Workspaces)),
	}
	for _, wm := range sm.Workspaces {
		resp.Workspaces = append(resp.Workspaces, workspaceMetricsEntry{
			Container:  wm.Container,
			CPUPercent: wm.CPUPercent,
			MemUsage:   wm.MemUsage,
			MemLimit:   wm.MemLimit,
			DiskUsage:  wm.DiskUsage,
			NetIn:      wm.NetIn,
			NetOut:     wm.NetOut,
		})
	}

	return toolSuccess(resp)
}
