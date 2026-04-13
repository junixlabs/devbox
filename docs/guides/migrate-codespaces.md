# Migrate from GitHub Codespaces

This guide helps you move from GitHub Codespaces to devbox. The migration is straightforward — most concepts map directly.

## Why migrate?

| | GitHub Codespaces | devbox |
|---|---|---|
| **Cost** | $40-80/dev/month (compute + storage) | Free (your hardware) |
| **Hardware** | GitHub-managed VMs | Any Linux machine you own |
| **Customization** | Limited VM sizes | Full control over specs |
| **Networking** | GitHub network, port forwarding | Tailscale (HTTPS, DNS, ACLs) |
| **Offline** | No | Yes (local server) |
| **Data residency** | GitHub's cloud | Your infrastructure |
| **Editor lock-in** | VS Code / JetBrains | Any SSH-capable editor |

## Concept Mapping

| Codespaces | devbox | Notes |
|---|---|---|
| Codespace | Workspace | Same idea: isolated dev environment |
| `devcontainer.json` | `devbox.yaml` | devbox also reads devcontainer.json as fallback |
| Machine type (2/4/8 core) | `resources.cpus` / `resources.memory` | Set in devbox.yaml |
| Features | Services + Plugins | Docker images for services, plugins for extensions |
| Secrets | `env` in devbox.yaml | Or use your server's env management |
| Port forwarding | `ports` in devbox.yaml | Auto-exposed via Tailscale |
| Prebuilds | Snapshots | `devbox snapshot` to save state |
| `gh codespace` CLI | `devbox` CLI | Similar command structure |

## Step-by-Step Migration

### 1. Set up your server

If you don't have a server yet, any Linux machine works — a spare laptop, mini PC, or $5/month VPS:

```bash
# On the server
curl -fsSL https://get.docker.com | sh
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

### 2. Install devbox

On your development machine:

```bash
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox
chmod +x devbox && sudo mv devbox /usr/local/bin/
```

### 3. Convert your devcontainer.json

If your project has a `.devcontainer/devcontainer.json`, devbox can read it directly. But for full control, create a `devbox.yaml`:

**Before** (devcontainer.json):
```json
{
  "name": "my-app",
  "image": "mcr.microsoft.com/devcontainers/javascript-node:18",
  "features": {
    "ghcr.io/devcontainers/features/docker-in-docker:2": {}
  },
  "forwardPorts": [3000, 5432],
  "postCreateCommand": "npm install",
  "customizations": {
    "vscode": {
      "extensions": ["dbaeumer.vscode-eslint"]
    }
  }
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
  app: 3000
  postgres: 5432
env:
  NODE_ENV: development
  DATABASE_URL: postgres://postgres:postgres@postgres:5432/myapp
```

### 4. Convert Docker Compose (if applicable)

If your Codespace uses a Docker Compose file:

```bash
devbox init --from-compose .devcontainer/docker-compose.yml
```

### 5. Start your workspace

```bash
devbox up
```

### 6. Connect your editor

Instead of "Open in Codespaces", use SSH remoting:

=== "VS Code"

    Same Remote - SSH extension you may already use:
    ```
    Remote-SSH: Connect to Host... → dev1
    Open folder: /workspaces/my-app
    ```

=== "Zed"

    ```bash
    zed ssh://dev1/workspaces/my-app
    ```

=== "Terminal"

    ```bash
    devbox ssh my-app
    ```

## What you gain

- **No monthly bill** — use hardware you already own
- **No cold starts** — workspaces persist until you destroy them
- **Full root access** — install anything, no restrictions
- **Tailscale networking** — HTTPS, DNS, and access control without configuration
- **Multi-server** — scale across multiple machines as your team grows

## What's different

- **No browser IDE** — devbox is SSH-based, so you use your local editor
- **Server maintenance** — you're responsible for keeping Docker and Tailscale updated
- **No prebuilds** — use `devbox snapshot` for similar functionality
