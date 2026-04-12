# devbox Product Vision

> "Turn any Linux machine into a ready-to-use dev environment in one command — no cloud, no DevOps."

## Positioning
Simple + Self-hosted + Cheap (vs Codespaces=expensive, Coder=complex, DevPod=dead)

## Killer Feature
Multi-agent collaboration (Agent Farm) — isolated workspaces per AI agent on shared servers.

## Target Users
- Small dev teams (2-20) without DevOps
- Individual developers with limited laptop storage (256GB SSD)

## Architecture
```
Editor (Zed/VS Code) → devbox CLI → Docker Compose → Tailscale → Linux Hardware
```

## Tech Stack
- Go (Cobra CLI, single binary, cross-compile)
- Config: devbox.yaml (yaml.v3), devcontainer.json (Phase 1)
- Docker Compose v2 on remote server
- Tailscale mesh VPN for networking
- License: MPL-2.0

## Roadmap

| Phase | Goal | Success Metric | Release |
|-------|------|---------------|---------|
| P0: Dogfood | Team uses daily | `devbox up` < 5 min, full lifecycle | — |
| P1: MVP | Outsiders can use | 3+ external users, 15 min onboard | v0.1.0 |
| P2: Multi-user | Shared servers | 3+ devs, 1 server, no conflicts | v0.2.0 |
| P3: TUI/Web UI | Dashboard | TUI usable, 3+ stack templates | v0.3.0 |
| P4: Community | Open source adoption | Plugins, community templates | v1.0.0 |

## Design Principles
1. Simple by default, powerful when needed
2. Convention over configuration
3. Fail fast, fix fast — clear errors with fix suggestions
4. Respect existing tools (Docker, Tailscale, SSH, devcontainer spec)
5. Offline-first — no SaaS dependency, no telemetry
