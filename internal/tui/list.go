package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/junixlabs/devbox/internal/workspace"
)

// ListModel displays workspaces in a navigable table with actions.
type ListModel struct {
	workspaces []workspace.Workspace
	filtered   []workspace.Workspace
	cursor     int
	manager    workspace.Manager
	filter     textinput.Model
	filtering  bool
	width      int
	height     int
	keys       KeyMap
	err        error
	statusMsg  string
}

// NewListModel creates a new workspace list view.
func NewListModel(mgr workspace.Manager, keys KeyMap) ListModel {
	ti := textinput.New()
	ti.Placeholder = "filter workspaces..."
	ti.PromptStyle = filterPromptStyle
	ti.Prompt = "/ "

	return ListModel{
		manager: mgr,
		filter:  ti,
		keys:    keys,
	}
}

// workspacesLoadedMsg carries refreshed workspace data.
type workspacesLoadedMsg struct {
	workspaces []workspace.Workspace
	err        error
}

// workspaceActionDoneMsg signals that an async action completed.
type workspaceActionDoneMsg struct {
	action string
	name   string
	err    error
}

// viewLogsMsg requests switching to the log viewer.
type viewLogsMsg struct {
	workspace workspace.Workspace
}

// sshRequestMsg requests launching an SSH session.
type sshRequestMsg struct {
	name string
}

func (m ListModel) Init() tea.Cmd {
	return m.refreshWorkspaces()
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {

	case workspacesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.workspaces = msg.workspaces
		m.applyFilter()
		m.err = nil
		return m, nil

	case workspaceActionDoneMsg:
		if msg.err != nil {
			m.statusMsg = errStyle.Render(fmt.Sprintf("%s failed: %v", msg.action, msg.err))
		} else {
			m.statusMsg = fmt.Sprintf("%s: %s done", msg.action, msg.name)
		}
		return m, m.refreshWorkspaces()

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

func (m ListModel) updateNormal(msg tea.KeyMsg) (ListModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshWorkspaces()
	case key.Matches(msg, m.keys.Start):
		if ws := m.selected(); ws != nil {
			return m, m.doAction("start", ws.Name)
		}
	case key.Matches(msg, m.keys.Stop):
		if ws := m.selected(); ws != nil {
			return m, m.doAction("stop", ws.Name)
		}
	case key.Matches(msg, m.keys.Destroy):
		if ws := m.selected(); ws != nil {
			return m, m.doAction("destroy", ws.Name)
		}
	case key.Matches(msg, m.keys.SSH):
		if ws := m.selected(); ws != nil {
			return m, func() tea.Msg { return sshRequestMsg{name: ws.Name} }
		}
	case key.Matches(msg, m.keys.Logs):
		if ws := m.selected(); ws != nil {
			return m, func() tea.Msg { return viewLogsMsg{workspace: *ws} }
		}
	}
	return m, nil
}

func (m ListModel) updateFilter(msg tea.KeyMsg) (ListModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.filtering = false
		m.filter.SetValue("")
		m.filter.Blur()
		m.applyFilter()
		return m, nil
	case tea.KeyEnter:
		m.filtering = false
		m.filter.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m ListModel) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("devbox workspaces"))
	b.WriteString("\n\n")

	if m.manager == nil {
		b.WriteString(dimStyle.Render("  No workspace manager configured"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Implement workspace.Manager to use the dashboard"))
		b.WriteString("\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.filtering || m.filter.Value() != "" {
		b.WriteString("  ")
		b.WriteString(m.filter.View())
		b.WriteString("\n\n")
	}

	if len(m.filtered) == 0 {
		if len(m.workspaces) == 0 {
			b.WriteString(dimStyle.Render("  No workspaces found"))
		} else {
			b.WriteString(dimStyle.Render("  No workspaces match filter"))
		}
		b.WriteString("\n")
	} else {
		// Column headers.
		header := fmt.Sprintf("  %-30s %-10s %-10s %-20s %s", "NAME", "STATUS", "USER", "SERVER", "PORTS")
		b.WriteString(dimStyle.Render(header))
		b.WriteString("\n")

		// Rows.
		maxRows := m.height - 10 // Leave room for header, filter, help.
		if maxRows < 3 {
			maxRows = 3
		}
		for i, ws := range m.filtered {
			if i >= maxRows {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.filtered)-maxRows)))
				b.WriteString("\n")
				break
			}
			line := fmt.Sprintf("  %-30s %s %-10s %-20s %s",
				truncate(ws.Name, 30),
				renderStatus(ws.Status),
				truncate(ws.User, 10),
				truncate(ws.ServerHost, 20),
				formatPorts(ws.Ports),
			)
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(normalStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString("  " + m.statusMsg)
	}

	b.WriteString(helpStyle.Render(helpText(m.keys)))
	b.WriteString("\n")

	return b.String()
}

func (m *ListModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ListModel) selected() *workspace.Workspace {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	ws := m.filtered[m.cursor]
	return &ws
}

func (m *ListModel) applyFilter() {
	query := strings.ToLower(m.filter.Value())
	if query == "" {
		m.filtered = m.workspaces
	} else {
		m.filtered = nil
		for _, ws := range m.workspaces {
			if strings.Contains(strings.ToLower(ws.Name), query) ||
				strings.Contains(strings.ToLower(ws.Project), query) {
				m.filtered = append(m.filtered, ws)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m ListModel) refreshWorkspaces() tea.Cmd {
	return func() tea.Msg {
		if m.manager == nil {
			return workspacesLoadedMsg{}
		}
		wsList, err := m.manager.List(workspace.ListOptions{All: true})
		if err != nil {
			return workspacesLoadedMsg{err: err}
		}
		return workspacesLoadedMsg{workspaces: wsList}
	}
}

func (m ListModel) doAction(action, name string) tea.Cmd {
	return func() tea.Msg {
		if m.manager == nil {
			return workspaceActionDoneMsg{action: action, name: name, err: fmt.Errorf("no workspace manager configured")}
		}
		var err error
		switch action {
		case "start":
			err = m.manager.Start(name)
		case "stop":
			err = m.manager.Stop(name)
		case "destroy":
			err = m.manager.Destroy(name)
		}
		return workspaceActionDoneMsg{action: action, name: name, err: err}
	}
}

func renderStatus(s workspace.Status) string {
	padded := fmt.Sprintf("%-10s", string(s))
	switch s {
	case workspace.StatusRunning:
		return statusRunning.Render(padded)
	case workspace.StatusStopped:
		return statusStopped.Render(padded)
	case workspace.StatusCreating:
		return statusCreating.Render(padded)
	case workspace.StatusError:
		return statusError.Render(padded)
	default:
		return padded
	}
}

func formatPorts(ports map[string]int) string {
	if len(ports) == 0 {
		return "-"
	}
	names := make([]string, 0, len(ports))
	for name := range ports {
		names = append(names, name)
	}
	sort.Strings(names)
	pairs := make([]string, 0, len(ports))
	for _, name := range names {
		pairs = append(pairs, fmt.Sprintf("%s:%d", name, ports[name]))
	}
	return strings.Join(pairs, " ")
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("~")
}
