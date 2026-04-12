package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the TUI.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Start   key.Binding
	Stop    key.Binding
	Destroy key.Binding
	SSH     key.Binding
	Logs    key.Binding
	Refresh key.Binding
	Filter  key.Binding
	Back    key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "down"),
		),
		Start: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "start"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop"),
		),
		Destroy: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "destroy"),
		),
		SSH: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "ssh"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// helpText returns a formatted help string for the help bar.
func helpText(km KeyMap) string {
	return "  s:start  x:stop  d:destroy  enter:ssh  l:logs  r:refresh  /:filter  q:quit"
}
