---
name: forge-staging
description: "Merge feature branch to baseBranch for staging deployment. Triggered when an issue reaches pass status — merges the ISS-* branch to the project's baseBranch and sets status to staging. Triggers on: /forge-staging, merging to staging, promoting to staging, deploying to staging. Also use when the pipeline needs to move an issue from pass to staging."
user_invocable: true
arguments: "documentId"
---

# Forge Staging

Merges the ISS-* feature branch to the project's baseBranch (staging) after QA passes. This is a git-only step — no code changes, just branch merging.

## Usage

```
/forge-staging <documentId>
```

## Tools

- **forge_issues** — get issue data, update status
- **forge_comments** — post staging comment
- **forge_config** — get project config (baseBranch)
- **forge_coolify_deploy** — trigger staging deployment
- **Codebase tools** — Bash (git commands)

## Workflow

### Step 1: Fetch Issue & Project Config

```
forge_issues → get → { documentId: "<id>" }
forge_config → get → {}
```

Read: `baseBranch` from project config (default: `main`). Verify status is `pass`. If not, stop.

### Step 2: Set In-Progress

```
forge_issues → update → { documentId: "<id>", data: { status: "in_progress" } }
```

### Step 3: Find Feature Branch

```bash
git fetch origin
git branch -r | grep -i "ISS-<issue-id>"
```

### Step 4: Merge to baseBranch

```bash
git checkout <baseBranch>
git pull origin <baseBranch>
git merge origin/ISS-XX-short-title --no-ff -m "Merge ISS-XX to <baseBranch> for staging"
git push origin <baseBranch>
```

If merge conflicts occur, stop and post a comment. Set status to `on_hold`.

### Step 5: Deploy to Staging

```
forge_coolify_deploy → deploy → {}
```

If no Coolify resources configured, skip.

### Step 6: Post Comment & Set Status

**Status update must be the LAST action.**

```
forge_comments → create → { data: { body: "**Staging** — Merged ISS-<id> to <baseBranch>. Coolify deploy triggered.", issue: "<documentId>", author: "Pidgeot" } }
forge_issues → update → { documentId: "<id>", data: { status: "staging" } }
```

## Output Rules (Save Tokens)

- **Zero narration.** Tool calls are self-documenting.
- **One-line status only.** "Merged to main." — nothing more.
- **Skip the recap.** The comment covers it.
