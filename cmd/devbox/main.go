package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/doctor"
	devboxerr "github.com/junixlabs/devbox/internal/errors"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/tailscale"
	"github.com/junixlabs/devbox/internal/ui"
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0-dev"
	verbose bool
	noColor bool
)

func main() {
	wm := workspace.NewManager()

	rootCmd := &cobra.Command{
		Use:          "devbox",
		Short:        "Manage remote development workspaces",
		Long:         "devbox turns any Linux machine into a ready-to-use dev environment in one command.\nNo cloud, no DevOps required.",
		Version:      version,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			level := slog.LevelInfo
			if verbose {
				level = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
			ui.SetNoColor(noColor)
		},
	}
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(upCmd(wm))
	rootCmd.AddCommand(stopCmd(wm))
	rootCmd.AddCommand(listCmd(wm))
	rootCmd.AddCommand(destroyCmd(wm))
	rootCmd.AddCommand(sshCmd(wm))
	rootCmd.AddCommand(doctorCmd())

	if err := rootCmd.Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}

// printError formats errors with suggestions when available.
func printError(err error) {
	var s devboxerr.Suggestible
	if errors.As(err, &s) && s.GetSuggestion() != "" {
		fmt.Fprintf(os.Stderr, "Hint: %s\n", s.GetSuggestion())
	}
}

// remoteRunner returns a tailscale.CommandRunner that executes commands on a
// remote server via SSH.
func remoteRunner(sshExec devboxssh.Executor, server string) tailscale.CommandRunner {
	return func(command string, args ...string) ([]byte, error) {
		parts := make([]string, 0, len(args)+1)
		parts = append(parts, command)
		parts = append(parts, args...)
		stdout, _, err := sshExec.Run(context.Background(), server, strings.Join(parts, " "))
		return []byte(stdout), err
	}
}

func upCmd(wm workspace.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up [project]",
		Short: "Create and start a workspace",
		Long:  "Create a new workspace (or start an existing one) for the given project.\nReads configuration from devbox.yaml in the project directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project := "."
			if len(args) > 0 {
				project = args[0]
			}

			cfg, err := config.LoadFromDir(project)
			if err != nil {
				return fmt.Errorf("devbox up: %w", err)
			}

			// Apply flag overrides
			if s, _ := cmd.Flags().GetString("server"); s != "" {
				cfg.Server = s
			}
			if b, _ := cmd.Flags().GetString("branch"); b != "" {
				cfg.Branch = b
			}

			if cfg.Server == "" {
				return fmt.Errorf("devbox up: server is required — add 'server:' to devbox.yaml or use --server flag")
			}

			spin := ui.StartSpinner("Starting workspace...")
			ws, err := wm.Create(workspace.CreateParams{
				Name:     cfg.Name,
				Server:   cfg.Server,
				Repo:     cfg.Repo,
				Branch:   cfg.Branch,
				Services: cfg.Services,
				Ports:    cfg.Ports,
				Env:      cfg.Env,
			})
			if err != nil {
				// If workspace already exists, start it instead.
				var wsErr *workspace.WorkspaceError
				if errors.As(err, &wsErr) && strings.Contains(wsErr.Message, "already exists") {
					if startErr := wm.Start(cfg.Name); startErr != nil {
						ui.StopSpinner(spin, false)
						return fmt.Errorf("devbox up: %w", startErr)
					}
					ws, err = wm.Get(cfg.Name)
					if err != nil {
						ui.StopSpinner(spin, false)
						return fmt.Errorf("devbox up: %w", err)
					}
				} else {
					ui.StopSpinner(spin, false)
					return fmt.Errorf("devbox up: %w", err)
				}
			}

			// Expose ports via Tailscale on the remote server
			sshExec, err := devboxssh.New()
			if err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox up: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, cfg.Server))
			for name, port := range cfg.Ports {
				if err := tm.Serve(port, ws.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to expose port %s (%d): %v\n", name, port, err)
				}
			}
			ui.StopSpinner(spin, true)

			tsStatus, _ := tm.Status()
			url := ""
			if tsStatus != nil {
				url = tailscale.WorkspaceURL(tsStatus.Hostname, tsStatus.TailnetName)
			}
			ui.PrintUpSuccess(ws.Name, cfg.Server, url, cfg.Ports)

			return nil
		},
	}
	cmd.Flags().String("branch", "", "Git branch to checkout")
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	return cmd
}

func stopCmd(wm workspace.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "stop [workspace]",
		Short: "Stop a running workspace",
		Long:  "Stop a running workspace without destroying it.\nThe workspace can be started again with 'devbox up'.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			ws, err := wm.Get(name)
			if err != nil {
				return fmt.Errorf("devbox stop: %w", err)
			}

			spin := ui.StartSpinner("Stopping workspace...")

			if err := wm.Stop(name); err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox stop: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox stop: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, ws.ServerHost))
			for _, port := range ws.Ports {
				if err := tm.Unserve(port); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to unserve port %d: %v\n", port, err)
				}
			}

			ui.StopSpinner(spin, true)
			fmt.Printf("Workspace %q stopped\n", name)
			return nil
		},
	}
}

func listCmd(wm workspace.Manager) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all workspaces",
		Long:    "List all workspaces across all configured servers.\nShows status, project, branch, and server for each workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaces, err := wm.List()
			if err != nil {
				return fmt.Errorf("devbox list: %w", err)
			}

			if len(workspaces) == 0 {
				fmt.Println("No workspaces found")
				return nil
			}

			headers := []string{"NAME", "STATUS", "SERVER", "PORTS", "CREATED"}
			rows := make([][]string, 0, len(workspaces))
			for _, ws := range workspaces {
				rows = append(rows, []string{
					ws.Name,
					ui.StatusColor(ws.Status),
					ws.ServerHost,
					formatPorts(ws.Ports),
					timeAgo(ws.CreatedAt),
				})
			}
			ui.PrintTable(headers, rows)

			return nil
		},
	}
}

func destroyCmd(wm workspace.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destroy [workspace]",
		Aliases: []string{"rm"},
		Short:   "Destroy a workspace",
		Long:    "Permanently destroy a workspace and all its data.\nThis removes the container, volumes, and cloned source code.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			force, _ := cmd.Flags().GetBool("force")
			if !force {
				fmt.Printf("Are you sure you want to destroy workspace %q? [y/N]: ", name)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(response)
				if response != "y" && response != "Y" {
					fmt.Println("Aborted")
					return nil
				}
			}

			ws, err := wm.Get(name)
			if err != nil {
				return fmt.Errorf("devbox destroy: %w", err)
			}

			spin := ui.StartSpinner("Destroying workspace...")

			if err := wm.Destroy(name); err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox destroy: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox destroy: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, ws.ServerHost))
			for _, port := range ws.Ports {
				if err := tm.Unserve(port); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to unserve port %d: %v\n", port, err)
				}
			}

			ui.StopSpinner(spin, true)
			fmt.Printf("Workspace %q destroyed\n", name)
			return nil
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	return cmd
}

func sshCmd(wm workspace.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "ssh [workspace]",
		Short: "SSH into a workspace",
		Long:  "Open an SSH session into a running workspace.\nIf the workspace is stopped, it will be started first.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			ws, err := wm.Get(name)
			if err != nil {
				return fmt.Errorf("devbox ssh: %w", err)
			}

			if ws.Status == workspace.StatusStopped {
				fmt.Println("Starting workspace...")
				if err := wm.Start(name); err != nil {
					return fmt.Errorf("devbox ssh: failed to start workspace: %w", err)
				}
			}

			if err := wm.SSH(name); err != nil {
				return fmt.Errorf("devbox ssh: %w", err)
			}

			return nil
		},
	}
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new devbox.yaml configuration",
		Long:  "Interactively create a devbox.yaml configuration file for the current project.\nDetects existing Docker and devcontainer configs and offers smart defaults.",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := config.DefaultConfigFile

			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("devbox init: %s already exists", configPath)
			}

			scanner := bufio.NewScanner(os.Stdin)
			w := os.Stdout

			fromCompose, _ := cmd.Flags().GetString("from-compose")
			if fromCompose != "" {
				cfg, err := config.ConvertFromCompose(fromCompose)
				if err != nil {
					return fmt.Errorf("devbox init: %w", err)
				}

				fmt.Fprintf(w, "Converted %s: %d service(s), %d port(s)\n\n", fromCompose, len(cfg.Services), len(cfg.Ports))

				dirName := filepath.Base(mustGetwd())
				cfg.Name = config.PromptString(w, scanner, "Project name", dirName)
				cfg.Server = config.PromptRequired(w, scanner, "Server")

				if err := config.WriteConfig(cfg, configPath); err != nil {
					return fmt.Errorf("devbox init: %w", err)
				}
				fmt.Fprintf(w, "\nCreated %s\n", configPath)
				return nil
			}

			// Detect existing configs
			detected := config.DetectExistingConfigs(".")
			for _, d := range detected {
				switch d.Type {
				case "compose":
					fmt.Fprintf(w, "Detected %s — use --from-compose %s to convert\n", d.Path, d.Path)
				case "devcontainer":
					fmt.Fprintf(w, "Detected %s\n", d.Path)
				case "dockerfile":
					fmt.Fprintf(w, "Detected %s\n", d.Path)
				}
			}
			if len(detected) > 0 {
				fmt.Fprintln(w)
			}

			// Interactive prompts
			dirName := filepath.Base(mustGetwd())
			name := config.PromptString(w, scanner, "Project name", dirName)
			server := config.PromptRequired(w, scanner, "Server")
			repo := config.PromptString(w, scanner, "Git repo", "")
			servicesInput := config.PromptString(w, scanner, "Services (comma-separated, e.g. mysql:8.0,redis:7)", "")
			portsInput := config.PromptString(w, scanner, "Ports (comma-separated, e.g. app:8080,db:3306)", "")

			cfg := &config.DevboxConfig{
				Name:     name,
				Server:   server,
				Repo:     repo,
				Services: config.ParseCommaSeparated(servicesInput),
				Ports:    config.ParsePortMappings(portsInput),
			}

			if err := config.WriteConfig(cfg, configPath); err != nil {
				return fmt.Errorf("devbox init: %w", err)
			}
			fmt.Fprintf(w, "\nCreated %s\n", configPath)
			return nil
		},
	}
	cmd.Flags().String("from-compose", "", "Convert from an existing docker-compose.yml")
	return cmd
}

func doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites and server health",
		Long:  "Run health checks against the local machine and remote server.\nChecks SSH connectivity, Docker, Tailscale, git, and disk space.",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetString("server")

			if server == "" {
				cfg, err := config.LoadFromDir(".")
				if err == nil {
					server = cfg.Server
				}
			}

			if server == "" {
				return fmt.Errorf("devbox doctor: no server specified — use --server flag or create devbox.yaml")
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox doctor: %w", err)
			}
			defer sshExec.Close()

			fmt.Printf("Running health checks against %s...\n\n", server)

			allPassed := doctor.Run(cmd.Context(), os.Stdout, sshExec, server)
			if !allPassed {
				os.Exit(1)
			}

			return nil
		},
	}
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	return cmd
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func formatPorts(ports map[string]int) string {
	if len(ports) == 0 {
		return "-"
	}
	pairs := make([]string, 0, len(ports))
	for name, port := range ports {
		pairs = append(pairs, fmt.Sprintf("%s:%d", name, port))
	}
	return strings.Join(pairs, ", ")
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
