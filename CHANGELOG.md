# Changelog

All notable changes to devbox are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-13

### Added
- Full v1.0.0 release: community-ready with comprehensive documentation
- CHANGELOG.md covering all phases
- CONTRIBUTING.md with contributor guide

### Changed
- Version bump from 0.3.0 to 1.0.0
- Deduplicated `firstService` helper (consolidated into `workspace.FirstService`)
- Deduplicated `formatBytes` (consolidated into `workspace.FormatBytes`)

## [0.3.0] - 2026-04-12

### Added
- **TUI Dashboard** (`devbox tui`) — interactive Bubble Tea dashboard with workspace list, log viewer, keyboard navigation, filtering, and real-time refresh
- **Workspace Templates** (`devbox template list|create`) — built-in templates (Laravel, Next.js, Go, Rails, Django) with custom template support via `~/.config/devbox/templates/`
- **Snapshot & Restore** (`devbox snapshot|restore`) — point-in-time workspace snapshots as compressed tar archives on the server, with list and restore commands
- **Resource Metrics** (`devbox stats`) — per-workspace CPU, memory, disk, and network I/O metrics via Docker stats; server-level summary with CPU cores, RAM, and disk usage

### Changed
- Root command (`devbox` with no args) now launches TUI dashboard
- `devbox up --template` flag creates workspaces from templates
- `devbox list` now shows live CPU% and MEM% columns from Docker stats
- Deduplicated `formatBytes` across packages, extracted `metricsSep` constant

## [0.2.0] - 2026-04-11

### Added
- **Server Pool Management** (`devbox server add|remove|list`) — manage multiple servers with YAML-backed config at `~/.config/devbox/servers.yaml`
- **Multi-Server Distribution** — auto-select least-loaded server via parallel resource probing (CPU + memory scoring)
- **Docker Resource Limits** — per-workspace CPU and memory limits via `resources` config field, merged with server defaults from `~/.devbox/config.yaml`
- **Workspace Naming & User Isolation** — workspaces named `{user}-{project}-{branch}`, user identity resolved from Tailscale login or `DEVBOX_USER` env var
- **Port Auto-Allocation** — automatic port assignment from configurable range with conflict detection across workspaces
- **Multi-User Integration Tests** — end-to-end test suite with shared test helpers for SSH, Docker, and assertions
- Server health checks (`devbox server list --check`) — SSH, Docker, and Tailscale connectivity verification
- `devbox list --all` and `--server` filtering flags
- Low-resource warnings when server CPU or memory exceeds 85%

### Changed
- `devbox up` now supports 3-tier server resolution: `--server` flag, `devbox.yaml`, or auto-select from pool
- `devbox list` shows per-user filtering by default (use `--all` for all users)

## [0.1.0] - 2026-04-10

### Added
- **CLI Skeleton** — Cobra-based CLI with `up`, `stop`, `list`, `destroy`, `ssh`, `doctor`, `init` commands
- **Config Parsing** — `devbox.yaml` with `name`, `server`, `repo`, `branch`, `services`, `ports`, `env` fields; validation with actionable error messages
- **SSH Executor** — ControlMaster-backed SSH with connection pooling, SCP file transfer, and streaming output
- **Tailscale Integration** — `tailscale serve` port exposure, status retrieval, workspace URL generation
- **Docker Compose Manager** — remote Docker Compose orchestration via SSH (deploy, up, down, ps, logs, destroy)
- **Workspace Manager** — full lifecycle management with local JSON state persistence (`~/.devbox/state.json`)
- **Devcontainer Support** — automatic fallback from `devbox.yaml` to `.devcontainer/devcontainer.json` with JSONC parsing
- **`devbox init`** — interactive config generator with `--from-compose` conversion from existing Docker Compose files
- **`devbox doctor`** — health checks for Git, SSH, Docker, Tailscale, and disk space
- **Custom Error Types** — `ConfigError`, `ConnectionError`, `DockerError` with `Suggestible` interface for actionable hints
- **CLI UX** — colored output, spinners, tabular display, `--verbose` debug logging, `--no-color` flag
- Unit tests for config, SSH, workspace, docker, and tailscale packages
- Integration test suite for E2E workspace lifecycle
