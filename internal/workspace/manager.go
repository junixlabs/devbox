package workspace

import "fmt"

// NewManager creates a workspace Manager.
// TODO(ISS-26): replace stub with real Docker/SSH implementation.
func NewManager() Manager {
	return &stubManager{}
}

type stubManager struct{}

func (m *stubManager) Create(name, project, branch string) (*Workspace, error) {
	return nil, fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) Start(name string) error {
	return fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) Stop(name string) error {
	return fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) Destroy(name string) error {
	return fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) List() ([]Workspace, error) {
	return nil, fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) Get(name string) (*Workspace, error) {
	return nil, fmt.Errorf("workspace manager not implemented (see ISS-26)")
}

func (m *stubManager) SSH(name string) error {
	return fmt.Errorf("workspace manager not implemented (see ISS-26)")
}
