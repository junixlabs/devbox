# Mobile preview (Expo / React Native)

Turn a Linux box into a **mobile preview target**: devbox serves an Expo/React
Native branch to a real device (or emulator) over the network, with automatic
escalation to a native build. It's the mobile analog of a staging URL — hand a
connect URL / QR to a phone and load the app.

This uses the bare-metal **`host` runtime** instead of Docker, so the workspace
has direct access to the Android SDK, emulators, USB, and Metro.

---

## 1. Prepare the box

The server that runs the app (here called `dev1`) needs:

| Requirement | Check |
|---|---|
| Node.js + npm | `node -v` |
| Android SDK (for Android builds/emulator) | `adb version`, `$ANDROID_HOME` |
| KVM (emulator acceleration) | `ls /dev/kvm` |
| Tailscale (for tailnet connect) | `tailscale status` |
| SSH server + **passwordless SSH from the devbox client to the box** | `ssh dev1 echo ok` |
| A writable workspace root at `/workspaces` | see below |

**Passwordless SSH.** devbox drives the box over SSH — even when the box *is*
your local machine. Authorize your key and pin it:

```bash
cat ~/.ssh/id_ed25519.pub >> ~/.ssh/authorized_keys   # if targeting localhost
# ~/.ssh/config
# Host dev1
#     HostName dev1.your-tailnet.ts.net
#     IdentityFile ~/.ssh/id_ed25519
#     IdentitiesOnly yes
ssh dev1 'echo OK'      # must succeed with no password prompt
```

**Workspace root.** The `host` runtime clones into `/workspaces`, which must be
writable by the SSH user:

```bash
sudo mkdir -p /workspaces && sudo chown "$(id -un):$(id -gn)" /workspaces
```

**Register the box and health-check it:**

```bash
devbox server add dev1 dev1
devbox doctor --server dev1     # all checks should pass
```

---

## 2. Write the workspace config

A minimal `devbox.yaml` for an Expo app (see
[`examples/expo-mobile/devbox.yaml`](https://github.com/junixlabs/devbox/blob/main/examples/expo-mobile/devbox.yaml)):

```yaml
name: expo-preview
server: dev1
runtime: host
repo: git@github.com:your-org/your-expo-app.git
branch: main
setup:
  - cd mobile && npm install --no-audit --no-fund   # drop `cd mobile &&` if the app is at repo root
serve: bash -lc "cd mobile && exec npx expo start --port 8081"
ports:
  metro: 8081
env:
  EXPO_PUBLIC_API_URL: https://staging.api.example.com
```

Notes:

- **App in a subdirectory** (monorepo): `cd mobile && …` in both `setup` and
  `serve`. `serve` must be `exec`-able, so wrap it: `bash -lc "cd mobile && exec …"`.
- **`.env` is usually gitignored** — inject public runtime config via `env:`.
- **Port** — if `8081` is already used by another Metro instance on the box,
  pick a free one (`--port 8090`) and set `ports.metro` to match.

---

## 3. Start it

```bash
devbox up ./path-to-config-dir
```

devbox clones the repo, runs `setup`, and starts Metro as a detached process.
Get a machine-readable result (for scripts / CI / Forge) with `--json`:

```bash
devbox up --json ./path-to-config-dir
# { "status": "running", "connect_url": "exp://…:8081", "qr": "data:image/png;base64,…", "mode": "fast-refresh" }
```

Manage it like any workspace: `devbox list`, `devbox logs expo-preview`,
`devbox stop expo-preview`, `devbox destroy expo-preview`. Re-running `devbox up`
syncs the branch in place (fast-refresh, or a rebuild when native files change).

---

## 4. Connect a device

How the phone reaches Metro depends on the network. Set it via
`REACT_NATIVE_PACKAGER_HOSTNAME` (Metro's advertised host):

| Scenario | Config | Connect URL |
|---|---|---|
| **Device on the same tailnet** | leave `REACT_NATIVE_PACKAGER_HOSTNAME` unset — devbox auto-advertises the box's Tailscale MagicDNS host | `exp://<box>.<tailnet>.ts.net:8081` |
| **Device on the same Wi-Fi/LAN** (no Tailscale) | set `REACT_NATIVE_PACKAGER_HOSTNAME: <box-lan-ip>` | `exp://<box-lan-ip>:8081` |
| **Device on any other network** | change `serve` to `expo start --tunnel` (Expo relay) | the tunnel URL Expo prints |

Open the URL in **Expo Go** (or scan the QR). Expo Go must be recent enough for
your app's Expo SDK.

> After changing `REACT_NATIVE_PACKAGER_HOSTNAME`, restart the serve process so
> the new host is advertised: `devbox stop <name> && devbox up <dir>`.

---

## 5. Native builds (EAS)

Fast-refresh serves JS changes. When a change touches native code (deps,
`app.json`, config plugins, `android/`/`ios/`), you need a real build:

```bash
devbox up --build --profile preview ./path-to-config-dir
```

This runs `eas build --platform android` (authenticated by `EAS_TOKEN` from
`env:`) and returns the installable artifact URL through the same `--json`
result shape (`mode: "build"`). iOS builds require a macOS host and are out of
scope for a Linux box.

---

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `docker compose up failed … no service selected` on a `runtime: host` workspace | Stale `devbox` binary that predates host-runtime support — rebuild/upgrade devbox. |
| `mkdir /workspaces … permission denied` | Create it writable: `sudo mkdir -p /workspaces && sudo chown $USER /workspaces`. |
| `tailscale serve … Access denied` warning | Harmless for mobile — Expo connects to raw TCP, not the serve proxy. (Silence with `sudo tailscale set --operator=$USER` if you also want HTTP ports proxied.) |
| Workspace shows `running` but nothing serves | Port conflict — another Metro already holds the port and Expo skipped in non-interactive mode. Pick a free `--port`. |
| Phone shows **"Something went wrong"** | Usually Expo Go is older than the app's SDK (update Expo Go), or the phone can't reach the advertised host. Test from the phone's browser: `http://<advertised-host>:<port>/status` should return `packager-status:running`. |
| Off-tailnet device can't resolve `*.ts.net` | Use the LAN-IP or `--tunnel` option above. |
