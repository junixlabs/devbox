---
hide:
  - navigation
  - toc
---

<div class="hero" markdown>

# devbox

<p class="hero-tagline">
Turn any Linux machine into a ready-to-use dev environment in one command.<br>
No cloud bills. No Kubernetes. No DevOps required.
</p>

<div class="hero-cta">
<a href="getting-started/quickstart/" class="primary">Get Started</a>
<a href="https://github.com/junixlabs/devbox" class="secondary">GitHub</a>
</div>

</div>

<div class="terminal">
<span class="comment"># Create a workspace in seconds</span><br>
<span class="prompt">$</span> devbox init<br>
<span class="prompt">$</span> devbox up<br>
<br>
<span class="comment"># Connect with any editor</span><br>
<span class="prompt">$</span> devbox ssh my-project<br>
<span class="prompt">$</span> zed ssh://dev1/workspaces/my-project<br>
</div>

---

## Why devbox?

You own the hardware. You deserve a dev environment tool that respects that.

| | **devbox** | GitHub Codespaces | Coder | DevPod |
|---|---|---|---|---|
| **Cost** | Free (your hardware) | $40-80/dev/month | Free (self-hosted) | Free |
| **Setup complexity** | 1 command | Managed | Kubernetes required | Medium |
| **Self-hosted** | Yes | No | Yes | Yes |
| **Container runtime** | Docker | Proprietary | Docker/K8s | Docker/K8s/Cloud |
| **Networking** | Tailscale (HTTPS, DNS) | GitHub network | Manual | Manual |
| **Multi-workspace** | Yes | Yes | Yes | Yes |
| **Editor support** | Any (SSH) | VS Code, JetBrains | Any (SSH) | Any (SSH) |
| **CI/CD previews** | Built-in | GitHub only | Manual | No |
| **Plugin system** | Yes | No | Yes | No |
| **Active development** | Yes | Yes | Yes | Abandoned (2024) |

---

<div class="feature-grid" markdown>

<div class="feature-card" markdown>

### :rocket: One Command Setup

`devbox up` creates your workspace, starts services, clones your repo, and exposes ports — all in one step.

</div>

<div class="feature-card" markdown>

### :whale: Docker Isolation

Each workspace runs in isolated Docker containers with configurable CPU, memory, and disk limits. No cross-project interference.

</div>

<div class="feature-card" markdown>

### :globe_with_meridians: Tailscale Networking

Automatic HTTPS, DNS hostnames, and access control. Connect from anywhere on your tailnet without port forwarding.

</div>

<div class="feature-card" markdown>

### :desktop_computer: Any Editor

Works with Zed, VS Code, JetBrains, Neovim, or any editor that supports SSH remoting. No proprietary lock-in.

</div>

<div class="feature-card" markdown>

### :jigsaw: Plugin System

Extend devbox with custom container providers (Podman, LXC) and lifecycle hooks. JSON protocol for any language.

</div>

<div class="feature-card" markdown>

### :bar_chart: Built-in Monitoring

Real-time CPU, memory, disk, and network metrics per workspace. Interactive TUI dashboard for management.

</div>

</div>

---

## Install

=== "Linux (amd64)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-amd64 -o devbox
    chmod +x devbox && sudo mv devbox /usr/local/bin/
    ```

=== "Linux (arm64)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-linux-arm64 -o devbox
    chmod +x devbox && sudo mv devbox /usr/local/bin/
    ```

=== "macOS (Apple Silicon)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-arm64 -o devbox
    chmod +x devbox && sudo mv devbox /usr/local/bin/
    ```

=== "macOS (Intel)"

    ```bash
    curl -fsSL https://github.com/junixlabs/devbox/releases/latest/download/devbox-darwin-amd64 -o devbox
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

---

## Quick Start

```bash
# 1. Check your server is ready
devbox doctor --server dev1

# 2. Create a config in your project
devbox init

# 3. Start a workspace
devbox up

# 4. Connect
devbox ssh my-project
```

[Full Quick Start Guide :arrow_right:](getting-started/quickstart.md){ .md-button }

---

## Architecture

```
 Your Machine                        Linux Server (dev1)
┌─────────────┐     Tailscale      ┌──────────────────────────┐
│             │    ◄──────────►    │  devbox daemon            │
│  Editor     │     Encrypted      │  ┌────────────────────┐  │
│  (Zed/VS    │     WireGuard      │  │ Workspace Container │  │
│   Code)     │     Tunnel         │  │  ├── App Code       │  │
│             │                    │  │  ├── MySQL 8.0      │  │
│  devbox CLI │                    │  │  └── Redis 7        │  │
└─────────────┘                    │  └────────────────────┘  │
                                   │                          │
                                   │  Docker + Tailscale      │
                                   └──────────────────────────┘
```

devbox connects your local editor to Docker containers on any Linux machine via Tailscale's encrypted WireGuard tunnel. No public IPs, no firewall rules, no VPN configuration.

---

## What's Next?

<div class="feature-grid" markdown>

<div class="feature-card" markdown>

### For Developers

Daily workflow from `devbox init` to production-ready workspace. Templates, snapshots, and multi-branch support.

[Developer Guide :arrow_right:](guides/developer.md)

</div>

<div class="feature-card" markdown>

### For Admins

Set up server pools, manage users, configure resource limits, and monitor your fleet.

[Admin Guide :arrow_right:](guides/admin.md)

</div>

<div class="feature-card" markdown>

### For AI Agent Builders

Give your AI agents isolated dev environments. MCP integration, workspace-per-agent, and automated cleanup.

[Agent Builder Guide :arrow_right:](guides/agent-builder.md)

</div>

</div>
