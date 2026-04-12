# devbox
Go CLI tool that turns any Linux machine into a ready-to-use dev environment in one command via Docker + Tailscale.

## Architecture
- `cmd/devbox/` — CLI entry point, all cobra commands defined in main.go
- `internal/config/` — devbox.yaml parsing, validation, Resources, PortRangeConfig
- `internal/identity/` — User identity resolution (Tailscale login → username, DEVBOX_USER fallback)
- `internal/port/` — Port auto-allocation registry with conflict detection
- `internal/server/` — Server pool management (add/remove/list/health), least-loaded selector
- `internal/workspace/` — Workspace model, Manager interface, resource limits, user-scoped naming
- `internal/tailscale/` — Tailscale Manager interface (serve/unserve/status)
- `internal/tui/` — Interactive TUI dashboard with Bubble Tea (workspace list, logs viewer, keyboard navigation)
- `internal/template/` — Workspace templates system (built-in + user-defined, YAML-based)
- `internal/snapshot/` — Snapshot & restore workspace state (Docker volumes, compressed tar archives)
- `internal/metrics/` — Resource metrics collector (CPU, memory, disk, network I/O per workspace/server)
- `internal/integration/` — Multi-user integration tests (build tag: integration)
- `internal/testutil/` — Shared test helpers for SSH, Docker, assertions
- `.claude/specs/` — Product vision and design documents

## Key Patterns
- **Cobra CLI**: All commands defined as funcs returning `*cobra.Command`, wired in `main()`
- **Interface-driven**: `workspace.Manager`, `tailscale.Manager`, `identity.Resolver`, `port.Registry`, `server.Pool`, `metrics.Collector`, `snapshot.Manager` define contracts
- **Config**: Per-project `devbox.yaml` parsed into `DevboxConfig` struct with yaml tags; `name` and `server` are required fields
- **Error wrapping**: `fmt.Errorf("context: %w", err)` for all error propagation
- **Single binary**: No runtime dependencies, cross-compile with `GOOS`/`GOARCH`

## Recipes
- **Add a command**: Create `{name}Cmd() *cobra.Command` in main.go → add `rootCmd.AddCommand({name}Cmd())`
- **Add an internal package**: Create `internal/{name}/{name}.go` → define interface → implement → import from CLI
- **Add a config field**: Add to `DevboxConfig` struct with yaml tag → validate in `Load()` → update `devbox.yaml.example`

## Commands
```
go build -o devbox ./cmd/devbox/   # Build binary
go test ./...                       # Run all tests
go vet ./...                        # Lint
./devbox up [project]               # Start workspace (auto-selects server, resolves user)
./devbox stop|list|destroy|ssh      # Workspace lifecycle commands
./devbox server add|remove|list     # Server pool management
./devbox tui                        # Interactive TUI dashboard
./devbox template list|create       # Workspace templates
./devbox snapshot|restore           # Snapshot & restore workspace state
./devbox stats [workspace]          # Resource metrics
```
