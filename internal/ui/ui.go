package ui

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/junixlabs/devbox/internal/workspace"
)

var noColor bool

// SetNoColor disables colored output globally.
func SetNoColor(v bool) {
	noColor = v
	color.NoColor = v
}

// StatusColor returns the status string colored by state.
func StatusColor(status workspace.Status) string {
	switch status {
	case workspace.StatusRunning:
		return color.GreenString(string(status))
	case workspace.StatusStopped:
		return color.YellowString(string(status))
	case workspace.StatusCreating:
		return color.CyanString(string(status))
	case workspace.StatusError:
		return color.RedString(string(status))
	default:
		return string(status)
	}
}

// StartSpinner creates and starts a spinner with the given message.
func StartSpinner(msg string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + msg
	if noColor {
		s.Writer = os.Stderr
	} else {
		s.Writer = os.Stderr
		s.Color("cyan")
	}
	s.Start()
	return s
}

// StopSpinner stops the spinner and prints a final status symbol.
func StopSpinner(s *spinner.Spinner, success bool) {
	s.Stop()
	if success {
		fmt.Fprintln(os.Stderr, color.GreenString("✓")+" "+s.Suffix[1:])
	} else {
		fmt.Fprintln(os.Stderr, color.RedString("✗")+" "+s.Suffix[1:])
	}
}

// PrintTable prints aligned tabular output with colored status column.
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, color.New(color.Bold).Sprint(h))
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// PrintUpSuccess prints the success output block for devbox up.
func PrintUpSuccess(name, server, url string, ports map[string]int) {
	fmt.Println()
	fmt.Printf("  %s Workspace %s created on %s\n\n",
		color.GreenString("✓"), color.CyanString(name), server)
	fmt.Printf("  %s  ssh %s\n", color.New(color.Bold).Sprint("SSH:"), server)
	if url != "" {
		fmt.Printf("  %s  %s\n", color.New(color.Bold).Sprint("URL:"), url)
	}
	fmt.Printf("  %s  zed ssh://%s//workspace\n", color.New(color.Bold).Sprint("Zed:"), server)
	for pname, port := range ports {
		fmt.Printf("  %s %s -> %d\n", color.New(color.Bold).Sprint("Port:"), pname, port)
	}
	fmt.Println()
}
