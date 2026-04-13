# Quick Start Guide

Get from zero to a running workspace in 15 minutes.

## Prerequisites

Before you start, make sure you have:

- [x] A Linux server (Ubuntu 22.04+) with SSH access
- [x] [Docker](https://docs.docker.com/engine/install/) installed on the server
- [x] [Tailscale](https://tailscale.com/download) installed on **both** your local machine and the server
- [x] Both machines connected to the same Tailscale network

## Step 1: Install devbox

Download the binary for your platform:

=== "Linux (amd64)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox
    chmod +x devbox && sudo mv devbox /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-arm64 -o devbox
    chmod +x devbox && sudo mv devbox /usr/local/bin/
    ```

=== "Build from source"

    ```bash
    git clone https://github.com/junixlabs/devbox.git
    cd devbox && make build
    sudo mv dist/devbox /usr/local/bin/
    ```

    Requires Go 1.22+.

Verify the installation:

```bash
devbox --version
```

## Step 2: Configure SSH access

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

!!! tip
    Use the Tailscale IP (`100.x.x.x`) as the hostname so the connection works from anywhere on your tailnet, not just your local network.

## Step 3: Check server health

Run the built-in health checker:

```bash
devbox doctor --server dev1
```

This verifies SSH connectivity, Docker, Tailscale, git, and disk space on the server. Fix any issues it reports before continuing.

## Step 4: Create a config

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

See [Configuration Reference](config.md) for all available fields.

## Step 5: Start a workspace

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

=== "SSH"

    ```bash
    devbox ssh my-project
    ```

=== "Zed (recommended)"

    ```bash
    zed ssh://dev1/workspaces/my-project
    ```

=== "VS Code"

    Use the [Remote - SSH](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh) extension to connect to your server.

## Step 7: Verify

Check that your workspace is running:

```bash
devbox list
```

You should see your workspace with status `running`, the server name, and exposed ports.

## What's next?

| Command | Description |
|---------|-------------|
| `devbox stop my-project` | Stop the workspace (data preserved) |
| `devbox up` | Restart an existing workspace |
| `devbox destroy my-project` | Permanently remove it |
| `devbox up --branch feature/xyz` | Workspace for a different branch |
| `devbox snapshot my-project` | Save workspace state |
| `devbox tui` | Interactive dashboard |

## Troubleshooting

If something goes wrong, see [Troubleshooting](../reference/troubleshooting.md) for common issues and fixes.
