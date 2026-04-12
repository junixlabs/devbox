package metrics

import "context"

// WorkspaceMetrics holds resource usage for a single workspace container.
type WorkspaceMetrics struct {
	Container  string
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	DiskUsage  uint64
	DiskTotal  uint64
	NetIn      uint64
	NetOut     uint64
	Stopped    bool
}

// ServerMetrics holds aggregate resource info for a server.
type ServerMetrics struct {
	TotalCPUs  int
	TotalMem   uint64
	UsedMem    uint64
	TotalDisk  uint64
	UsedDisk   uint64
	Workspaces []WorkspaceMetrics
}

// Collector gathers resource metrics from remote servers.
type Collector interface {
	// CollectWorkspace returns metrics for a single workspace container.
	CollectWorkspace(ctx context.Context, host, container string) (*WorkspaceMetrics, error)

	// CollectServer returns aggregate metrics plus per-workspace breakdown.
	CollectServer(ctx context.Context, host string) (*ServerMetrics, error)
}
