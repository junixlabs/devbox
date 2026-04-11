package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// stateFile is the JSON structure persisted to disk.
type stateFile struct {
	Workspaces map[string]*Workspace `json:"workspaces"`
}

// stateStore manages workspace state in a local JSON file (~/.devbox/state.json).
type stateStore struct {
	path string
	mu   sync.Mutex
}

// newStateStore creates a stateStore, ensuring the parent directory exists.
func newStateStore() (*stateStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}
	dir := filepath.Join(home, ".devbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating state directory %s: %w", dir, err)
	}
	return &stateStore{path: filepath.Join(dir, "state.json")}, nil
}

// newStateStoreAt creates a stateStore at a specific path (for testing).
func newStateStoreAt(path string) *stateStore {
	return &stateStore{path: path}
}

// load reads workspaces without acquiring the mutex (caller must hold it).
func (s *stateStore) load() (map[string]*Workspace, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Workspace), nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	if sf.Workspaces == nil {
		return make(map[string]*Workspace), nil
	}
	return sf.Workspaces, nil
}

// save writes workspaces without acquiring the mutex (caller must hold it).
func (s *stateStore) save(workspaces map[string]*Workspace) error {
	sf := stateFile{Workspaces: workspaces}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Write to temp file then rename for atomicity.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming state file: %w", err)
	}
	return nil
}

// Load reads all workspaces from the state file.
// Returns an empty map if the file does not exist.
func (s *stateStore) Load() (map[string]*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// Get returns a single workspace by name, or nil if not found.
func (s *stateStore) Get(name string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.load()
	if err != nil {
		return nil, err
	}
	ws, ok := workspaces[name]
	if !ok {
		return nil, nil
	}
	return ws, nil
}

// Put adds or updates a workspace in the state file.
// The entire read-modify-write is atomic under one lock.
func (s *stateStore) Put(ws *Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.load()
	if err != nil {
		return err
	}
	workspaces[ws.Name] = ws
	return s.save(workspaces)
}

// Delete removes a workspace from the state file.
// The entire read-modify-write is atomic under one lock.
func (s *stateStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspaces, err := s.load()
	if err != nil {
		return err
	}
	delete(workspaces, name)
	return s.save(workspaces)
}
