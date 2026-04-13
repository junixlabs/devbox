# MCP Server Reference

devbox exposes all workspace, server, snapshot, template, and agent operations via the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/). Any MCP-compatible client — Claude Desktop, custom scripts, or agent frameworks — can manage dev environments programmatically.

## Quick Start

Start the MCP server:

```bash
devbox mcp serve
```

The server communicates over **stdio** (stdin/stdout). Each `devbox mcp serve` process handles one client connection.

### Claude Desktop

Add to your `claude_desktop_config.json`:

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

See [`examples/claude-desktop-config.json`](../examples/claude-desktop-config.json) for a complete example.

### Custom Client

Any process that speaks MCP over stdio can connect:

```bash
# Python example — see examples/agent-script.py for full implementation
devbox mcp serve < request.jsonl > response.jsonl
```

---

## Tools

### Workspace Tools

#### `devbox_workspace_create`

Create a new workspace with optional template and server auto-selection.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Workspace name (must be unique) |
| `server` | string | no | Target server name. Auto-selected from pool if omitted |
| `template` | string | no | Template name to use (local or registry) |
| `repo` | string | no | Git repository URL to clone |
| `branch` | string | no | Git branch to checkout (default: `main`) |
| `services` | string[] | no | Service images (e.g., `["redis:7-alpine"]`) |
| `env` | object | no | Environment variables as key-value pairs |

**Example response:**

```json
{
  "name": "my-workspace",
  "server": "dev1",
  "status": "running",
  "user": "alice",
  "services": ["redis:7-alpine"],
  "ports": {"app": 8080}
}
```

#### `devbox_workspace_list`

List all workspaces, optionally filtered by user.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user` | string | no | Filter by workspace owner |

**Example response:**

```json
[
  {
    "name": "my-workspace",
    "server": "dev1",
    "status": "running",
    "user": "alice"
  },
  {
    "name": "test-env",
    "server": "dev2",
    "status": "stopped",
    "user": "bob"
  }
]
```

#### `devbox_workspace_exec`

Execute a command inside a running workspace. Returns stdout, stderr, and exit code.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Workspace name |
| `command` | string | yes | Shell command to execute |

**Example response:**

```json
{
  "stdout": "ok\n",
  "stderr": "",
  "exit_code": 0
}
```

#### `devbox_workspace_destroy`

Permanently remove a workspace and its containers.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Workspace name |

**Example response:**

```json
{
  "destroyed": "my-workspace"
}
```

---

### Server Tools

#### `devbox_server_list`

List all servers in the pool with health status and available resources.

No parameters.

**Example response:**

```json
[
  {
    "name": "dev1",
    "host": "100.64.0.1",
    "status": "online",
    "health": {
      "ssh": true,
      "docker": true,
      "tailscale": true
    },
    "resources": {
      "total_cpus": 8,
      "cpu_used_percent": 45.2,
      "total_memory_bytes": 34359738368,
      "used_memory_bytes": 17179869184,
      "available_score": 0.55
    }
  },
  {
    "name": "dev2",
    "host": "100.64.0.2",
    "status": "offline",
    "health": {
      "ssh": false,
      "docker": false,
      "tailscale": true
    }
  }
]
```

Offline servers are included with `"status": "offline"` and no resource data.

#### `devbox_server_status`

Get detailed status for a single server including per-workspace resource breakdown.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Server name |

**Example response:**

```json
{
  "name": "dev1",
  "host": "100.64.0.1",
  "health": {
    "ssh": true,
    "docker": true,
    "tailscale": true
  },
  "resources": {
    "total_cpus": 8,
    "cpu_used_percent": 45.2,
    "total_memory_bytes": 34359738368,
    "used_memory_bytes": 17179869184,
    "total_disk": 107374182400,
    "used_disk": 53687091200
  },
  "workspaces": [
    {
      "container": "my-workspace-app-1",
      "cpu_percent": 12.5,
      "mem_usage": 536870912,
      "mem_limit": 2147483648,
      "disk_usage": 1073741824,
      "net_in": 104857600,
      "net_out": 52428800
    }
  ]
}
```

#### `devbox_metrics`

Get resource metrics for a specific workspace or all workspaces on a server.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `workspace` | string | no | Workspace name (returns single workspace metrics) |
| `server` | string | no | Server name (returns all workspace metrics on that server) |

Provide either `workspace` or `server` — not both, not neither.

**Example response (workspace):**

```json
{
  "container": "my-workspace-app-1",
  "cpu_percent": 12.5,
  "mem_usage": 536870912,
  "mem_limit": 2147483648,
  "disk_usage": 1073741824,
  "net_in": 104857600,
  "net_out": 52428800
}
```

---

### Snapshot Tools

#### `devbox_snapshot_create`

Save the current state of a workspace (Docker volumes as compressed tar archive).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Workspace name to snapshot |
| `server` | string | yes | Server where the workspace runs |

**Example response:**

```json
{
  "snapshot": "my-workspace-20260414-153000",
  "workspace": "my-workspace",
  "server": "dev1",
  "size_bytes": 524288000
}
```

#### `devbox_snapshot_restore`

Restore a workspace from a previously saved snapshot.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `snapshot` | string | yes | Snapshot name |
| `server` | string | yes | Server to restore on |

**Example response:**

```json
{
  "restored": "my-workspace-20260414-153000",
  "workspace": "my-workspace",
  "server": "dev1"
}
```

#### `devbox_snapshot_list`

List available snapshots on a server.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `server` | string | yes | Server name |

**Example response:**

```json
[
  {
    "name": "my-workspace-20260414-153000",
    "workspace": "my-workspace",
    "created_at": "2026-04-14T15:30:00Z",
    "size_bytes": 524288000
  }
]
```

---

### Template Tools

#### `devbox_template_list`

List available templates (built-in and user-defined).

No parameters.

**Example response:**

```json
[
  {
    "name": "go-service",
    "description": "Go microservice with Redis and PostgreSQL",
    "source": "builtin"
  },
  {
    "name": "react-app",
    "description": "React + Vite development environment",
    "source": "local"
  }
]
```

#### `devbox_template_search`

Search the community template registry.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | yes | Search query |

**Example response:**

```json
[
  {
    "name": "python-ml",
    "description": "Python ML environment with Jupyter and GPU support",
    "author": "community",
    "downloads": 1250
  }
]
```

---

### Agent Tools

These tools enable multi-agent session management. See the [Agent Farm Setup Guide](AGENT_FARM.md) for full details.

#### `devbox_agent_register`

Register an agent in the current MCP session. Auto-creates an isolated workspace with resource limits.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Agent name |
| `capabilities` | string[] | no | Agent capabilities (e.g., `["code", "test"]`) |
| `cpus` | number | no | CPU limit for agent workspace (default: `1.0`) |
| `memory` | string | no | Memory limit for agent workspace (default: `"1g"`) |

**Example response:**

```json
{
  "agent_id": "agent-coder-a1b2c3d4",
  "workspace": "agent-coder-a1b2c3d4",
  "server": "dev1"
}
```

Each MCP session supports one agent registration. The workspace is automatically destroyed when the session disconnects.

#### `devbox_agent_list`

List all active agents and their workspace assignments. Automatically prunes stale sessions (dead PIDs).

No parameters.

**Example response:**

```json
[
  {
    "id": "agent-coder-a1b2c3d4",
    "name": "coder",
    "capabilities": ["code", "test"],
    "workspace": "agent-coder-a1b2c3d4",
    "server": "dev1",
    "registered_at": "2026-04-14T10:30:00Z"
  }
]
```

#### `devbox_agent_workspace`

Get the current agent's workspace details. Only available after `devbox_agent_register`.

No parameters.

**Example response:**

```json
{
  "name": "agent-coder-a1b2c3d4",
  "server": "dev1",
  "status": "running",
  "user": "agent-coder",
  "agent_id": "agent-coder-a1b2c3d4"
}
```

---

## Error Codes

All error responses include an `error_code` and human-readable `message`:

```json
{
  "error_code": "NOT_FOUND",
  "message": "workspace 'my-workspace' not found"
}
```

| Code | Description |
|------|-------------|
| `NOT_FOUND` | Resource (workspace, server, snapshot) does not exist |
| `INVALID_INPUT` | Missing or invalid parameters |
| `NOT_RUNNING` | Operation requires a running workspace but it is stopped |
| `FORBIDDEN` | Agent attempted to access another agent's workspace |
| `INTERNAL` | Unexpected server error |

---

## Tool Summary

| Tool | Category | Description |
|------|----------|-------------|
| `devbox_workspace_create` | Workspace | Create a new workspace |
| `devbox_workspace_list` | Workspace | List workspaces |
| `devbox_workspace_exec` | Workspace | Execute command in workspace |
| `devbox_workspace_destroy` | Workspace | Remove a workspace |
| `devbox_server_list` | Server | List servers with health and resources |
| `devbox_server_status` | Server | Detailed single server status |
| `devbox_metrics` | Server | Resource metrics per workspace or server |
| `devbox_snapshot_create` | Snapshot | Save workspace state |
| `devbox_snapshot_restore` | Snapshot | Restore workspace from snapshot |
| `devbox_snapshot_list` | Snapshot | List available snapshots |
| `devbox_template_list` | Template | List local + built-in templates |
| `devbox_template_search` | Template | Search community registry |
| `devbox_agent_register` | Agent | Register agent + auto-create workspace |
| `devbox_agent_list` | Agent | List active agents |
| `devbox_agent_workspace` | Agent | Get current agent's workspace |
