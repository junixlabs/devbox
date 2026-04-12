package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/junixlabs/devbox/internal/workspace"
)

// mockManager implements workspace.Manager for testing.
type mockManager struct {
	workspaces []workspace.Workspace
	startErr   error
	stopErr    error
	destroyErr error
}

func (m *mockManager) Create(params workspace.CreateParams) (*workspace.Workspace, error) {
	return nil, nil
}

func (m *mockManager) Start(name string) error { return m.startErr }
func (m *mockManager) Stop(name string) error  { return m.stopErr }
func (m *mockManager) Destroy(name string) error { return m.destroyErr }

func (m *mockManager) List(opts workspace.ListOptions) ([]workspace.Workspace, error) {
	return m.workspaces, nil
}

func (m *mockManager) Get(name string) (*workspace.Workspace, error) {
	for _, ws := range m.workspaces {
		if ws.Name == name {
			return &ws, nil
		}
	}
	return nil, &workspace.WorkspaceError{Message: "not found"}
}

func (m *mockManager) SSH(name string) error { return nil }

func (m *mockManager) DockerStats(host string) (map[string]*workspace.ResourceUsage, error) {
	return nil, nil
}

func (m *mockManager) ServerResources(host string) (*workspace.ServerResourceInfo, error) {
	return nil, nil
}

func testWorkspaces() []workspace.Workspace {
	return []workspace.Workspace{
		{
			Name:       "alice-myapp-main",
			User:       "alice",
			Project:    "myapp",
			Branch:     "main",
			Status:     workspace.StatusRunning,
			ServerHost: "dev-1",
			Ports:      map[string]int{"web": 8080},
			CreatedAt:  time.Now().Add(-1 * time.Hour),
		},
		{
			Name:       "bob-api-feature",
			User:       "bob",
			Project:    "api",
			Branch:     "feature",
			Status:     workspace.StatusStopped,
			ServerHost: "dev-2",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
		},
	}
}

func TestNewModel(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	m := New(mgr, nil)

	if m.active != viewList {
		t.Errorf("expected viewList, got %d", m.active)
	}
	if m.manager == nil {
		t.Error("manager should not be nil")
	}
}

func TestNewModelNilManager(t *testing.T) {
	m := New(nil, nil)

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestListModelLoadWorkspaces(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	keys := DefaultKeyMap()
	lm := NewListModel(mgr, keys)

	cmd := lm.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}

	msg := cmd()
	loaded, ok := msg.(workspacesLoadedMsg)
	if !ok {
		t.Fatalf("expected workspacesLoadedMsg, got %T", msg)
	}
	if len(loaded.workspaces) != 2 {
		t.Errorf("expected 2 workspaces, got %d", len(loaded.workspaces))
	}

	lm, _ = lm.Update(loaded)
	if len(lm.filtered) != 2 {
		t.Errorf("expected 2 filtered workspaces, got %d", len(lm.filtered))
	}
}

func TestListModelNavigation(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	keys := DefaultKeyMap()
	lm := NewListModel(mgr, keys)

	// Load workspaces.
	cmd := lm.Init()
	lm, _ = lm.Update(cmd())

	if lm.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", lm.cursor)
	}

	// Move down.
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if lm.cursor != 1 {
		t.Errorf("cursor should be 1 after down, got %d", lm.cursor)
	}

	// Move up.
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if lm.cursor != 0 {
		t.Errorf("cursor should be 0 after up, got %d", lm.cursor)
	}

	// Don't go below 0.
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if lm.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", lm.cursor)
	}
}

func TestListModelFilter(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	keys := DefaultKeyMap()
	lm := NewListModel(mgr, keys)

	// Load workspaces.
	cmd := lm.Init()
	lm, _ = lm.Update(cmd())

	// Enter filter mode.
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !lm.filtering {
		t.Error("should be in filter mode")
	}

	// Type "alice".
	for _, r := range "alice" {
		lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(lm.filtered) != 1 {
		t.Errorf("expected 1 filtered workspace, got %d", len(lm.filtered))
	}
	if lm.filtered[0].Name != "alice-myapp-main" {
		t.Errorf("expected alice-myapp-main, got %s", lm.filtered[0].Name)
	}

	// Esc to exit filter.
	lm, _ = lm.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if lm.filtering {
		t.Error("should not be in filter mode")
	}
	if len(lm.filtered) != 2 {
		t.Errorf("expected 2 filtered workspaces after esc, got %d", len(lm.filtered))
	}
}

func TestListModelActions(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	keys := DefaultKeyMap()
	lm := NewListModel(mgr, keys)

	// Load workspaces.
	cmd := lm.Init()
	lm, _ = lm.Update(cmd())

	// Press 's' to start.
	lm, actionCmd := lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if actionCmd == nil {
		t.Fatal("start action should return a command")
	}

	msg := actionCmd()
	done, ok := msg.(workspaceActionDoneMsg)
	if !ok {
		t.Fatalf("expected workspaceActionDoneMsg, got %T", msg)
	}
	if done.action != "start" {
		t.Errorf("expected action 'start', got %q", done.action)
	}
}

func TestViewSwitching(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	m := New(mgr, nil)

	// Load workspaces via list init.
	cmd := m.Init()
	var newModel tea.Model
	newModel, _ = m.Update(cmd())
	m = newModel.(Model)

	if m.active != viewList {
		t.Errorf("expected viewList, got %d", m.active)
	}

	// Trigger viewLogs.
	ws := testWorkspaces()[0]
	newModel, _ = m.Update(viewLogsMsg{workspace: ws})
	m = newModel.(Model)

	if m.active != viewLogs {
		t.Errorf("expected viewLogs, got %d", m.active)
	}

	// Trigger backToList.
	newModel, _ = m.Update(backToListMsg{})
	m = newModel.(Model)

	if m.active != viewList {
		t.Errorf("expected viewList after back, got %d", m.active)
	}
}

func TestListViewNilManager(t *testing.T) {
	keys := DefaultKeyMap()
	lm := NewListModel(nil, keys)

	view := lm.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// Should contain the "not configured" message.
	expected := "No workspace manager configured"
	if !strings.Contains(view, expected) {
		t.Errorf("view should contain %q, got:\n%s", expected, view)
	}
}

func TestRenderStatus(t *testing.T) {
	tests := []struct {
		status workspace.Status
		want   string
	}{
		{workspace.StatusRunning, "running"},
		{workspace.StatusStopped, "stopped"},
		{workspace.StatusCreating, "creating"},
		{workspace.StatusError, "error"},
	}
	for _, tt := range tests {
		got := renderStatus(tt.status)
		if !strings.Contains(got, tt.want) {
			t.Errorf("renderStatus(%s) = %q, should contain %q", tt.status, got, tt.want)
		}
	}
}

func TestWindowResize(t *testing.T) {
	mgr := &mockManager{workspaces: testWorkspaces()}
	m := New(mgr, nil)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(Model)

	if m.width != 120 || m.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", m.width, m.height)
	}
}

