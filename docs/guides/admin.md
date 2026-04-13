# Admin Guide

This guide covers setting up and managing devbox infrastructure — server pools, user isolation, resource limits, and monitoring.

## Prerequisites

- One or more Linux servers (Ubuntu 22.04+ recommended)
- Root or sudo access on each server
- Tailscale account with the servers on your tailnet

## Server Setup

### 1. Install Docker

On each server:

```bash
curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo systemctl start docker
```

### 2. Install Tailscale

```bash
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Verify the server appears in your [Tailscale admin console](https://login.tailscale.com/admin/machines).

### 3. Configure SSH

Ensure users can SSH into the server via Tailscale IP:

```bash
# On the server, verify SSH is running
sudo systemctl status sshd
```

Users will configure their local `~/.ssh/config` to connect:

```
Host dev1
    HostName 100.x.x.x    # Tailscale IP
    User devuser
```

### 4. Add users to the docker group

Each user who will create workspaces needs docker access:

```bash
sudo usermod -aG docker username
```

### 5. Verify with devbox doctor

From a client machine:

```bash
devbox doctor --server dev1
```

This checks: SSH connectivity, Docker availability, Tailscale status, git, and disk space.

## Server Pool Management

devbox supports multiple servers for distributing workspaces.

### Add servers

```bash
devbox server add dev1 100.64.0.1
devbox server add dev2 100.64.0.2
devbox server add gpu-box 100.64.0.10
```

### List servers

```bash
devbox server list
```

Shows each server with its health status, number of running workspaces, and resource usage.

### Remove servers

```bash
devbox server remove dev2
```

!!! warning
    Removing a server does not destroy workspaces running on it. Migrate or destroy workspaces first.

### Automatic server selection

When users run `devbox up` without specifying a server, devbox uses a **least-loaded selection** algorithm:

1. Queries all servers in the pool
2. Checks resource availability (CPU, memory, disk)
3. Selects the server with the most headroom

Users can override with `devbox up --server dev2`.

## User Isolation

devbox automatically isolates workspaces per user.

### How identity works

1. devbox resolves the user's identity from their **Tailscale login** (e.g., `alice@example.com` → `alice`)
2. Fallback: the `DEVBOX_USER` environment variable
3. Workspace names are scoped: `{user}-{project}-{branch}`

This means two users can run `devbox up` in the same project without conflicts — each gets their own workspace.

### Resource limits

Configure per-workspace resource limits to prevent any single workspace from consuming all server resources:

```yaml
# In devbox.yaml
resources:
  cpus: 2.0      # Max CPU cores
  memory: 4g     # Max memory
```

These map directly to Docker's `--cpus` and `--memory` flags.

## Monitoring

### Per-workspace metrics

```bash
devbox stats my-project
```

Shows:

- **CPU** — usage percentage and allocated cores
- **Memory** — used vs. allocated
- **Disk** — volume size
- **Network I/O** — bytes in/out

### All workspaces

```bash
devbox list
```

### Interactive dashboard

```bash
devbox tui
```

The TUI dashboard shows all workspaces across all servers with live status, logs, and resource indicators.

## CI/CD Preview Workspaces

devbox integrates with GitHub Actions to create temporary workspaces for pull requests.

### Setup

1. Add the preview workflow to your repository:

    ```yaml
    # .github/workflows/preview.yml
    name: PR Preview
    on:
      pull_request:
        types: [opened, synchronize, reopened, closed]

    permissions:
      pull-requests: write
      statuses: write

    jobs:
      preview-up:
        if: github.event.action != 'closed'
        runs-on: ubuntu-latest
        steps:
          - uses: junixlabs/devbox-preview@v1
            with:
              action: up
              server: ${{ secrets.DEVBOX_SERVER }}
            env:
              GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      preview-down:
        if: github.event.action == 'closed'
        runs-on: ubuntu-latest
        steps:
          - uses: junixlabs/devbox-preview@v1
            with:
              action: down
              server: ${{ secrets.DEVBOX_SERVER }}
            env:
              GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    ```

2. Set `DEVBOX_SERVER` as a repository secret with the server hostname or Tailscale IP.

### How it works

- On PR open/update: `devbox ci preview-up` creates a workspace for the PR
- On PR close/merge: `devbox ci preview-down` destroys it
- A commit status check links to the preview workspace

## Backup and Recovery

### Snapshots

Regularly snapshot important workspaces:

```bash
devbox snapshot my-project
```

Snapshots are stored as compressed tar archives of Docker volumes.

### Restore

```bash
devbox snapshot list
devbox restore my-project-20260414-1200
```

## Security Considerations

- **Tailscale ACLs**: Use [Tailscale ACLs](https://tailscale.com/kb/1018/acls/) to control which users can reach which servers
- **Docker group**: Only add trusted users to the docker group — docker access is effectively root access
- **SSH keys**: Use SSH key authentication, disable password auth
- **Resource limits**: Always set resource limits to prevent resource exhaustion

## Next Steps

- [Configuration Reference](../getting-started/config.md) — all `devbox.yaml` fields
- [Plugin API](../reference/plugin-api.md) — extend devbox with custom providers
- [Troubleshooting](../reference/troubleshooting.md) — common issues and fixes
