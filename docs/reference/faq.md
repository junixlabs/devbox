# FAQ

## General

### What is devbox?

devbox is a CLI tool that turns any Linux machine into a ready-to-use dev environment. You run `devbox up` and get an isolated workspace with Docker containers, services, and ports — all accessible via Tailscale networking.

### Is devbox free?

Yes. devbox is open source (MPL-2.0) and free to use. You provide your own hardware.

### What hardware do I need?

Any Linux machine with Docker and Tailscale. This can be:

- A spare laptop or desktop
- A mini PC (Intel NUC, Beelink, etc.)
- A cloud VPS ($5-20/month from any provider)
- An old workstation under your desk

Minimum: 2 CPU cores, 4GB RAM, 20GB disk. Recommended: 4+ cores, 16GB+ RAM.

### Does devbox work offline?

Yes — if your server is on your local network. Tailscale also works over LAN without internet connectivity. For remote servers, you need internet access to reach the Tailscale network.

### Which editors are supported?

Any editor that supports SSH remoting:

- **Zed** (recommended) — native SSH remote support
- **VS Code** — via Remote - SSH extension
- **JetBrains IDEs** — via Gateway or SSH
- **Neovim** — direct SSH
- **Emacs** — via TRAMP

### Can multiple people use the same server?

Yes. devbox automatically isolates workspaces per user using Tailscale identity. Two developers running `devbox up` on the same project get separate workspaces with no conflicts.

## Setup

### Do I need Kubernetes?

No. devbox uses Docker directly — no Kubernetes, no orchestrator, no cluster setup. Just Docker on a Linux machine.

### Can I use Podman instead of Docker?

Yes — via the [plugin system](plugin-api.md). Write a Podman provider plugin or use one from the community registry.

### Does devbox work with devcontainer.json?

Yes. devbox reads `.devcontainer/devcontainer.json` as a fallback configuration source. For full control, use `devbox.yaml`.

### Can I use Docker Compose?

Yes. Convert your existing Docker Compose file:

```bash
devbox init --from-compose docker-compose.yml
```

## Networking

### How does Tailscale networking work?

Tailscale creates an encrypted WireGuard tunnel between your machine and the server. Ports defined in `devbox.yaml` are exposed via Tailscale, giving you:

- **Automatic HTTPS** — via Tailscale's MagicDNS
- **DNS hostnames** — access services by server name (e.g., `dev1:8080`)
- **Access control** — use Tailscale ACLs to control who can reach what

No public IPs, no firewall rules, no VPN configuration needed.

### Do I need to open ports on my server?

No. Tailscale uses NAT traversal — no inbound ports needed. Everything goes through the encrypted WireGuard tunnel.

### Can I access workspaces from my phone/tablet?

Yes — if Tailscale is installed on the device. Access workspace ports via the Tailscale hostname.

## Workspaces

### How do I work on multiple branches?

Create separate workspaces per branch:

```bash
devbox up                          # main branch
devbox up --branch feature/auth    # feature branch
```

Each branch gets its own isolated workspace.

### Where is workspace data stored?

On the server, in Docker volumes. Data persists across `devbox stop` / `devbox up` cycles. Only `devbox destroy` removes data permanently.

### Can I back up a workspace?

Yes — use snapshots:

```bash
devbox snapshot my-project          # Save state
devbox restore my-project-snapshot  # Restore later
```

### What happens if the server reboots?

Docker containers stop. Run `devbox up` to restart them. Data in Docker volumes is preserved.

## Troubleshooting

### devbox doctor shows failures

Run `devbox doctor --server <name>` and fix issues in order:

1. **SSH** — verify `~/.ssh/config` and Tailscale connectivity
2. **Docker** — install Docker, add user to docker group
3. **Tailscale** — run `tailscale up` on the server
4. **Disk** — free up space if disk is full

See [Troubleshooting](troubleshooting.md) for detailed fixes.

### My workspace is slow

Check resource usage:

```bash
devbox stats my-project
```

Common causes:

- **CPU/memory limits too low** — increase in `devbox.yaml`
- **Disk full** — clear unused Docker images: `docker system prune`
- **Network issues** — check Tailscale connection quality

### How do I get help?

1. Check [Troubleshooting](troubleshooting.md)
2. Run `devbox doctor --server <name>`
3. Use verbose logging: `devbox -v up`
4. [Open an issue](https://github.com/junixlabs/devbox/issues)
