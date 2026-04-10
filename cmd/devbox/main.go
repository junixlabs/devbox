package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/tailscale"
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	wm := workspace.NewManager()

	rootCmd := &cobra.Command{
		Use:          "devbox",
		Short:        "Manage remote development workspaces",
		Long:         "devbox turns any Linux machine into a ready-to-use dev environment in one command.\nNo cloud, no DevOps required.",
		Version:      version,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(upCmd(wm))
	rootCmd.AddCommand(stopCmd(wm))
	rootCmd.AddCommand(listCmd(wm))
	rootCmd.AddCommand(destroyCmd(wm))
	rootCmd.AddCommand(sshCmd(wm))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
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

			ws, err := wm.Create(cfg.Name, project, cfg.Branch)
			if err != nil {
				return fmt.Errorf("devbox up: %w", err)
			}

			// Expose ports via Tailscale on the remote server
			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox up: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, cfg.Server))
			for name, port := range cfg.Ports {
				if err := tm.Serve(port, ws.Name); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to expose port %s (%d): %v\n", name, port, err)
				}
			}

			tsStatus, _ := tm.Status()

			fmt.Printf("\nWorkspace %q created on %s\n\n", ws.Name, cfg.Server)
			fmt.Printf("  SSH:    ssh %s\n", cfg.Server)
			if tsStatus != nil {
				fmt.Printf("  URL:    %s\n", tailscale.WorkspaceURL(tsStatus.Hostname, tsStatus.TailnetName))
			}
			for name, port := range cfg.Ports {
				fmt.Printf("  Port:   %s -> %d\n", name, port)
			}
			fmt.Println()

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

			if err := wm.Stop(name); err != nil {
				return fmt.Errorf("devbox stop: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox stop: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, ws.ServerHost))
			for _, port := range ws.Ports {
				if err := tm.Unserve(port); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to unserve port %d: %v\n", port, err)
				}
			}

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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tSTATUS\tSERVER\tPORTS\tCREATED")
			for _, ws := range workspaces {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					ws.Name, ws.Status, ws.ServerHost,
					formatPorts(ws.Ports), timeAgo(ws.CreatedAt))
			}
			w.Flush()

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

			if err := wm.Destroy(name); err != nil {
				return fmt.Errorf("devbox destroy: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox destroy: %w", err)
			}
			defer sshExec.Close()

			tm := tailscale.NewManager(remoteRunner(sshExec, ws.ServerHost))
			for _, port := range ws.Ports {
				if err := tm.Unserve(port); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to unserve port %d: %v\n", port, err)
				}
			}

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
