# Plugin API

## Overview

The devbox plugin system allows extending devbox with custom container providers (Docker, Podman, LXC) and lifecycle hooks. Plugins can be built-in (compiled into the binary) or external (standalone executables discovered at runtime).

## Plugin Types

### Provider

A Provider handles container lifecycle operations. The built-in Docker provider is the default.

```go
type Provider interface {
    Create(name string, image string, opts CreateOpts) error
    Start(name string) error
    Stop(name string) error
    Destroy(name string) error
    Status(name string) (string, error)
    ProviderName() string
}

type CreateOpts struct {
    Ports   map[string]int
    Env     map[string]string
    CPUs    float64
    Memory  string
    Volumes []string
    Command string
}
```

### Hook

A Hook runs custom logic before or after workspace lifecycle events.

```go
type Hook interface {
    Execute(ctx HookContext) error
    Events() []Event
    HookName() string
}

type HookContext struct {
    WorkspaceName string
    Event         Event
    ServerHost    string
    Env           map[string]string
}
```

### Lifecycle Events

| Event | Description |
|-------|-------------|
| `pre_create` | Before workspace creation |
| `post_create` | After workspace creation |
| `pre_start` | Before workspace start |
| `post_start` | After workspace start |
| `pre_stop` | Before workspace stop |
| `post_stop` | After workspace stop |
| `pre_destroy` | Before workspace destruction |
| `post_destroy` | After workspace destruction |

## Plugin Manifest

Every plugin must include a `plugin.yaml` file:

```yaml
name: my-provider
version: "1.0.0"
type: provider          # "provider" or "hook"
entrypoint: ./my-provider  # path to executable (relative to plugin dir)
description: Custom container provider using Podman
events:                    # only for hooks — events to subscribe to
  - pre_create
  - post_destroy
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique plugin name |
| `version` | string | Semantic version |
| `type` | string | `provider` or `hook` |
| `entrypoint` | string | Path to executable |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Human-readable description |
| `events` | list | Hook events to subscribe to |

## External Plugin Protocol

External plugins communicate via JSON over stdin/stdout.

### Request Format

```json
{
  "action": "create",
  "params": {
    "name": "workspace-name",
    "image": "ubuntu:22.04",
    "ports": {"app": 8080},
    "env": {"NODE_ENV": "development"},
    "cpus": 2.0,
    "memory": "4g"
  }
}
```

### Provider Actions

| Action | Params | Description |
|--------|--------|-------------|
| `create` | name, image, ports, env, cpus, memory, volumes, command | Create container |
| `start` | name | Start container |
| `stop` | name | Stop container |
| `destroy` | name | Remove container |
| `status` | name | Get container status |

### Hook Actions

| Action | Params | Description |
|--------|--------|-------------|
| `execute` | workspace, event, server, env | Run hook logic |

### Response Format

Success:
```json
{
  "ok": true,
  "result": "running"
}
```

Error:
```json
{
  "ok": false,
  "error": "container not found"
}
```

## Plugin Directory Structure

```
~/.config/devbox/plugins/
├── my-provider/
│   ├── plugin.yaml
│   └── my-provider      # executable
└── notify-hook/
    ├── plugin.yaml
    └── notify-hook       # executable
```

## CLI Commands

```bash
devbox plugin list              # List all plugins (built-in + external)
devbox plugin install <path>    # Install plugin from local directory
devbox plugin remove <name>     # Remove an external plugin
```

## Example: Minimal Provider Plugin

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type Request struct {
    Action string         `json:"action"`
    Params map[string]any `json:"params"`
}

type Response struct {
    OK     bool   `json:"ok"`
    Result string `json:"result,omitempty"`
    Error  string `json:"error,omitempty"`
}

func main() {
    var req Request
    if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
        respond(Response{Error: err.Error()})
        return
    }

    name, _ := req.Params["name"].(string)

    switch req.Action {
    case "create":
        image, _ := req.Params["image"].(string)
        // Your container creation logic here
        fmt.Fprintf(os.Stderr, "Creating %s with image %s\n", name, image)
        respond(Response{OK: true})
    case "start":
        respond(Response{OK: true})
    case "stop":
        respond(Response{OK: true})
    case "destroy":
        respond(Response{OK: true})
    case "status":
        respond(Response{OK: true, Result: "running"})
    default:
        respond(Response{Error: "unknown action: " + req.Action})
    }
}

func respond(r Response) {
    json.NewEncoder(os.Stdout).Encode(r)
}
```

Build and install:

```bash
go build -o my-provider .
mkdir -p ~/.config/devbox/plugins/my-provider
cp my-provider plugin.yaml ~/.config/devbox/plugins/my-provider/
devbox plugin list   # verify it shows up
```
