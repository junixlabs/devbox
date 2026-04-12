# devbox Infrastructure

## Dev Machine (dev1)
- This machine — Ubuntu desktop, i5-11400, 16GB RAM
- Tailscale hostname: `dev1`, IP: `100.102.160.5`
- Where devbox CLI code lives and runs

## Test VPS (devbox-vps)
- SSH: `ssh devbox-vps` or `ssh root@165.22.96.128`
- Tailscale hostname: `devbox-vps`, IP: `100.117.246.55`
- Ubuntu 24.04, 2 CPU, 3.8GB RAM
- Docker 29.2.0 + Compose v5.0.2
- Workspace dir: `/workspaces/`

## GitHub
- Repo: `junixlabs/devbox` (public, MPL-2.0)
- SSH remote: `git@github.com-junixlabs:junixlabs/devbox.git`
- gh CLI: logged in as `junixlabs`

## Forge
- Project: `home-kieutrung-tools-devbox`
- productionBranch: `main`
- baseBranch: `main` (staging branch added after v0.1.0)
- Auto-pipeline: triage, plan, code, review, test, staging, release

## Tailscale Tailnet: chuongle741@
- dev1: 100.102.160.5
- devbox-vps: 100.117.246.55
- chuongs-macbook-air: 100.76.107.39
