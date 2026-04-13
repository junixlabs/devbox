# AI Agent Builder Guide

This guide covers using devbox to give AI agents their own isolated dev environments — one workspace per agent, with programmatic control via CLI or MCP.

## Why devbox for AI agents?

AI coding agents need the same things human developers need: a machine to run code on, isolated from other work. devbox provides:

- **Workspace per agent** — each agent gets its own containers, filesystem, and ports
- **Programmatic control** — all operations available via CLI (scriptable)
- **Resource isolation** — CPU and memory limits prevent runaway agents
- **Automatic cleanup** — destroy workspaces when agents finish
- **Multi-server scaling** — distribute agents across a server pool

## Prerequisites

- devbox installed and configured ([Quick Start](../getting-started/quickstart.md))
- At least one server in the pool
- For MCP: devbox MCP server ([reference](../reference/mcp-server.md))

## Architecture

```
Agent Orchestrator
├── Agent 1 → devbox workspace (dev1) → Container + Services
├── Agent 2 → devbox workspace (dev1) → Container + Services
├── Agent 3 → devbox workspace (dev2) → Container + Services
└── Agent N → devbox workspace (devN) → Container + Services
```

Each agent gets a dedicated workspace with:

- Isolated filesystem (cloned repo)
- Its own service containers (databases, caches)
- Tailscale-exposed ports
- Resource limits (CPU, memory)

## Basic Usage: CLI Scripting

### Create a workspace for an agent

```bash
# Create a unique workspace per agent
AGENT_ID="agent-$(uuidgen | head -c 8)"

cat > /tmp/devbox-${AGENT_ID}.yaml << EOF
name: ${AGENT_ID}
server: dev1
repo: git@github.com:your-org/your-repo.git
branch: main
services:
  - redis:7-alpine
ports:
  app: 8080
env:
  AGENT_ID: ${AGENT_ID}
EOF

cd /tmp && devbox up
```

### Execute commands in the workspace

```bash
# Run commands via SSH
devbox ssh ${AGENT_ID} -- "cd /workspaces/${AGENT_ID} && go test ./..."
devbox ssh ${AGENT_ID} -- "cd /workspaces/${AGENT_ID} && make build"
```

### Check workspace status

```bash
devbox list | grep ${AGENT_ID}
devbox stats ${AGENT_ID}
```

### Cleanup when done

```bash
devbox destroy ${AGENT_ID}
```

## Orchestration Pattern

Here's a Python script pattern for managing agent workspaces:

```python
import subprocess
import uuid
import json
import yaml
from pathlib import Path

class AgentWorkspace:
    def __init__(self, server="dev1", template=None):
        self.agent_id = f"agent-{uuid.uuid4().hex[:8]}"
        self.server = server
        self.template = template

    def create(self, repo, branch="main", services=None):
        """Create a workspace for this agent."""
        config = {
            "name": self.agent_id,
            "server": self.server,
            "repo": repo,
            "branch": branch,
        }
        if services:
            config["services"] = services

        config_path = Path(f"/tmp/{self.agent_id}")
        config_path.mkdir(exist_ok=True)
        (config_path / "devbox.yaml").write_text(yaml.dump(config))

        subprocess.run(
            ["devbox", "up"],
            cwd=config_path,
            check=True,
        )
        return self.agent_id

    def exec(self, command):
        """Execute a command in the workspace."""
        result = subprocess.run(
            ["devbox", "ssh", self.agent_id, "--", command],
            capture_output=True,
            text=True,
        )
        return result.stdout, result.stderr, result.returncode

    def destroy(self):
        """Cleanup the workspace."""
        subprocess.run(
            ["devbox", "destroy", self.agent_id],
            check=True,
        )


# Usage
workspace = AgentWorkspace(server="dev1")
workspace.create(
    repo="git@github.com:your-org/project.git",
    services=["redis:7-alpine"],
)

stdout, stderr, code = workspace.exec("cd /workspaces && ls")
print(f"Files: {stdout}")

# When the agent is done
workspace.destroy()
```

## Resource Limits

Always set resource limits for agent workspaces to prevent runaway processes:

```yaml
name: agent-abc123
server: dev1
resources:
  cpus: 1.0      # 1 CPU core max
  memory: 2g     # 2GB memory max
```

For a pool of agents, plan your server capacity:

| Server | CPUs | Memory | Max Agents (1 CPU, 2GB each) |
|--------|------|--------|------|
| 4-core, 16GB | 4 | 16GB | 4 |
| 8-core, 32GB | 8 | 32GB | 8 |
| 16-core, 64GB | 16 | 64GB | 16 |

## Multi-Server Distribution

For large agent fleets, use server pools:

```bash
# Add servers to the pool
devbox server add dev1 100.64.0.1
devbox server add dev2 100.64.0.2
devbox server add dev3 100.64.0.3
```

When creating workspaces without specifying a server, devbox automatically selects the least-loaded server:

```yaml
name: agent-abc123
# No server specified — auto-selected from pool
repo: git@github.com:your-org/project.git
```

## MCP Integration

devbox exposes all operations via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). Any MCP-compatible client — Claude Desktop, custom scripts, or agent frameworks — can manage workspaces programmatically.

### Connect Claude Desktop

```json
{
  "mcpServers": {
    "devbox": {
      "command": "devbox",
      "args": ["mcp", "serve"]
    }
  }
}
```

Once connected, an AI agent can create workspaces, execute commands, run tests, and destroy environments — all through MCP tools.

See the [MCP Server Reference](../reference/mcp-server.md) for the full tool catalog, and the [Agent Farm Setup Guide](agent-farm.md) for running multi-agent fleets.

## Monitoring Agent Workspaces

### List all agent workspaces

```bash
devbox list | grep "agent-"
```

### Resource usage across all agents

```bash
devbox stats
```

### Interactive monitoring

```bash
devbox tui
```

## Best Practices

1. **Always set resource limits** — prevent agents from consuming all server resources
2. **Use unique workspace names** — include agent ID or task ID in the name
3. **Cleanup on completion** — always destroy workspaces when agents finish
4. **Monitor resource usage** — use `devbox stats` to track fleet utilization
5. **Use server pools** — distribute agents across multiple servers for reliability
6. **Snapshot before risky operations** — save workspace state before destructive agent actions

## Next Steps

- [Admin Guide](admin.md) — server setup and pool management
- [Configuration Reference](../getting-started/config.md) — all `devbox.yaml` fields
- [Plugin API](../reference/plugin-api.md) — extend devbox with custom lifecycle hooks
