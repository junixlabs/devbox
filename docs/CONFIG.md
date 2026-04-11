# Configuration Reference

devbox uses a per-project `devbox.yaml` file to define workspace settings. Place it in your project root.

## Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | **Yes** | — | Workspace name. Used as container prefix and Tailscale hostname. |
| `server` | string | **Yes** | — | Target server. Must be reachable via SSH and Tailscale. |
| `repo` | string | No | — | Git repository URL to clone into the workspace. |
| `branch` | string | No | `main` | Git branch to checkout. Can be overridden with `devbox up --branch`. |
| `services` | list of strings | No | — | Docker images to run alongside the app (e.g. `mysql:8.0`). |
| `ports` | map (string → int) | No | — | Named port mappings exposed via Tailscale. |
| `env` | map (string → string) | No | — | Environment variables injected into the workspace. |

## Validation

- `name` and `server` are **required**. devbox will exit with an error and a hint if either is missing.
- All other fields are optional.

## Full example

```yaml
# Workspace name (used as container prefix and Tailscale hostname)
name: my-project

# Target server (must be reachable via SSH and Tailscale)
server: dev1

# Git repository to clone into the workspace
repo: git@github.com:your-org/your-repo.git

# Branch to checkout (optional, defaults to main)
branch: feature/new-ui

# Services to run alongside the app (Docker images)
services:
  - mysql:8.0
  - redis:7-alpine

# Port mappings exposed to the developer machine via Tailscale
ports:
  app: 8080
  mysql: 3306
  redis: 6379

# Environment variables injected into the workspace
env:
  APP_ENV: local
  APP_DEBUG: "true"
  DB_CONNECTION: mysql
  DB_HOST: mysql
  DB_PORT: "3306"
  DB_DATABASE: my-project
  DB_USERNAME: root
  DB_PASSWORD: secret
  CACHE_DRIVER: redis
  REDIS_HOST: redis
```

## Minimal example

Only `name` and `server` are required:

```yaml
name: my-project
server: dev1
```

## CLI overrides

Some fields can be overridden via CLI flags:

```bash
# Override server
devbox up --server dev2

# Override branch
devbox up --branch feature/auth
```

CLI flags take precedence over `devbox.yaml` values.

## Creating a config

Use `devbox init` to create a config interactively:

```bash
devbox init
```

Or convert from an existing Docker Compose file:

```bash
devbox init --from-compose docker-compose.yml
```

See [devbox.yaml.example](../devbox.yaml.example) for a complete annotated example.
