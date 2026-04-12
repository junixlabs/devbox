package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Header styles.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			PaddingLeft(1)

	// Table styles.
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	normalStyle = lipgloss.NewStyle()

	// Status badge styles.
	statusRunning  = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	statusStopped  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	statusCreating = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	statusError    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

	// Help bar style.
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(1).
			PaddingTop(1)

	// Filter input style.
	filterPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	// Log viewer header.
	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			PaddingLeft(1)

	// Dim text for empty states.
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Error message style.
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)
