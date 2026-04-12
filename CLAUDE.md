# devbox
Go CLI tool that turns any Linux machine into a ready-to-use dev environment in one command via Docker + Tailscale.

## Architecture
- `cmd/devbox/` — CLI entry point, all cobra commands defined in main.go
- `internal/config/` — devbox.yaml parsing and validation
- `internal/workspace/` — Workspace model and Manager interface (lifecycle: create/start/stop/destroy/list/ssh)
- `internal/tailscale/` — Tailscale Manager interface (serve/unserve/status)
- `.claude/specs/` — Product vision and design documents

## Key Patterns
- **Cobra CLI**: All commands defined as funcs returning `*cobra.Command`, wired in `main()`
- **Interface-driven**: `workspace.Manager` and `tailscale.Manager` define contracts; implementations are TBD (Phase 0 skeleton)
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
./devbox up [project]               # Start workspace
./devbox stop|list|destroy|ssh      # Other commands (stubs in Phase 0)
```
