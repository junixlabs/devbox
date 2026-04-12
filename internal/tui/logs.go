package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/junixlabs/devbox/internal/docker"
	"github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/workspace"
)

// LogModel displays streaming logs for a workspace.
type LogModel struct {
	viewport      viewport.Model
	workspace     workspace.Workspace
	executor      ssh.Executor
	content       strings.Builder
	lines         <-chan string
	cancel        context.CancelFunc
	ready         bool
	done          bool
	width, height int
	keys          KeyMap
}

// logLineMsg carries a new chunk of log output.
type logLineMsg string

// logDoneMsg signals log streaming finished.
type logDoneMsg struct{}

// backToListMsg requests returning to the workspace list.
type backToListMsg struct{}

// NewLogModel creates a log viewer for a specific workspace.
// Channel and cancel func are set up here (not in Init) so they survive
// bubbletea's value-receiver copy semantics in Init()/Update().
func NewLogModel(ws workspace.Workspace, exec ssh.Executor, keys KeyMap) LogModel {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan string, 64)

	m := LogModel{
		workspace: ws,
		executor:  exec,
		keys:      keys,
		cancel:    cancel,
		lines:     ch,
	}

	if exec != nil {
		go m.streamToChannel(ctx, ch)
	} else {
		close(ch)
	}

	return m
}

func (m LogModel) Init() tea.Cmd {
	if m.executor == nil {
		return func() tea.Msg {
			return logLineMsg(errStyle.Render("[error] SSH not available — logs require SSH connectivity"))
		}
	}
	return m.waitForLine()
}

func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	switch msg := msg.(type) {

	case logLineMsg:
		m.content.WriteString(string(msg))
		m.content.WriteString("\n")
		if m.ready {
			m.viewport.SetContent(m.content.String())
			m.viewport.GotoBottom()
		}
		if !m.done {
			return m, m.waitForLine()
		}
		return m, nil

	case logDoneMsg:
		m.done = true
		m.content.WriteString(dimStyle.Render("\n[stream ended]"))
		if m.ready {
			m.viewport.SetContent(m.content.String())
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.String() == "esc" || msg.String() == "q":
			if m.cancel != nil {
				m.cancel()
			}
			return m, func() tea.Msg { return backToListMsg{} }
		}

		if m.ready {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m LogModel) View() string {
	var b strings.Builder

	title := fmt.Sprintf("logs: %s (%s)", m.workspace.Name, m.workspace.ServerHost)
	b.WriteString(logHeaderStyle.Render(title))
	b.WriteString("\n\n")

	if !m.ready {
		b.WriteString(dimStyle.Render("  Initializing log viewer..."))
		b.WriteString("\n")
	} else {
		b.WriteString(m.viewport.View())
		b.WriteString("\n")
	}

	b.WriteString(helpStyle.Render("  esc/q:back  up/down:scroll"))
	b.WriteString("\n")

	return b.String()
}

func (m *LogModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	headerH := 3
	footerH := 2
	vpH := h - headerH - footerH
	if vpH < 1 {
		vpH = 1
	}
	if !m.ready {
		m.viewport = viewport.New(w, vpH)
		m.ready = true
	} else {
		m.viewport.Width = w
		m.viewport.Height = vpH
	}
}

// streamToChannel runs in a goroutine, streaming docker compose logs into ch.
func (m LogModel) streamToChannel(ctx context.Context, ch chan<- string) {
	defer close(ch)

	composePath := docker.WorkspaceBaseDir + "/" + m.workspace.Name + "/docker-compose.yml"
	cmd := fmt.Sprintf("docker compose -f '%s' logs --tail 100 -f 2>&1", composePath)

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		defer pw.Close()
		_ = m.executor.RunStream(ctx, m.workspace.ServerHost, cmd, pw, pw)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		select {
		case ch <- scanner.Text():
		case <-ctx.Done():
			return
		}
	}
}

func (m LogModel) waitForLine() tea.Cmd {
	ch := m.lines
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logDoneMsg{}
		}
		return logLineMsg(line)
	}
}
