# Contributing to devbox

Thanks for your interest in contributing to devbox! This guide covers everything you need to get started.

## Prerequisites

- **Go 1.25+** — [install](https://go.dev/dl/)
- **SSH client** — OpenSSH (`ssh`, `scp`)
- **Docker + Docker Compose** — on the target server
- **Tailscale** — on both client and server (optional, for port exposure)

## Development Setup

```bash
# Clone
git clone https://github.com/junixlabs/devbox.git
cd devbox

# Build
go build -o devbox ./cmd/devbox/

# Run tests
go test ./...

# Lint
go vet ./...
```

## Project Structure

```
cmd/devbox/          CLI entry point (all cobra commands in main.go)
internal/
  config/            devbox.yaml parsing, validation, devcontainer support
  docker/            Docker Compose generation and remote management via SSH
  doctor/            Health check runner (Git, SSH, Docker, Tailscale, disk)
  errors/            Custom error types with actionable suggestions
  identity/          User identity resolution (Tailscale login, DEVBOX_USER)
  integration/       Multi-user integration tests (build tag: integration)
  metrics/           Resource metrics collector (CPU, memory, disk, network)
  port/              Port auto-allocation registry with conflict detection
  server/            Server pool management, least-loaded selector
  snapshot/          Snapshot & restore workspace state (Docker volumes)
  ssh/               SSH executor with ControlMaster connection pooling
  tailscale/         Tailscale serve/unserve/status interface
  template/          Workspace templates (built-in + user-defined YAML)
  testutil/          Shared test helpers for integration tests
  tui/               Interactive Bubble Tea dashboard
  ui/                CLI output helpers (spinners, tables, colors)
  workspace/         Workspace model, Manager interface, state persistence
```

## How to Add a Command

1. Create a function `{name}Cmd() *cobra.Command` in `cmd/devbox/main.go`
2. Wire it in `main()`: `rootCmd.AddCommand({name}Cmd())`
3. Follow the existing pattern — see `statsCmd()` or `snapshotCmd()` for examples

## How to Add an Internal Package

1. Create `internal/{name}/{name}.go`
2. Define an interface for the contract (e.g., `Manager`, `Collector`)
3. Implement the interface
4. Add `_test.go` with unit tests
5. Import from the CLI layer (`cmd/devbox/main.go`)

## How to Add a Config Field

1. Add the field to `DevboxConfig` struct in `internal/config/config.go` with a `yaml` tag
2. Add validation in `Load()` if needed
3. Update `devbox.yaml.example` (if applicable)

## How to Add a Built-in Template

1. Create `internal/template/builtin/{name}.yaml`
2. Follow the existing format (see `laravel.yaml`, `nextjs.yaml`)
3. The template is automatically embedded via `//go:embed`

## Code Conventions

- **Error wrapping**: Always use `fmt.Errorf("context: %w", err)`
- **Interface-driven**: Define interfaces for contracts, implement concretely
- **Input validation**: Validate names/inputs before shell interpolation (use `validName` regex)
- **Single binary**: No runtime dependencies beyond SSH, Docker, and Tailscale on the server

## Testing

```bash
# Unit tests (fast, no server needed)
go test ./...

# Integration tests (requires SSH access to test server)
go test -tags integration ./internal/integration/ -v

# Override test server
DEVBOX_TEST_SERVER=my-server go test -tags integration ./internal/integration/ -v

# Skip integration tests
DEVBOX_TEST_SERVER=skip go test -tags integration ./internal/integration/ -v
```

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with tests
3. Run `go test ./...` and `go vet ./...` — both must pass
4. Open a PR with a clear description of what and why
5. Address review feedback

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.
