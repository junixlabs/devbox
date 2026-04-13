package mcp

import (
	"context"

	gomcp "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/junixlabs/devbox/internal/metrics"
	"github.com/junixlabs/devbox/internal/server"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
)

// serverListEntry is the JSON response for a single server in devbox_server_list.
type serverListEntry struct {
	Name      string              `json:"name"`
	Host      string              `json:"host"`
	Status    string              `json:"status"`
	Health    healthInfo          `json:"health"`
	Resources *resourceInfo       `json:"resources,omitempty"`
}

type healthInfo struct {
	SSH       bool `json:"ssh"`
	Docker    bool `json:"docker"`
	Tailscale bool `json:"tailscale"`
}

type resourceInfo struct {
	TotalCPUs      int     `json:"total_cpus"`
	CPUUsedPercent float64 `json:"cpu_used_percent"`
	TotalMemBytes  int64   `json:"total_memory_bytes"`
	UsedMemBytes   int64   `json:"used_memory_bytes"`
	AvailableScore float64 `json:"available_score"`
}

// serverStatusResponse is the JSON response for devbox_server_status.
type serverStatusResponse struct {
	Name       string                  `json:"name"`
	Host       string                  `json:"host"`
	Health     healthInfo              `json:"health"`
	Resources  *serverDetailResources  `json:"resources,omitempty"`
	Workspaces []workspaceMetricsEntry `json:"workspaces,omitempty"`
}

type serverDetailResources struct {
	TotalCPUs      int     `json:"total_cpus"`
	CPUUsedPercent float64 `json:"cpu_used_percent"`
	TotalMemBytes  int64   `json:"total_memory_bytes"`
	UsedMemBytes   int64   `json:"used_memory_bytes"`
	TotalDisk      uint64  `json:"total_disk_bytes"`
	UsedDisk       uint64  `json:"used_disk_bytes"`
}

type workspaceMetricsEntry struct {
	Container  string  `json:"container"`
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   uint64  `json:"mem_usage_bytes"`
	MemLimit   uint64  `json:"mem_limit_bytes"`
	DiskUsage  uint64  `json:"disk_usage_bytes,omitempty"`
	NetIn      uint64  `json:"net_in_bytes"`
	NetOut     uint64  `json:"net_out_bytes"`
}

// handleServerList returns a handler for the devbox_server_list tool.
func handleServerList(pool server.Pool, exec devboxssh.Executor) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		servers, err := pool.List()
		if err != nil {
			return toolErrorf(ErrInternal, "listing servers: %v", err), nil
		}

		if len(servers) == 0 {
			return toolSuccess([]serverListEntry{})
		}

		healthMap, err := pool.HealthCheckAll()
		if err != nil {
			return toolErrorf(ErrInternal, "health check: %v", err), nil
		}

		entries := make([]serverListEntry, 0, len(servers))
		for i := range servers {
			srv := &servers[i]
			h := healthMap[srv.Name]

			entry := serverListEntry{
				Name: srv.Name,
				Host: srv.Host,
				Health: healthInfo{
					SSH:       h != nil && h.SSH,
					Docker:    h != nil && h.Docker,
					Tailscale: h != nil && h.Tailscale,
				},
			}

			if h != nil && h.SSH {
				entry.Status = "online"
				// Query resource availability for online servers.
				host := server.SSHHost(srv)
				ri, qErr := server.QueryResources(ctx, exec, host)
				if qErr == nil {
					entry.Resources = &resourceInfo{
						TotalCPUs:      ri.TotalCPUs,
						CPUUsedPercent: ri.UsedCPUPercent,
						TotalMemBytes:  ri.TotalMemoryBytes,
						UsedMemBytes:   ri.UsedMemoryBytes,
						AvailableScore: server.AvailableScore(ri),
					}
				}
			} else {
				entry.Status = "offline"
			}

			entries = append(entries, entry)
		}

		return toolSuccess(entries)
	}
}

// handleServerStatus returns a handler for the devbox_server_status tool.
func handleServerStatus(pool server.Pool, collector metrics.Collector, exec devboxssh.Executor) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		args := request.GetArguments()
		name := getString(args, "name")
		if name == "" {
			return toolError(ErrInvalidInput, "name is required"), nil
		}

		srv, err := resolveServerHost(pool, name)
		if err != nil {
			return toolErrorf(ErrNotFound, "server %q not found", name), nil
		}

		health, err := pool.HealthCheck(name)
		if err != nil {
			return toolErrorf(ErrInternal, "health check: %v", err), nil
		}

		resp := serverStatusResponse{
			Name: srv.Name,
			Host: srv.Host,
			Health: healthInfo{
				SSH:       health.SSH,
				Docker:    health.Docker,
				Tailscale: health.Tailscale,
			},
		}

		// Only query resources and metrics if the server is online.
		if health.SSH {
			host := server.SSHHost(srv)

			ri, qErr := server.QueryResources(ctx, exec, host)
			if qErr == nil {
				resp.Resources = &serverDetailResources{
					TotalCPUs:      ri.TotalCPUs,
					CPUUsedPercent: ri.UsedCPUPercent,
					TotalMemBytes:  ri.TotalMemoryBytes,
					UsedMemBytes:   ri.UsedMemoryBytes,
				}
			}

			// Collect per-workspace metrics and server disk.
			sm, mErr := collector.CollectServer(ctx, host)
			if mErr == nil {
				if resp.Resources != nil {
					resp.Resources.TotalDisk = sm.TotalDisk
					resp.Resources.UsedDisk = sm.UsedDisk
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
			}
		}

		return toolSuccess(resp)
	}
}
