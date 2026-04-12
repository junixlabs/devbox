# Contributing to devbox

Thanks for your interest in contributing to devbox! This guide covers everything you need to get started.

## Development Setup

**Requirements:** Go 1.24+

```bash
git clone https://github.com/junixlabs/devbox.git
cd devbox
go build -o devbox ./cmd/devbox/
go test ./...
```

## Project Structure

```
cmd/devbox/         CLI entry point — all commands in main.go
internal/
  config/           devbox.yaml parsing and validation
  workspace/        Workspace model, Manager interface, resource limits
  ssh/              SSH remote command executor
  docker/           Docker Compose management via SSH
  tailscale/        Tailscale serve/unserve/status
  identity/         User identity resolution (Tailscale login)
  server/           Server pool management
  port/             Port auto-allocation registry
  metrics/          Resource metrics collector
  snapshot/         Snapshot & restore workspace state
  template/         Workspace templates (built-in + user-defined)
  tui/              Interactive TUI dashboard (Bubble Tea)
  plugin/           Plugin system (Provider/Hook interfaces)
  registry/         Community template registry
  ci/               CI/CD integration (GitHub Actions)
  errors/           Typed errors with suggestions
  ui/               CLI output helpers (spinners, colors)
  testutil/         Shared test helpers
  integration/      Multi-user E2E tests
```

## Adding a New Command

1. Create a function `yourCmd() *cobra.Command` in `cmd/devbox/main.go`
2. Wire it: `rootCmd.AddCommand(yourCmd())`
3. Follow existing patterns — use `RunE` (not `Run`), wrap errors with `fmt.Errorf("devbox your-cmd: %w", err)`

## Adding a New Package

1. Create `internal/yourpkg/yourpkg.go`
2. Define an interface for the contract
3. Implement the interface
4. Import from CLI in `main.go`

## Adding a Config Field

1. Add to `DevboxConfig` struct in `internal/config/config.go` with a `yaml` tag
2. Add validation in `Load()` if needed
3. Update `devbox.yaml.example`

## Writing Plugins

See [docs/PLUGIN_API.md](docs/PLUGIN_API.md) for the full plugin development guide.

Plugins implement `plugin.Provider` (custom workspace backends) or `plugin.Hook` (lifecycle callbacks). Create a directory with a `plugin.yaml` manifest and a Go binary, then install with `devbox plugin install <path>`.

## Contributing Templates

Templates are YAML files in `internal/template/builtin/`. To add a new built-in template:

1. Create `internal/template/builtin/yourtemplate.yaml`
2. Follow the schema from existing templates (name, image, services, ports)
3. Register it in the built-in embed

To share templates via the community registry, use `devbox template push`.

## Code Style

- **Error wrapping**: Always use `fmt.Errorf("context: %w", err)`
- **Naming**: Go conventions — CamelCase exports, lowercase packages
- **Interfaces**: Define contracts before implementations
- **Tests**: Every package should have `_test.go` files
- **No runtime dependencies**: Single binary, cross-compile friendly

## Pull Request Process

1. Fork and create a feature branch
2. Make your changes with tests
3. Run `go test ./...` and `go vet ./...`
4. Submit a PR with a clear description of what and why

## License

By contributing, you agree that your contributions will be licensed under the project's license.
