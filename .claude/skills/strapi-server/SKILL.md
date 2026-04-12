---
name: strapi-server
description: Start, stop, and manage the Strapi development server. Use for running dev server, checking status, and testing API endpoints.
---

# Strapi Server Management

## Commands

```bash
python3 .claude/skills/strapi-server/scripts/server.py start   # Start server
python3 .claude/skills/strapi-server/scripts/server.py stop    # Stop server
python3 .claude/skills/strapi-server/scripts/server.py status  # Check status
python3 .claude/skills/strapi-server/scripts/server.py wait    # Wait for ready
python3 .claude/skills/strapi-server/scripts/server.py test    # Test API
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `--host` | localhost | Server host |
| `--port` | 1337 | Server port |
| `--timeout` | 120 | Wait timeout (seconds) |
| `-v` | - | Verbose output |

## Environment Variables

| Variable | Default |
|----------|---------|
| `STRAPI_HOST` | localhost |
| `STRAPI_PORT` | 1337 |
| `BACKEND_DIR` | backend |
