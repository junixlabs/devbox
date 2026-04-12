package port

// DefaultPortRange is the default range for auto-allocated ports.
var DefaultPortRange = PortRange{Min: 10000, Max: 60000}

// PortRange defines the inclusive range of ports available for auto-allocation.
type PortRange struct {
	Min int
	Max int
}

// Allocation represents a single port assignment for a workspace service.
type Allocation struct {
	WorkspaceName string `json:"workspace_name"`
	ServiceName   string `json:"service_name"`
	Port          int    `json:"port"`
	Manual        bool   `json:"manual"`
}

// Conflict describes a port collision between two workspace services.
type Conflict struct {
	Port       int
	WorkspaceA string
	WorkspaceB string
	Service    string
}

// Registry manages port allocations across workspaces.
type Registry interface {
	// Allocate assigns a port for the given workspace service.
	// If override is non-nil, it uses that port (checking for conflicts).
	// Otherwise it auto-assigns from the configured range.
	Allocate(workspace, service string, override *int) (int, error)

	// Release removes all port allocations for the given workspace.
	Release(workspace string) error

	// GetAllocations returns the port map for a workspace (service → port).
	GetAllocations(workspace string) (map[string]int, error)

	// CheckConflicts detects duplicate port assignments across all workspaces.
	CheckConflicts() ([]Conflict, error)

	// ListAll returns every current allocation.
	ListAll() ([]Allocation, error)
}
