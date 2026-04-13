# Agent Farm Setup Guide

Run multiple AI agents in parallel on shared infrastructure. Each agent gets an isolated workspace with its own containers, filesystem, and resource limits — managed automatically via the devbox MCP server.

## How It Works

```
Agent 1 ──stdio──▶ devbox mcp serve ──▶ Workspace (dev1)
Agent 2 ──stdio──▶ devbox mcp serve ──▶ Workspace (dev1)
Agent 3 ──stdio──▶ devbox mcp serve ──▶ Workspace (dev2)
Agent N ──stdio──▶ devbox mcp serve ──▶ Workspace (devN)
```

Each agent connects to its own `devbox mcp serve` process over stdio. On registration, devbox:

1. Auto-selects the least-loaded server from the pool
2. Creates an isolated workspace with resource limits
3. Returns the agent ID and workspace name
4. Enforces workspace isolation — agents cannot access each other's workspaces
5. Auto-cleans the workspace when the agent disconnects

## Prerequisites

- devbox installed and at least one server added to the pool
- Servers accessible via SSH with Docker installed
- Tailscale configured on all servers

## Step 1: Configure Server Pool

Add servers to the pool. devbox automatically distributes agents across them:

```bash
devbox server add dev1 100.64.0.1
devbox server add dev2 100.64.0.2
devbox server add dev3 100.64.0.3
```

Verify all servers are healthy:

```bash
devbox doctor --server dev1
devbox doctor --server dev2
devbox doctor --server dev3
```

## Step 2: Plan Resources

Each agent workspace consumes CPU and memory. Plan capacity based on your workload:

| Server Spec | Agents @ 1 CPU, 1GB | Agents @ 2 CPU, 2GB | Agents @ 4 CPU, 4GB |
|-------------|---------------------|---------------------|---------------------|
| 4 CPU, 16GB | 4 | 2 | 1 |
| 8 CPU, 32GB | 8 | 4 | 2 |
| 16 CPU, 64GB | 16 | 8 | 4 |
| 32 CPU, 128GB | 32 | 16 | 8 |

Leave 10-20% headroom for the host OS and Docker overhead.

## Step 3: Register Agents

Each agent connects via its own MCP session and registers:

```
→ devbox_agent_register
  {
    "name": "coder",
    "capabilities": ["code", "test"],
    "cpus": 1.0,
    "memory": "2g"
  }

← {
    "agent_id": "agent-coder-a1b2c3d4",
    "workspace": "agent-coder-a1b2c3d4",
    "server": "dev1"
  }
```

The agent now has a running workspace on `dev1` with 1 CPU and 2GB memory.

## Step 4: Agent Workflow

A typical agent session:

```
1. devbox_agent_register    → get workspace
2. devbox_workspace_exec    → clone repo, install deps
3. devbox_workspace_exec    → run tasks (code, test, build)
4. devbox_snapshot_create   → checkpoint before risky ops (optional)
5. devbox_workspace_exec    → more work
6. disconnect               → workspace auto-destroyed
```

### Execute Commands

```
→ devbox_workspace_exec
  {
    "name": "agent-coder-a1b2c3d4",
    "command": "cd /workspaces && git clone https://github.com/org/repo.git && cd repo && go test ./..."
  }

← {
    "stdout": "ok\tall tests passed\n",
    "stderr": "",
    "exit_code": 0
  }
```

### Check Resources

```
→ devbox_metrics
  { "workspace": "agent-coder-a1b2c3d4" }

← {
    "cpu_percent": 45.2,
    "mem_usage": 1073741824,
    "mem_limit": 2147483648,
    "disk_usage": 5368709120
  }
```

## Workspace Isolation

Agents can only interact with their own workspace:

- `devbox_workspace_exec` with another agent's workspace returns `FORBIDDEN`
- `devbox_workspace_destroy` with another agent's workspace returns `FORBIDDEN`
- `devbox_workspace_list` returns all workspaces (read-only, no isolation needed)
- `devbox_workspace_create` is blocked for registered agents — use `devbox_agent_register` instead

Non-agent MCP sessions (no `devbox_agent_register` call) bypass isolation and have full access — this is the single-user mode for direct integration.

## Crash Recovery

If an agent process crashes (SIGKILL, power loss), the workspace may be orphaned. devbox handles this automatically:

- **PID tracking**: Each agent's OS process ID is recorded in `~/.devbox/agents.json`
- **Stale pruning**: `devbox_agent_list` checks PIDs and cleans up dead sessions
- **Manual cleanup**: Use `devbox destroy <workspace-name>` to remove orphaned workspaces

No intervention is needed for graceful disconnects — the `devbox mcp serve` process cleans up on stdin EOF.

## Multi-Server Scaling

When no server is specified during agent registration, devbox selects the server with the most available resources (least-loaded selector). This distributes agents across the pool automatically.

To check current distribution:

```
→ devbox_server_list

← [
    {"name": "dev1", "status": "online", "resources": {"available_score": 0.35}},
    {"name": "dev2", "status": "online", "resources": {"available_score": 0.72}},
    {"name": "dev3", "status": "online", "resources": {"available_score": 0.90}}
  ]
```

The next agent registration would be placed on `dev3` (highest available score).

## Monitoring

### List Active Agents

```
→ devbox_agent_list

← [
    {"id": "agent-coder-a1b2c3d4", "name": "coder", "workspace": "agent-coder-a1b2c3d4", "server": "dev1"},
    {"id": "agent-tester-e5f6g7h8", "name": "tester", "workspace": "agent-tester-e5f6g7h8", "server": "dev2"}
  ]
```

### CLI Monitoring

```bash
# List all agent workspaces
devbox list | grep "agent-"

# Resource usage across all agents
devbox stats

# Interactive dashboard
devbox tui
```

## Claude Desktop Integration

Add devbox to Claude Desktop's MCP configuration:

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

Claude can then create workspaces, execute commands, and manage environments through natural language. See [`examples/claude-desktop-config.json`](../examples/claude-desktop-config.json) for a complete configuration.

## Best Practices

1. **Always set resource limits** — prevent runaway agents from consuming all server resources. Default is 1 CPU, 1GB memory.
2. **Use meaningful agent names** — names appear in `devbox_agent_list` and help identify workloads.
3. **Snapshot before risky operations** — use `devbox_snapshot_create` to checkpoint state before destructive actions.
4. **Monitor fleet utilization** — check `devbox_server_list` regularly to ensure servers aren't overloaded.
5. **Scale horizontally** — add more servers to the pool rather than increasing per-server density.
6. **Plan for crashes** — `devbox_agent_list` auto-prunes stale sessions, but monitor for orphaned workspaces during heavy usage.

## Further Reading

- [MCP Server Reference](MCP.md) — all tools, parameters, responses, and error codes
- [CLI Quick Start](QUICKSTART.md) — getting started with devbox
- [Configuration Reference](CONFIG.md) — all `devbox.yaml` fields
