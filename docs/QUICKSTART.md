# Quick Start Guide

Get from zero to a running workspace in 15 minutes.

## Prerequisites

Before you start, make sure you have:

- [ ] A Linux server (Ubuntu 22.04+) with SSH access
- [ ] [Docker](https://docs.docker.com/engine/install/) installed on the server
- [ ] [Tailscale](https://tailscale.com/download) installed on **both** your local machine and the server
- [ ] Both machines connected to the same Tailscale network

## Step 1: Install devbox (~1 min)

Download the binary for your platform:

```bash
# Linux (amd64)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox

# macOS (Apple Silicon)
curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-arm64 -o devbox

chmod +x devbox && sudo mv devbox /usr/local/bin/
```

Verify the installation:

```bash
devbox --version
```

### Build from source (alternative)

```bash
git clone https://github.com/junixlabs/devbox.git
cd devbox
make build
sudo mv dist/devbox /usr/local/bin/
```

Requires Go 1.22+.

## Step 2: Configure SSH access (~2 min)

Make sure you can SSH into your server by name:

```bash
ssh dev1 "hostname"
```

If this doesn't work, add an entry to `~/.ssh/config`:

```
Host dev1
    HostName 100.x.x.x    # Tailscale IP of your server
    User your-username
```

> **Tip:** Use the Tailscale IP (`100.x.x.x`) as the hostname so the connection works from anywhere on your tailnet, not just your local network.

## Step 3: Check server health (~1 min)

Run the built-in health checker:

```bash
devbox doctor --server dev1
```

This verifies SSH connectivity, Docker, Tailscale, git, and disk space on the server. Fix any issues it reports before continuing.

## Step 4: Create a config (~2 min)

In your project directory, run:

```bash
devbox init
```

This creates a `devbox.yaml` with interactive prompts for project name, server, git repo, services, and ports.

### Already have a docker-compose.yml?

Convert it directly:

```bash
devbox init --from-compose docker-compose.yml
```

This extracts services and port mappings from your existing Compose file.

### Manual config

Alternatively, create `devbox.yaml` by hand:

```yaml
name: my-project
server: dev1
repo: git@github.com:your-org/your-repo.git
services:
  - mysql:8.0
  - redis:7-alpine
ports:
  app: 8080
  mysql: 3306
env:
  APP_ENV: local
  DB_HOST: mysql
```

See [CONFIG.md](CONFIG.md) for all available fields.

## Step 5: Start a workspace (~5 min)

```bash
devbox up
```

This will:
1. SSH into the server
2. Clone your repo
3. Start Docker containers for your app and services
4. Expose ports via Tailscale

You'll see a summary with the workspace URL and connection details when it's done.

## Step 6: Connect

### SSH

```bash
devbox ssh my-project
```

### Zed (recommended)

```bash
zed ssh://dev1/workspaces/my-project
```

### VS Code

Use the [Remote - SSH](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh) extension to connect to your server.

## Step 7: Verify

Check that your workspace is running:

```bash
devbox list
```

You should see your workspace with status `running`, the server name, and exposed ports.

## What's next?

- Run `devbox stop my-project` to stop the workspace (data is preserved)
- Run `devbox up` again to restart it
- Run `devbox destroy my-project` to permanently remove it
- Create multiple workspaces for different branches with `devbox up --branch feature/xyz`

## Troubleshooting

If something goes wrong, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues and fixes.
