package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/workspace"
)

type activeView int

const (
	viewList activeView = iota
	viewLogs
)

// refreshInterval is how often the workspace list auto-refreshes.
const refreshInterval = 5 * time.Second

type tickMsg time.Time

// Model is the root Bubble Tea model for the devbox TUI dashboard.
type Model struct {
	list     ListModel
	logs     LogModel
	active   activeView
	manager  workspace.Manager
	executor ssh.Executor
	keys     KeyMap
	width    int
	height   int
}

// New creates the root TUI model.
func New(mgr workspace.Manager, sshExec ssh.Executor) Model {
	keys := DefaultKeyMap()
	return Model{
		list:     NewListModel(mgr, keys),
		manager:  mgr,
		executor: sshExec,
		keys:     keys,
		active:   viewList,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		if m.active == viewLogs {
			m.logs.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tickMsg:
		if m.active == viewList {
			return m, tea.Batch(m.list.refreshWorkspaces(), tickCmd())
		}
		return m, tickCmd()

	case tea.KeyMsg:
		// ctrl+c always quits.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// q quits from list view (not during filtering).
		if m.active == viewList && msg.String() == "q" && !m.list.filtering {
			return m, tea.Quit
		}

	case viewLogsMsg:
		m.active = viewLogs
		m.logs = NewLogModel(msg.workspace, m.executor, m.keys)
		m.logs.SetSize(m.width, m.height)
		return m, m.logs.Init()

	case sshRequestMsg:
		return m, m.execSSH(msg.name)

	case backToListMsg:
		m.active = viewList
		return m, m.list.refreshWorkspaces()
	}

	switch m.active {
	case viewList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	case viewLogs:
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.active {
	case viewLogs:
		return m.logs.View()
	default:
		return m.list.View()
	}
}

// execSSH launches an SSH session by suspending the TUI via tea.ExecProcess.
func (m Model) execSSH(name string) tea.Cmd {
	if m.manager == nil {
		return func() tea.Msg {
			return workspaceActionDoneMsg{action: "ssh", name: name, err: fmt.Errorf("no workspace manager")}
		}
	}

	ws, err := m.manager.Get(name)
	if err != nil {
		return func() tea.Msg {
			return workspaceActionDoneMsg{action: "ssh", name: name, err: err}
		}
	}

	containerName := ws.Name + "-" + tuiFirstService(ws.Services) + "-1"
	sshCmd := fmt.Sprintf("docker exec -it '%s' /bin/sh", containerName)
	c := exec.Command("ssh", "-t", ws.ServerHost, sshCmd)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return workspaceActionDoneMsg{action: "ssh", name: name, err: err}
	})
}

// tuiFirstService returns the base name of the first service, or "app" as default.
func tuiFirstService(services []string) string {
	if len(services) == 0 {
		return "app"
	}
	svc := services[0]
	if i := strings.LastIndex(svc, ":"); i != -1 {
		svc = svc[:i]
	}
	if i := strings.LastIndex(svc, "/"); i != -1 {
		svc = svc[i+1:]
	}
	return svc
}

// Run starts the TUI program with alt-screen.
func Run(mgr workspace.Manager, sshExec ssh.Executor) error {
	p := tea.NewProgram(New(mgr, sshExec), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
