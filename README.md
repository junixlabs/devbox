# devbox

Turn any Linux machine into a ready-to-use dev environment in one command — no cloud, no DevOps required.

## Why devbox?

- **Codespaces** is managed but expensive ($40–80/dev/month)
- **Coder** is self-hosted but complex (needs Kubernetes)
- **DevPod** is simple but abandoned (last release June 2024)
- **devbox** is simple + self-hosted + free. You bring any Linux machine — old desktop, mini PC, cheap VPS — and devbox handles the rest.

## Features

- One-command workspace creation with `devbox up`
- Docker-based isolation — each workspace gets its own containers
- Tailscale networking — HTTPS, DNS, and access control out of the box
- Multi-workspace — run parallel workspaces for different branches or agents
- Works with any editor — Zed (recommended), VS Code, terminal SSH
- Import existing `docker-compose.yml` with `devbox init --from-compose`
- Health checks with `devbox doctor`

## Install

### Download binary

```bash
# Linux (amd64)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/

# Linux (arm64)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-arm64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/

# macOS (Apple Silicon)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-arm64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/

# macOS (Intel)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-amd64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/junixlabs/devbox.git
cd devbox
make build
sudo mv dist/devbox /usr/local/bin/
```

Requires Go 1.22+.

## Quick start

```bash
# 1. Verify your server is ready
devbox doctor --server dev1

# 2. Create a config in your project
devbox init

# 3. Start a workspace
devbox up

# 4. Connect
devbox ssh my-project
```

See [docs/QUICKSTART.md](docs/QUICKSTART.md) for a detailed step-by-step guide.

## Commands

| Command | Description |
|---------|-------------|
| `devbox init` | Create a `devbox.yaml` config interactively |
| `devbox up [project]` | Create and start a workspace |
| `devbox stop <workspace>` | Stop a running workspace |
| `devbox list` | List all workspaces |
| `devbox destroy <workspace>` | Permanently remove a workspace |
| `devbox ssh <workspace>` | SSH into a workspace |
| `devbox doctor [--server name]` | Check prerequisites and server health |

## Prerequisites

- A Linux server (Ubuntu 22.04+ recommended) accessible via SSH
- [Docker](https://docs.docker.com/engine/install/) installed on the server
- [Tailscale](https://tailscale.com/download) installed on both your machine and the server

## Documentation

- [Quick Start Guide](docs/QUICKSTART.md) — from install to `devbox up` in 15 minutes
- [Configuration Reference](docs/CONFIG.md) — all `devbox.yaml` fields explained
- [Troubleshooting](docs/TROUBLESHOOTING.md) — common issues and fixes

## License

[MPL-2.0](LICENSE)
