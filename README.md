# devbox

Turn any Linux machine into a ready-to-use dev environment in one command.

## Prerequisites

- A Linux server (Ubuntu 22.04+ recommended) accessible via SSH
- [Docker](https://docs.docker.com/engine/install/) installed on the server
- [Tailscale](https://tailscale.com/download) installed on both your machine and the server
- Go 1.22+ (to build from source)

## Install

```bash
# Build from source
git clone https://github.com/junixlabs/devbox.git
cd devbox
go build -o devbox ./cmd/devbox/
sudo mv devbox /usr/local/bin/
```

## Quick start

### 1. Configure your server

Make sure you can SSH into your server:

```bash
ssh dev1 "hostname"
```

### 2. Add devbox.yaml to your project

```yaml
name: my-project
server: dev1
repo: git@github.com:your-org/your-repo.git
services:
  - mysql:8.0
  - redis:7-alpine
ports:
  app: 8080
  mysql: 3306
env:
  APP_ENV: local
  DB_HOST: mysql
```

See `devbox.yaml.example` for a full example.

### 3. Start a workspace

```bash
devbox up my-project
```

### 4. Connect

```bash
# SSH into the workspace
devbox ssh my-project

# Or open in Zed
zed ssh://dev1/workspaces/my-project
```

## Commands

| Command | Description |
|---------|-------------|
| `devbox up [project]` | Create and start a workspace |
| `devbox stop <workspace>` | Stop a running workspace |
| `devbox list` | List all workspaces |
| `devbox destroy <workspace>` | Permanently remove a workspace |
| `devbox ssh <workspace>` | SSH into a workspace |

## License

[MPL-2.0](LICENSE)
