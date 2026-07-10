# Host-runtime executor

## Overview

devbox originally ran every workspace through Docker Compose on the remote server. `runtime: host` adds a second, container-free execution path for workloads that need native access to the box — an Android emulator, `/dev/kvm`, a USB device, or anything else that doesn't play well inside a container. A workspace picks its runtime with one config field; everything else (branch checkout, ports, env, Tailscale exposure) works the same way regardless of runtime.

```yaml
runtime: host       # "docker" (default) | "host"
setup:               # commands run once when the workspace is provisioned
  - npm install
serve: npm start     # long-lived command kept running (required for runtime: host)
```

## The executor seam

Before this change, `workspace.remoteManager` (`internal/workspace/manager.go`) called `docker.NewManager(...)` directly inside `Create`/`Start`/`Stop`/`Destroy`. That hardcoded the Docker path into the workspace lifecycle. `internal/plugin.Provider` looked like an extensibility seam but isn't — its only caller is `devbox plugin list`; it was never wired into workspace lifecycle and is not used here.

`internal/executor` introduces the real seam:

```go
type Executor interface {
    Deploy(ctx context.Context) error
    Up(ctx context.Context) error
    Down(ctx context.Context) error
    Logs(ctx context.Context, follow bool, stdout, stderr io.Writer) error
    Destroy(ctx context.Context) error
}

func New(sshExec ssh.Executor, cfg *config.DevboxConfig, host, name string) (Executor, error)
```

`New` selects an implementation from `cfg.Runtime` ("" and `"docker"` both select the Docker path). `remoteManager` calls `executor.New(...)` once per operation and no longer branches on runtime itself — it just calls `Deploy`/`Up`/`Down`/`Logs`/`Destroy` on whatever `Executor` it gets back.

## Docker path (unchanged)

`internal/executor/docker.go` is a thin adapter over the existing, untouched `internal/docker.Manager`. `Deploy` generates the compose YAML and calls `Manager.Deploy`; `Up`/`Down`/`Destroy` delegate directly; `Logs` delegates to `Manager.Logs` for the first service. Docker workspaces behave identically to before this change.

## Host path

`internal/executor/host.go` runs `setup` and `serve` over SSH, in `{WorkspacesRoot}/{name}/src` (the same directory the repo is cloned into during `Create`, for both runtimes).

**Why detached + PID file, not systemd/tmux:** `ssh.Executor.Run` is one-shot — the command exits when the SSH session ends. To keep `serve` alive across that, `Deploy`/`Up` launch it with `setsid` (so it becomes its own process group leader, detached from the SSH session) and redirect output to `serve.log`, capturing the launched PID into `serve.pid`:

```sh
cd {src} && setsid bash -c '{exports} exec {serve}' >{log} 2>&1 </dev/null & echo $! >{pid}
```

This assumes a user-managed box (single fixed device, no HA requirement) rather than a supervisor daemon — reasonable given the target use case (mobile dev boxes), but it means a host reboot does not auto-restart `serve`; `devbox up` (which calls `Up`) does.

- **Up** reads `serve.pid`, checks liveness with `kill -0`, and only relaunches `serve` if it isn't running. No re-clone, no re-setup — `setup` only runs once, in `Deploy`.
- **Down** sends `kill -TERM -- -{pid}` (the negative PID targets the whole process group `setsid` created) and removes `serve.pid`. The workdir (and any caches inside it) is left alone.
- **Destroy** calls `Down` then `rm -rf` the workdir.
- **Logs** either `tail -n +1 -f serve.log` (follow) or `cat serve.log` (dump), streamed the same way `docker compose logs` is.
- **PID** (an executor-specific method, not part of the `Executor` interface — see below) reads `serve.pid` back so the workspace manager can display it.

Env vars are exported ahead of `setup`/`serve` as `export KEY='value'; ...`, with keys restricted to `^[a-zA-Z_][a-zA-Z0-9_]*$` and values single-quote shell-escaped, so arbitrary values in `devbox.yaml`'s `env:` map can't break out of the command.

**Distinguishing "not running" from "can't reach the host":** the PID-file read always exits 0 (`cat serve.pid 2>/dev/null || true`), so a missing PID file (never started, or already stopped) comes back as an empty read, not a command failure. A non-nil error from that read therefore means the host is genuinely unreachable. `Down`/`Up` rely on this: a real connectivity failure is propagated as an error rather than being swallowed as "already stopped" — otherwise `devbox stop` could falsely report success while the process was actually left running.

## Per-branch caching

Because there's no container filesystem layer, `node_modules`/Gradle/etc. caches only exist if the host workdir persists. `Down`/`Stop` never touch the workdir — only `Destroy` removes it. Since workspace names are already branch-scoped (`workspace.FormatName` includes the branch), each branch gets its own persistent workdir for free; stopping and restarting a workspace reuses whatever `setup` already installed.

## Resource limits

`resources:` (cpus/memory) has no meaning without a container to apply cgroup/ulimit limits to. `DevboxConfig.ValidateForUp` warns (via `slog.Warn`) when `runtime: host` is combined with non-zero `resources`, but does not reject the config — the fields are simply unenforced.

## PID tracking in workspace state

`Workspace` gained `Runtime`, `Setup`, `Serve`, and `ServePID` fields. `Setup`/`Serve` are persisted so `remoteManager` can reconstruct the right executor on `Start`/`Stop`/`Destroy`/`Logs` without re-reading `devbox.yaml`. `ServePID` is populated opportunistically: `hostExecutor` implements an additional `executor.PIDReporter` interface (`PID(ctx) (int, error)`) that isn't part of the core `Executor` interface (Docker workspaces have no equivalent single PID). `remoteManager` type-asserts on it after `Deploy`/`Up` to populate `ServePID` for `devbox list`/`tui` display.

## What this does not do

- **Fixing `Template.Setup`** — templates already declare a `Setup []string` field but `Template.ToDevboxConfig` drops it; Docker workspaces still don't execute it. Wiring that up is out of scope here (Docker's runtime model has no `setup` step to run it against without also inventing one).
- **Host resource enforcement** — no cgroups/ulimit are applied under `runtime: host`, only a warning.
- **Live stats for host workspaces** — `DockerStats`/`ServerResources` remain Docker-only; `devbox list` shows `-` for CPU/memory usage on host workspaces the same way it does for any workspace with no matching container stats.

## Extension points

This abstraction is the base the rest of the mobile-preview epic builds on:

- **Metro/Tailscale exposure** — `Up`'s liveness check and `Logs`' log-file tail are the hooks for surfacing Metro's dev-server state without adding a second lifecycle model.
- **MCP preview output** — `Logs` (dump mode) gives a stable, non-interactive way to read recent serve output.
- **Idempotent refresh** — a future "reinstall deps" flow can reuse `Deploy`'s setup-then-serve sequence without needing a new interface method; today it isn't exposed as a separate operation.
