package port

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// fileRegistry implements Registry by persisting allocations to a JSON file.
type fileRegistry struct {
	path      string
	portRange PortRange
	mu        sync.Mutex
}

// NewFileRegistry creates a Registry backed by a JSON state file.
// TODO: In-process mutex only protects single-process access. For concurrent
// devbox processes, file-level locking (flock) would be needed.
func NewFileRegistry(path string, portRange PortRange) Registry {
	return &fileRegistry{
		path:      path,
		portRange: portRange,
	}
}

func (r *fileRegistry) Allocate(workspace, service string, override *int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocs, err := r.load()
	if err != nil {
		return 0, err
	}

	// Check if this workspace+service already has an allocation.
	for _, a := range allocs {
		if a.WorkspaceName == workspace && a.ServiceName == service {
			return a.Port, nil
		}
	}

	usedPorts := make(map[int]string) // port → workspace
	for _, a := range allocs {
		usedPorts[a.Port] = a.WorkspaceName
	}

	var port int
	var manual bool

	if override != nil {
		port = *override
		manual = true
		if owner, taken := usedPorts[port]; taken {
			return 0, fmt.Errorf("port %d already allocated to workspace %q", port, owner)
		}
	} else {
		port, err = r.findFreePort(usedPorts)
		if err != nil {
			return 0, err
		}
	}

	allocs = append(allocs, Allocation{
		WorkspaceName: workspace,
		ServiceName:   service,
		Port:          port,
		Manual:        manual,
	})

	if err := r.save(allocs); err != nil {
		return 0, err
	}

	return port, nil
}

func (r *fileRegistry) Release(workspace string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocs, err := r.load()
	if err != nil {
		return err
	}

	filtered := make([]Allocation, 0, len(allocs))
	for _, a := range allocs {
		if a.WorkspaceName != workspace {
			filtered = append(filtered, a)
		}
	}

	return r.save(filtered)
}

func (r *fileRegistry) GetAllocations(workspace string) (map[string]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocs, err := r.load()
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, a := range allocs {
		if a.WorkspaceName == workspace {
			result[a.ServiceName] = a.Port
		}
	}

	return result, nil
}

func (r *fileRegistry) CheckConflicts() ([]Conflict, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	allocs, err := r.load()
	if err != nil {
		return nil, err
	}

	// Group allocations by port.
	byPort := make(map[int][]Allocation)
	for _, a := range allocs {
		byPort[a.Port] = append(byPort[a.Port], a)
	}

	var conflicts []Conflict
	for port, group := range byPort {
		if len(group) < 2 {
			continue
		}
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				conflicts = append(conflicts, Conflict{
					Port:       port,
					WorkspaceA: group[i].WorkspaceName,
					WorkspaceB: group[j].WorkspaceName,
					Service:    group[i].ServiceName,
				})
			}
		}
	}

	return conflicts, nil
}

func (r *fileRegistry) ListAll() ([]Allocation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.load()
}

// findFreePort scans the configured range for the first unoccupied port.
func (r *fileRegistry) findFreePort(usedPorts map[int]string) (int, error) {
	for p := r.portRange.Min; p <= r.portRange.Max; p++ {
		if _, taken := usedPorts[p]; !taken {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free ports in range %d-%d", r.portRange.Min, r.portRange.Max)
}

// load reads allocations from the state file. Returns empty slice if file doesn't exist.
func (r *fileRegistry) load() ([]Allocation, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading port state file: %w", err)
	}

	var allocs []Allocation
	if err := json.Unmarshal(data, &allocs); err != nil {
		return nil, fmt.Errorf("parsing port state file: %w", err)
	}

	return allocs, nil
}

// save writes allocations to the state file, creating directories if needed.
func (r *fileRegistry) save(allocs []Allocation) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return fmt.Errorf("creating port state directory: %w", err)
	}

	data, err := json.MarshalIndent(allocs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling port state: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0o644); err != nil {
		return fmt.Errorf("writing port state file: %w", err)
	}

	return nil
}
