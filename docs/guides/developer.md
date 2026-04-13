# Developer Guide

This guide walks through the daily workflow of using devbox as a developer — from setting up your first workspace to managing multiple branches and snapshots.

## Prerequisites

- devbox installed ([Quick Start](../getting-started/quickstart.md))
- A Linux server with Docker and Tailscale configured
- SSH access to the server

## Daily Workflow

### 1. Initialize your project

Navigate to your project directory and create a config:

```bash
cd ~/projects/my-app
devbox init
```

The interactive prompts will ask for:

- **Project name** — used as workspace and container identifier
- **Server** — which server to deploy to
- **Git repo** — repository URL to clone
- **Services** — databases, caches, etc.
- **Ports** — which ports to expose

!!! tip "Using templates"
    Skip manual setup with a built-in template:
    ```bash
    devbox template list              # See available templates
    devbox init --template go         # Go project with common defaults
    devbox init --template rails      # Rails with PostgreSQL + Redis
    devbox init --template nextjs     # Next.js with Node.js
    ```

### 2. Start your workspace

```bash
devbox up
```

devbox will:

1. Connect to the server via SSH
2. Clone your repository
3. Start Docker containers for your app and services
4. Expose ports via Tailscale
5. Print connection details

### 3. Connect with your editor

=== "Zed (recommended)"

    ```bash
    zed ssh://dev1/workspaces/my-app
    ```

=== "VS Code"

    1. Install the [Remote - SSH](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-ssh) extension
    2. Connect to `dev1` and open `/workspaces/my-app`

=== "Terminal"

    ```bash
    devbox ssh my-app
    ```

### 4. Code, test, iterate

Your workspace is a full Linux environment. Run your app, execute tests, install packages — everything works as expected.

```bash
# Inside the workspace
go test ./...
npm run dev
python manage.py runserver
```

Services defined in `devbox.yaml` are accessible by their image name:

```bash
# MySQL is available at mysql:3306
mysql -h mysql -u root -p

# Redis at redis:6379
redis-cli -h redis ping
```

### 5. Stop when done

```bash
devbox stop my-app
```

Data is preserved. Run `devbox up` again to resume exactly where you left off.

## Working with Multiple Branches

Create separate workspaces for different branches:

```bash
# Main branch workspace
devbox up

# Feature branch workspace (separate container)
devbox up --branch feature/auth

# List all workspaces
devbox list
```

Each branch gets its own isolated workspace with independent containers, services, and ports.

## Using Templates

### List available templates

```bash
devbox template list
```

Built-in templates: Go, Python, Node.js, Rails, Laravel, Next.js, Django, Rust.

### Create a custom template

```bash
devbox template create my-stack
```

This saves your current `devbox.yaml` as a reusable template.

### Community templates

```bash
devbox template search laravel    # Search the registry
devbox template pull author/name  # Download a template
devbox template push my-stack     # Share yours
```

## Snapshots

Save and restore workspace state:

```bash
# Save current state
devbox snapshot my-app

# List snapshots
devbox snapshot list

# Restore a previous state
devbox restore my-app-20260414-1200
```

Snapshots capture Docker volumes (databases, file state) as compressed tar archives.

## Monitoring

### Check workspace status

```bash
devbox list
```

### View resource usage

```bash
devbox stats my-app
```

Shows CPU, memory, disk, and network I/O for the workspace.

### Interactive dashboard

```bash
devbox tui
```

The TUI provides a real-time view of all workspaces with logs, metrics, and keyboard navigation.

## Port Access

Ports defined in `devbox.yaml` are exposed via Tailscale:

```yaml
ports:
  app: 8080
  mysql: 3306
```

Access them from your local machine at `http://dev1:8080` (via Tailscale hostname).

## Importing from Docker Compose

If your project already uses Docker Compose:

```bash
devbox init --from-compose docker-compose.yml
```

This extracts services, ports, and environment variables into `devbox.yaml`.

## Cleanup

```bash
# Stop workspace (preserves data)
devbox stop my-app

# Permanently remove workspace and all data
devbox destroy my-app
```

## Next Steps

- [Configuration Reference](../getting-started/config.md) — all `devbox.yaml` fields
- [Troubleshooting](../reference/troubleshooting.md) — common issues and fixes
- [Plugin API](../reference/plugin-api.md) — extend devbox with custom plugins
