package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "devbox",
		Short: "Manage remote development workspaces",
		Long:  "devbox turns any Linux machine into a ready-to-use dev environment in one command.\nNo cloud, no DevOps required.",
		Version: version,
	}

	rootCmd.AddCommand(upCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(destroyCmd())
	rootCmd.AddCommand(sshCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func upCmd() *cobra.Command {
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
			fmt.Printf("Starting workspace for project: %s\n", project)
			fmt.Println("not implemented yet")
			return nil
		},
	}
	cmd.Flags().String("branch", "", "Git branch to checkout")
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	return cmd
}

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop [workspace]",
		Short: "Stop a running workspace",
		Long:  "Stop a running workspace without destroying it.\nThe workspace can be started again with 'devbox up'.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Stopping workspace: %s\n", args[0])
			fmt.Println("not implemented yet")
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all workspaces",
		Long:    "List all workspaces across all configured servers.\nShows status, project, branch, and server for each workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Listing workspaces...")
			fmt.Println("not implemented yet")
			return nil
		},
	}
}

func destroyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destroy [workspace]",
		Aliases: []string{"rm"},
		Short:   "Destroy a workspace",
		Long:    "Permanently destroy a workspace and all its data.\nThis removes the container, volumes, and cloned source code.",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Destroying workspace: %s\n", args[0])
			fmt.Println("not implemented yet")
			return nil
		},
	}
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	return cmd
}

func sshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ssh [workspace]",
		Short: "SSH into a workspace",
		Long:  "Open an SSH session into a running workspace.\nIf the workspace is stopped, it will be started first.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Connecting to workspace: %s\n", args[0])
			fmt.Println("not implemented yet")
			return nil
		},
	}
}
