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
| `devbox mcp serve` | Start MCP server for AI agent integration |

## MCP Integration

devbox exposes all operations via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) — AI agents can create workspaces, execute commands, and manage environments programmatically.

```json
{
  "mcpServers": {
    "devbox": {
      "command": "devbox",
      "args": ["mcp", "serve"]
    }
  }
}
```

14 tools available: workspace lifecycle, server management, snapshots, templates, and multi-agent sessions with workspace isolation.

- [MCP Server Reference](docs/MCP.md) — all tools, parameters, responses, and error codes
- [Agent Farm Setup Guide](docs/AGENT_FARM.md) — multi-agent configuration, resource planning, isolation
- [Claude Desktop Example](examples/claude-desktop-config.json) — drop-in MCP configuration
- [Agent Script Example](examples/agent-script.py) — Python script using devbox MCP

## Prerequisites

- A Linux server (Ubuntu 22.04+ recommended) accessible via SSH
- [Docker](https://docs.docker.com/engine/install/) installed on the server
- [Tailscale](https://tailscale.com/download) installed on both your machine and the server

## Documentation

**[https://junixlabs.github.io/devbox](https://junixlabs.github.io/devbox)**

- [Quick Start Guide](https://junixlabs.github.io/devbox/getting-started/quickstart/) — from install to `devbox up` in 15 minutes
- [Configuration Reference](https://junixlabs.github.io/devbox/getting-started/config/) — all `devbox.yaml` fields explained
- [Developer Guide](https://junixlabs.github.io/devbox/guides/developer/) — daily workflow, templates, snapshots
- [Admin Guide](https://junixlabs.github.io/devbox/guides/admin/) — server pools, user isolation, monitoring
- [AI Agent Builder Guide](https://junixlabs.github.io/devbox/guides/agent-builder/) — workspace-per-agent, MCP integration
- [MCP Server Reference](https://junixlabs.github.io/devbox/reference/mcp-server/) — AI agent integration via Model Context Protocol
- [Plugin API](https://junixlabs.github.io/devbox/reference/plugin-api/) — extend devbox with custom providers and hooks
- [Troubleshooting](https://junixlabs.github.io/devbox/reference/troubleshooting/) — common issues and fixes
- [FAQ](https://junixlabs.github.io/devbox/reference/faq/) — frequently asked questions

## License

[MPL-2.0](LICENSE)
