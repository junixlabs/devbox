---
name: antigravity-guide
description: "Environment guide for Antigravity server-side agent. High-level CLI for Forge API, Windows environment, common pitfalls. Auto-loaded by all pipeline steps."
user_invocable: false
---

# Antigravity Environment Guide

You are running inside **Antigravity** on a **Windows** host.

## Environment

- **OS**: Windows (`c:\Canaran.Antigravity.Service_Sessions\<sessionId>\`)
- **Repo**: Cloned into a subdirectory of your session root (not the root itself)
- **Node.js**: Available
- **Git**: Available
- **Shell**: May NOT have Bash — use native file tools (read, write, list)

## Forge API — Use the CLI

**DO NOT use curl.** Use the high-level CLI instead:

```
node forge-api.mjs get-issue <documentId>
node forge-api.mjs update-issue <documentId> --status=confirmed --category=bug
node forge-api.mjs update-issue <documentId> --plan=@plan.md --status=approved
node forge-api.mjs update-issue <documentId> --data-file=payload.json
node forge-api.mjs search-issues "keyword1 keyword2" --exclude=<documentId> --limit=10
node forge-api.mjs list-comments <issueDocumentId> --limit=5
node forge-api.mjs create-comment <issueDocumentId> --body=@file.md --author=Snorlax
node forge-api.mjs create-comment <issueDocumentId> --body=@file.md --author=Snorlax --attachments=42,43
node forge-api.mjs upload <filepath>
```

### Flags

- `--status=`, `--category=`, `--priority=`, `--complexity=` — simple string values, safe inline
- `--plan=@plan.md`, `--body=@file.md` — read value from file (for long text)
- `--relations=@relations.json` — read JSON from file
- `--data-file=payload.json` — read entire update body from JSON file
- `--attachments=42,43,44` — comma-separated media IDs (from upload command)

### Complex values: always use file-based payloads

For any value containing JSON (relations, sessionContext, arrays, objects), write to a file first and reference it.

**Updating an issue with complex fields:**
1. Write JSON file: `{ "status": "confirmed", "relations": [...] }`
2. Pass via `--data-file`: `node forge-api.mjs update-issue <documentId> --data-file=payload.json`

**Creating a comment with long body:**
1. Write body to `.md` file
2. `node forge-api.mjs create-comment <issueDocId> --body=@comment.md --author=Snorlax`

### Uploading Files (Screenshots)

```
node forge-api.mjs upload screenshot.png
```
Response: `{ "data": { "id": 42, "url": "...", "name": "screenshot.png" } }`

## File Operations

Use native tools, not Unix commands:
- **Read file**: file read tool (not `cat`)
- **List directory**: directory listing tool (not `ls`)
- **Write file**: file write tool (not `echo >`)

## Common Errors

1. **JSON parse error** → write data to a file first, use `--data-file=` or `--field=@file.json`
2. **"Bash not available"** → use native tools + `node forge-api.mjs`
3. **API errors** → never use curl, always use the CLI
4. **"command not found: cat/ls"** → use `type`/`dir` or native tools
5. **Git fails** → `cd` into repo dir first (not session root), then `git fetch && git pull`
