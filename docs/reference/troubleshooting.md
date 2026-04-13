# Troubleshooting

Start with `devbox doctor` — it checks most prerequisites automatically:

```bash
devbox doctor --server dev1
```

## Common Issues

### "failed to read config file" / config file not found

**Cause:** No `devbox.yaml` in the current directory.

**Fix:**

```bash
devbox init
```

Or specify the project directory:

```bash
devbox up ./path/to/project
```

### "'name' is required" / "'server' is required"

**Cause:** `devbox.yaml` is missing the `name` or `server` field.

**Fix:** Add the missing field to your `devbox.yaml`:

```yaml
name: my-project
server: dev1
```

See [Configuration Reference](../getting-started/config.md) for all required fields.

### Cannot connect to server via SSH

**Cause:** SSH is not configured for the server name, or the server is unreachable.

**Fix:**

1. Verify SSH access:
   ```bash
   ssh dev1 "hostname"
   ```

2. If it fails, add an entry to `~/.ssh/config`:
   ```
   Host dev1
       HostName 100.x.x.x
       User your-username
   ```

3. Check that Tailscale is running on both machines:
   ```bash
   tailscale status
   ```

### Docker not found on server

**Cause:** Docker is not installed or the daemon is not running on the server.

**Fix:**

1. Install Docker on the server:
   ```bash
   ssh dev1 "curl -fsSL https://get.docker.com | sh"
   ```

2. Start the Docker daemon:
   ```bash
   ssh dev1 "sudo systemctl start docker"
   ```

3. Add your user to the docker group (avoids needing `sudo`):
   ```bash
   ssh dev1 "sudo usermod -aG docker $USER"
   ```

### Tailscale serve failed

**Cause:** Tailscale is not running, not logged in, or doesn't have permissions to serve ports.

**Fix:**

1. Check Tailscale status on the server:
   ```bash
   ssh dev1 "tailscale status"
   ```

2. If not connected, log in:
   ```bash
   ssh dev1 "sudo tailscale up"
   ```

3. Verify that Tailscale Funnel/Serve is enabled in your [Tailscale admin console](https://login.tailscale.com/admin/dns) under DNS settings.

### Port already in use

**Cause:** Another container or process is using the same port on the server.

**Fix:**

1. Check what's using the port:
   ```bash
   ssh dev1 "sudo lsof -i :8080"
   ```

2. Either stop the conflicting process or change the port in `devbox.yaml`:
   ```yaml
   ports:
     app: 8081  # Use a different port
   ```

### Workspace already exists

**Cause:** A workspace with the same name already exists on the server.

**What happens:** `devbox up` automatically starts the existing workspace instead of creating a new one. This is expected behavior.

**To start fresh:**

```bash
devbox destroy my-project
devbox up
```

### Slow clone / large repository

**Cause:** The git repository is large and takes time to clone over SSH.

**Fix:**

1. Use a shallow clone by keeping your repo lean
2. Ensure the server has good network connectivity
3. If the repo is already cloned, `devbox up` will reuse it (only the first run is slow)

### Permission denied on server

**Cause:** Your user doesn't have permission to run Docker commands or access the workspace directory.

**Fix:**

1. Add your user to the docker group:
   ```bash
   ssh dev1 "sudo usermod -aG docker $USER"
   ```

2. Log out and back in for the group change to take effect:
   ```bash
   ssh dev1  # reconnect
   ```

## Getting Help

If your issue isn't listed here:

1. Run `devbox doctor --server <name>` and check all health checks pass
2. Run with verbose logging: `devbox -v up`
3. [Open an issue](https://github.com/junixlabs/devbox/issues) with the error output
