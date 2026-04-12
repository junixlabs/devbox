# Changelog

All notable changes to devbox are documented in this file.

Format follows [Keep a Changelog](https://keepachangelog.com/).

## [1.0.0] - 2026-04-13

### Added
- **Plugin system** (`internal/plugin/`): Provider and Hook interfaces for extending devbox with custom backends (Docker, Podman, LXC) and lifecycle hooks
- **Built-in Docker provider** (`internal/plugin/docker/`): Default provider plugin shipping with devbox
- **External plugin support**: Discover, install, and remove third-party plugins from local directories with manifest validation
- **Community template registry** (`internal/registry/`): Search, pull, and push workspace templates to remote registries
- **CI/CD integration** (`internal/ci/`): GitHub Actions support for PR preview workspaces with commit status checks
- **GitHub Action** (`action.yml`): Reusable `devbox-preview` action for automated PR preview workflows
- **Preview workflow** (`.github/workflows/preview.yml`): Example workflow for PR preview lifecycle
- Plugin CLI commands: `devbox plugin list|install|remove`
- Template registry CLI commands: `devbox template search|pull|push`
- CI/CD CLI commands: `devbox ci preview-up|preview-down`
- Plugin API documentation (`docs/PLUGIN_API.md`)
- New built-in templates: Django, Rust

## [0.3.0] - 2026-04-12

### Added
- **TUI dashboard** (`internal/tui/`): Interactive terminal UI with Bubble Tea — workspace list, real-time logs viewer, keyboard navigation
- **Workspace templates** (`internal/template/`): Built-in and user-defined YAML templates (Go, Python, Node.js, Rails, Laravel, Next.js)
- **Snapshot & restore** (`internal/snapshot/`): Save and restore workspace state via Docker volume tar archives
- **Resource metrics** (`internal/metrics/`): CPU, memory, disk, and network I/O collection per workspace and server
- CLI commands: `devbox tui`, `devbox template list|create`, `devbox snapshot|restore`, `devbox stats`

## [0.2.0] - 2026-04-11

### Added
- **User isolation** (`internal/identity/`): Tailscale login-based identity resolution, user-scoped workspace naming
- **Port auto-allocation** (`internal/port/`): Registry with conflict detection and range-based allocation
- **Docker resource limits** (`internal/workspace/`): CPU and memory constraints per workspace
- **Server pool management** (`internal/server/`): Add/remove/list servers, health checks, least-loaded selection
- **Multi-server distribution**: Automatic workspace placement across server pools
- **Multi-user integration tests** (`internal/integration/`): E2E test suite with build tag isolation
- CLI commands: `devbox server add|remove|list`

## [0.1.0] - 2026-04-10

### Added
- **CLI skeleton** (`cmd/devbox/`): Cobra-based CLI with `up`, `stop`, `list`, `destroy`, `ssh` commands
- **Config parsing** (`internal/config/`): `devbox.yaml` with validation, services, ports, environment variables
- **SSH executor** (`internal/ssh/`): Remote command execution via native SSH
- **Docker Compose manager** (`internal/docker/`): Workspace lifecycle via `docker compose` over SSH
- **Tailscale integration** (`internal/tailscale/`): Serve/unserve workspaces, status queries
- **Workspace manager** (`internal/workspace/`): Orchestration layer wiring SSH, Docker, and Tailscale
- **Devcontainer support**: `.devcontainer/devcontainer.json` fallback config loading
- **`devbox init`**: Interactive project setup with compose file conversion
- **`devbox doctor`**: Health checks for Docker, Tailscale, SSH connectivity
- **Structured error handling** (`internal/errors/`): Typed errors with suggestions
- **CLI UX polish** (`internal/ui/`): Spinners, colored output, `--no-color` flag
- **Cross-compilation**: Makefile and GitHub Actions release workflow for linux/darwin amd64/arm64
