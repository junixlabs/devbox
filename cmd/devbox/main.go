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
	"text/tabwriter"
	"time"

	"github.com/junixlabs/devbox/internal/config"
	"github.com/junixlabs/devbox/internal/doctor"
	devboxerr "github.com/junixlabs/devbox/internal/errors"
	"github.com/junixlabs/devbox/internal/identity"
	"github.com/junixlabs/devbox/internal/server"
	"github.com/junixlabs/devbox/internal/snapshot"
	devboxssh "github.com/junixlabs/devbox/internal/ssh"
	"github.com/junixlabs/devbox/internal/tailscale"
	"github.com/junixlabs/devbox/internal/ui"
	"github.com/junixlabs/devbox/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	version = "0.2.0"
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
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(snapshotCmd())
	rootCmd.AddCommand(restoreCmd())

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

// unservePorts tears down Tailscale serve entries for all workspace ports.
// Errors are logged as warnings but do not stop the operation.
func unservePorts(ws *workspace.Workspace) {
	sshExec, err := devboxssh.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to connect for port cleanup: %v\n", err)
		return
	}
	defer sshExec.Close()

	tm := tailscale.NewManager(remoteRunner(sshExec, ws.ServerHost))
	for _, port := range ws.Ports {
		if err := tm.Unserve(port); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to unserve port %d: %v\n", port, err)
		}
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

			if b, _ := cmd.Flags().GetString("branch"); b != "" {
				cfg.Branch = b
			}

			// Load server pool (best-effort).
			configPath, _ := server.DefaultConfigPath()
			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox up: %w", err)
			}
			defer sshExec.Close()

			pool, poolErr := server.NewFilePool(configPath, sshExec)
			var poolServers []server.Server
			if poolErr == nil {
				poolServers, _ = pool.List()
			}
			poolConfigured := len(poolServers) > 0

			// 3-tier server resolution: --server flag → config.Server → auto-select from pool.
			targetServer := cfg.Server
			serverFlag, _ := cmd.Flags().GetString("server")

			if serverFlag != "" {
				// Tier 1: --server flag — look up from pool by name.
				found := false
				for _, srv := range poolServers {
					if srv.Name == serverFlag {
						targetServer = server.SSHHost(&srv)
						found = true
						break
					}
				}
				if !found {
					// Not in pool — use as raw hostname (backward compat).
					targetServer = serverFlag
				}
			} else if targetServer == "" && poolConfigured {
				// Tier 3: Auto-select from pool.
				selector := server.NewLeastLoaded(sshExec)
				selected, err := selector.Select(cmd.Context(), poolServers)
				if err != nil {
					return fmt.Errorf("devbox up: %w", err)
				}
				targetServer = server.SSHHost(selected)
				fmt.Fprintf(os.Stderr, "Using server: %s (%s)\n", selected.Name, selected.Host)
			}
			// Tier 2: config.Server is already set in targetServer.

			cfg.Server = targetServer
			if err := cfg.ValidateForUp(poolConfigured); err != nil {
				return fmt.Errorf("devbox up: %w", err)
			}

			// Merge resource limits: server defaults <- workspace overrides.
			globalCfg, err := config.LoadGlobal()
			if err != nil {
				slog.Warn("failed to load global config", "error", err)
			}
			resources := config.MergeResources(
				globalCfg.ServerResourceDefaults(cfg.Server),
				cfg.Resources,
			)

			// Resolve user identity for workspace naming.
			idResolver := identity.NewResolver(nil)
			user := ""
			if id, err := idResolver.Current(); err == nil {
				user = id.Username
				slog.Debug("resolved user identity", "user", user, "source", id.Source)
			} else {
				slog.Debug("no user identity available, using unnamed workspace", "error", err)
			}

			wsName := cfg.Name
			if user != "" {
				wsName = workspace.FormatName(user, cfg.Name, cfg.Branch)
			}

			spin := ui.StartSpinner("Starting workspace...")
			ws, err := wm.Create(workspace.CreateParams{
				Name:      wsName,
				User:      user,
				Server:    cfg.Server,
				Repo:      cfg.Repo,
				Branch:    cfg.Branch,
				Services:  cfg.Services,
				Ports:     cfg.Ports,
				Env:       cfg.Env,
				Resources: resources,
			})
			if err != nil {
				// If workspace already exists, start it instead.
				var wsErr *workspace.WorkspaceError
				if errors.As(err, &wsErr) && strings.Contains(wsErr.Message, "already exists") {
					if startErr := wm.Start(wsName); startErr != nil {
						ui.StopSpinner(spin, false)
						return fmt.Errorf("devbox up: %w", startErr)
					}
					ws, err = wm.Get(wsName)
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
	cmd.Flags().String("server", "", "Target server name from pool (or hostname)")
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

			unservePorts(ws)
			ui.StopSpinner(spin, true)
			fmt.Printf("Workspace %q stopped\n", name)
			return nil
		},
	}
}

func listCmd(wm workspace.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workspaces",
		Long:    "List workspaces across all configured servers.\nBy default, shows only your workspaces. Use --all to show all users' workspaces.\nShows status, resource limits, and server for each workspace.\nUse --server to filter to a specific server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			allFlag, _ := cmd.Flags().GetBool("all")

			opts := workspace.ListOptions{All: allFlag}
			if !allFlag {
				idResolver := identity.NewResolver(nil)
				if id, err := idResolver.Current(); err == nil {
					opts.User = id.Username
				}
			}

			workspaces, err := wm.List(opts)
			if err != nil {
				return fmt.Errorf("devbox list: %w", err)
			}

			// Filter by --server if provided.
			serverFilter, _ := cmd.Flags().GetString("server")
			if serverFilter != "" {
				// Resolve server name from pool if possible.
				resolvedHost := serverFilter
				configPath, _ := server.DefaultConfigPath()
				if pool, err := server.NewFilePool(configPath, nil); err == nil {
					if servers, err := pool.List(); err == nil {
						for _, srv := range servers {
							if srv.Name == serverFilter {
								resolvedHost = server.SSHHost(&srv)
								break
							}
						}
					}
				}
				filtered := make([]workspace.Workspace, 0)
				for _, ws := range workspaces {
					if ws.ServerHost == resolvedHost || ws.ServerHost == serverFilter {
						filtered = append(filtered, ws)
					}
				}
				workspaces = filtered
			}

			if len(workspaces) == 0 {
				fmt.Println("No workspaces found")
				return nil
			}

			// Collect unique servers with running workspaces for live stats.
			serverHosts := make(map[string]bool)
			for _, ws := range workspaces {
				if ws.Status == workspace.StatusRunning {
					serverHosts[ws.ServerHost] = true
				}
			}

			// Fetch live docker stats per server (best-effort).
			allStats := make(map[string]*workspace.ResourceUsage)
			serverInfos := make(map[string]*workspace.ServerResourceInfo)
			for host := range serverHosts {
				stats, err := wm.DockerStats(host)
				if err != nil {
					slog.Debug("failed to fetch docker stats", "host", host, "error", err)
				} else {
					for k, v := range stats {
						allStats[k] = v
					}
				}
				info, err := wm.ServerResources(host)
				if err != nil {
					slog.Debug("failed to fetch server resources", "host", host, "error", err)
				} else {
					// Aggregate used resources from container stats.
					for _, s := range stats {
						if info.TotalCPUs > 0 {
							info.UsedCPUs += s.CPUPercent / 100.0 * float64(info.TotalCPUs)
						}
						info.UsedMemoryBytes += s.MemoryUsed
					}
					serverInfos[host] = info
				}
			}

			headers := []string{"NAME", "USER", "STATUS", "SERVER", "CPUS", "MEMORY", "CPU%", "MEM%", "PORTS", "CREATED"}
			rows := make([][]string, 0, len(workspaces))
			for _, ws := range workspaces {
				cpus := "-"
				mem := "-"
				if !ws.Resources.IsZero() {
					if ws.Resources.CPUs > 0 {
						cpus = fmt.Sprintf("%.1f", ws.Resources.CPUs)
					}
					if ws.Resources.Memory != "" {
						mem = ws.Resources.Memory
					}
				}
				cpuPct := "-"
				memPct := "-"
				// Match container name: workspace containers are named <name>-<service>-1
				for statName, ru := range allStats {
					if strings.HasPrefix(statName, ws.Name+"-") {
						cpuPct, memPct = workspace.FormatResourceUsage(ru)
						break
					}
				}
				rows = append(rows, []string{
					ws.Name,
					ws.User,
					ui.StatusColor(ws.Status),
					ws.ServerHost,
					cpus,
					mem,
					cpuPct,
					memPct,
					formatPorts(ws.Ports),
					timeAgo(ws.CreatedAt),
				})
			}
			ui.PrintTable(headers, rows)

			// Emit low-resource warnings to stderr.
			for host, info := range serverInfos {
				warnings := workspace.CheckLowResources(info, workspace.LowResourceThreshold)
				for _, w := range warnings {
					fmt.Fprintf(os.Stderr, "⚠ %s: %s\n", host, w)
				}
			}

			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Show all users' workspaces")
	cmd.Flags().String("server", "", "Filter workspaces by server name or hostname")
	return cmd
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

			unservePorts(ws)
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

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the server pool",
		Long:  "Add, remove, and list servers in the devbox server pool.\nServers are stored in ~/.config/devbox/servers.yaml.",
	}
	cmd.AddCommand(serverAddCmd())
	cmd.AddCommand(serverRemoveCmd())
	cmd.AddCommand(serverListCmd())
	return cmd
}

func serverAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <host>",
		Short: "Add a server to the pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, host := args[0], args[1]

			configPath, err := server.DefaultConfigPath()
			if err != nil {
				return fmt.Errorf("devbox server add: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not check server health: %v\n", err)
				sshExec = nil
			}
			if sshExec != nil {
				defer sshExec.Close()
			}

			pool, err := server.NewFilePool(configPath, sshExec)
			if err != nil {
				return fmt.Errorf("devbox server add: %w", err)
			}

			var opts []server.AddOption
			if u, _ := cmd.Flags().GetString("user"); u != "" {
				opts = append(opts, server.WithUser(u))
			}
			if p, _ := cmd.Flags().GetInt("port"); p != 0 {
				opts = append(opts, server.WithPort(p))
			}

			srv, err := pool.Add(name, host, opts...)
			if err != nil {
				return fmt.Errorf("devbox server add: %w", err)
			}

			fmt.Printf("Server %q (%s) added\n", srv.Name, srv.Host)

			status, err := pool.HealthCheck(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: health check failed: %v\n", err)
				return nil
			}

			if !status.SSH || !status.Docker || !status.Tailscale {
				fmt.Fprintf(os.Stderr, "Warning: some health checks failed — SSH=%v Docker=%v Tailscale=%v\n",
					status.SSH, status.Docker, status.Tailscale)
			}

			return nil
		},
	}
	cmd.Flags().String("user", "", "SSH user (default: current user)")
	cmd.Flags().Int("port", 0, "SSH port (default: 22)")
	return cmd
}

func serverRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a server from the pool",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			configPath, err := server.DefaultConfigPath()
			if err != nil {
				return fmt.Errorf("devbox server remove: %w", err)
			}

			pool, err := server.NewFilePool(configPath, nil)
			if err != nil {
				return fmt.Errorf("devbox server remove: %w", err)
			}

			if err := pool.Remove(name); err != nil {
				return fmt.Errorf("devbox server remove: %w", err)
			}

			fmt.Printf("Server %q removed\n", name)
			return nil
		},
	}
}

func serverListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List servers in the pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := server.DefaultConfigPath()
			if err != nil {
				return fmt.Errorf("devbox server list: %w", err)
			}

			check, _ := cmd.Flags().GetBool("check")

			var sshExec devboxssh.Executor
			if check {
				sshExec, err = devboxssh.New()
				if err != nil {
					return fmt.Errorf("devbox server list: %w", err)
				}
				defer sshExec.Close()
			}

			pool, err := server.NewFilePool(configPath, sshExec)
			if err != nil {
				return fmt.Errorf("devbox server list: %w", err)
			}

			servers, err := pool.List()
			if err != nil {
				return fmt.Errorf("devbox server list: %w", err)
			}

			if len(servers) == 0 {
				fmt.Println("No servers in pool. Add one with: devbox server add <name> <host>")
				return nil
			}

			var healthMap map[string]*server.HealthStatus
			if check {
				healthMap, err = pool.HealthCheckAll()
				if err != nil {
					return fmt.Errorf("devbox server list: health check failed: %w", err)
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if check {
				fmt.Fprintln(w, "NAME\tHOST\tUSER\tPORT\tSSH\tDOCKER\tTAILSCALE\tADDED")
			} else {
				fmt.Fprintln(w, "NAME\tHOST\tUSER\tPORT\tADDED")
			}

			for _, srv := range servers {
				user := srv.User
				if user == "" {
					user = "-"
				}
				port := "-"
				if srv.Port != 0 {
					port = fmt.Sprintf("%d", srv.Port)
				}
				added := timeAgo(srv.AddedAt)

				if check {
					status := healthMap[srv.Name]
					if status == nil {
						fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
							srv.Name, srv.Host, user, port, "err", "err", "err", added)
						continue
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						srv.Name, srv.Host, user, port,
						checkMark(status.SSH), checkMark(status.Docker), checkMark(status.Tailscale), added)
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						srv.Name, srv.Host, user, port, added)
				}
			}

			return w.Flush()
		},
	}
	cmd.Flags().Bool("check", false, "Run health checks against each server")
	return cmd
}

func checkMark(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

func snapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot <workspace> [name]",
		Short: "Create a snapshot of a workspace",
		Long:  "Save a point-in-time snapshot of a workspace's Docker volumes and config files.\nSnapshots are stored as compressed tar archives on the server.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsName := args[0]
			snapName := ""
			if len(args) > 1 {
				snapName = args[1]
			}

			serverFlag, _ := cmd.Flags().GetString("server")
			host, err := resolveServer(serverFlag)
			if err != nil {
				return fmt.Errorf("devbox snapshot: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox snapshot: %w", err)
			}
			defer sshExec.Close()

			mgr := snapshot.NewManager(sshExec)

			spin := ui.StartSpinner("Creating snapshot...")
			snap, err := mgr.Create(cmd.Context(), host, wsName, snapName)
			if err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox snapshot: %w", err)
			}
			ui.StopSpinner(spin, true)

			fmt.Printf("Snapshot %q created (%s)\n", snap.Name, formatBytes(snap.Size))
			return nil
		},
	}
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	cmd.AddCommand(snapshotListCmd())
	return cmd
}

func snapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <workspace>",
		Short: "List snapshots for a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsName := args[0]

			serverFlag, _ := cmd.Flags().GetString("server")
			host, err := resolveServer(serverFlag)
			if err != nil {
				return fmt.Errorf("devbox snapshot list: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox snapshot list: %w", err)
			}
			defer sshExec.Close()

			mgr := snapshot.NewManager(sshExec)
			snaps, err := mgr.List(cmd.Context(), host, wsName)
			if err != nil {
				return fmt.Errorf("devbox snapshot list: %w", err)
			}

			if len(snaps) == 0 {
				fmt.Println("No snapshots found")
				return nil
			}

			headers := []string{"NAME", "SIZE", "CREATED"}
			rows := make([][]string, 0, len(snaps))
			for _, s := range snaps {
				rows = append(rows, []string{
					s.Name,
					formatBytes(s.Size),
					s.CreatedAt.Format("2006-01-02 15:04:05"),
				})
			}
			ui.PrintTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	return cmd
}

func restoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <workspace> <snapshot>",
		Short: "Restore a workspace from a snapshot",
		Long:  "Restore a workspace's Docker volumes and config files from a previously saved snapshot.\nThe workspace containers will be stopped before restoring and restarted after.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			wsName, snapName := args[0], args[1]

			serverFlag, _ := cmd.Flags().GetString("server")
			host, err := resolveServer(serverFlag)
			if err != nil {
				return fmt.Errorf("devbox restore: %w", err)
			}

			sshExec, err := devboxssh.New()
			if err != nil {
				return fmt.Errorf("devbox restore: %w", err)
			}
			defer sshExec.Close()

			mgr := snapshot.NewManager(sshExec)

			spin := ui.StartSpinner("Restoring snapshot...")
			if err := mgr.Restore(cmd.Context(), host, wsName, snapName); err != nil {
				ui.StopSpinner(spin, false)
				return fmt.Errorf("devbox restore: %w", err)
			}
			ui.StopSpinner(spin, true)

			fmt.Printf("Workspace %q restored from snapshot %q\n", wsName, snapName)
			return nil
		},
	}
	cmd.Flags().String("server", "", "Target server (overrides devbox.yaml)")
	return cmd
}

// resolveServer determines the target server from flag or devbox.yaml config.
// When a flag is provided, it first tries to resolve it as a pool server name.
func resolveServer(serverFlag string) (string, error) {
	if serverFlag != "" {
		// Try to resolve as pool server name.
		configPath, _ := server.DefaultConfigPath()
		if pool, err := server.NewFilePool(configPath, nil); err == nil {
			if servers, err := pool.List(); err == nil {
				for _, srv := range servers {
					if srv.Name == serverFlag {
						return server.SSHHost(&srv), nil
					}
				}
			}
		}
		// Not in pool — use as raw hostname.
		return serverFlag, nil
	}
	cfg, err := config.LoadFromDir(".")
	if err == nil && cfg.Server != "" {
		return cfg.Server, nil
	}
	return "", fmt.Errorf("no server specified — use --server flag or create devbox.yaml")
}

// formatBytes returns a human-readable byte size string.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
