# Migrate from DevPod

This guide helps you move from DevPod to devbox. Both tools solve the same problem — self-hosted dev environments — but devbox is actively maintained and adds features DevPod lacks.

## Why migrate?

DevPod's last release was June 2024. devbox is actively developed with regular releases.

| | DevPod | devbox |
|---|---|---|
| **Status** | Abandoned (last release June 2024) | Active development (v1.0.0) |
| **Networking** | Manual port forwarding | Tailscale (automatic HTTPS, DNS) |
| **Multi-server** | Single provider | Server pools with auto-selection |
| **Monitoring** | None | Built-in metrics + TUI dashboard |
| **Snapshots** | None | Built-in snapshot/restore |
| **Plugins** | Provider-based | Provider + Hook system |
| **CI/CD** | None | GitHub Actions PR previews |
| **Templates** | devcontainer only | devbox templates + community registry |

## Concept Mapping

| DevPod | devbox | Notes |
|---|---|---|
| Workspace | Workspace | Same concept |
| Provider (Docker, SSH, Cloud) | Server pool | devbox uses SSH + Docker on any server |
| `devcontainer.json` | `devbox.yaml` | devbox also reads devcontainer.json |
| `devpod up` | `devbox up` | Almost identical usage |
| `devpod ssh` | `devbox ssh` | Same concept |
| `devpod delete` | `devbox destroy` | Same concept |
| Desktop app | `devbox tui` | Terminal-based dashboard |

## Step-by-Step Migration

### 1. Export your current setup

Note your DevPod workspace configurations — you'll recreate them as `devbox.yaml` files.

```bash
# Check your current DevPod workspaces
devpod list
```

### 2. Install devbox

```bash
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/
```

### 3. Set up Tailscale (if not already)

The main difference from DevPod: devbox uses Tailscale for networking instead of manual SSH tunnels.

On both your machine and server:

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

### 4. Convert your workspace config

**Before** (DevPod with devcontainer.json):
```json
{
  "name": "my-app",
  "image": "mcr.microsoft.com/devcontainers/go:1.22",
  "forwardPorts": [8080, 5432],
  "postCreateCommand": "go mod download"
}
```

**After** (devbox.yaml):
```yaml
name: my-app
server: dev1
repo: git@github.com:your-org/my-app.git
services:
  - postgres:15
ports:
  app: 8080
  postgres: 5432
env:
  GOPATH: /go
```

Or use a template:

```bash
devbox init --template go
```

### 5. Add your server

```bash
devbox server add dev1 100.64.0.1
devbox doctor --server dev1
```

### 6. Create workspace

```bash
cd ~/projects/my-app
devbox up
```

### 7. Connect

```bash
devbox ssh my-app
# or
zed ssh://dev1/workspaces/my-app
```

## Command Comparison

| DevPod | devbox | Description |
|---|---|---|
| `devpod up` | `devbox up` | Create/start workspace |
| `devpod ssh` | `devbox ssh` | Connect to workspace |
| `devpod stop` | `devbox stop` | Stop workspace |
| `devpod delete` | `devbox destroy` | Remove workspace |
| `devpod list` | `devbox list` | List workspaces |
| `devpod provider add` | `devbox server add` | Add infrastructure |
| — | `devbox doctor` | Health checks |
| — | `devbox snapshot` | Save workspace state |
| — | `devbox stats` | Resource metrics |
| — | `devbox tui` | Interactive dashboard |
| — | `devbox template search` | Community templates |
| — | `devbox ci preview-up` | PR preview workspaces |

## What you gain

- **Active maintenance** — regular updates and bug fixes
- **Tailscale networking** — automatic HTTPS, DNS, no manual port forwarding
- **Server pools** — distribute workspaces across multiple servers
- **Snapshots** — save and restore workspace state
- **Monitoring** — real-time metrics and TUI dashboard
- **CI/CD integration** — PR preview workspaces out of the box
- **Plugin system** — extend with custom providers and lifecycle hooks
- **Community templates** — share and discover workspace configurations
