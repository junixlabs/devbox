---
name: forge-fix
description: "Fix rejected Forge issues based on review or QA feedback. Use this skill when an issue has been reopened with rejection comments — reads the feedback, applies a scoped fix, builds, re-tests, and pushes. Triggers on: /forge-fix, fixing rejected issues, addressing review feedback, fixing QA failures, resolving reopen comments, fixing CI build failures. Also use when the pipeline needs to move an issue from reopen back to deploying."
user_invocable: true
arguments: "documentId"
---

# Forge Fix

Applies scoped fixes based on review or QA rejection feedback. This is NOT a full reimplementation — it reads what failed, fixes only that, and re-submits.

The key discipline: **fix what the feedback says, nothing more.** Expanding scope during a fix cycle leads to new bugs and infinite review loops.

## Usage

```
/forge-fix <documentId>
```

## Tools

- **forge_issues** — get issue data, update status
- **forge_comments** — read rejection feedback, post fix summary
- **forge_coolify_deploy** — trigger deployment after push
- **Codebase tools** — Read, Edit, Write, Glob, Grep, Bash

Read `references/fix-workflow.md` for parsing rejection formats, branch handling, and fix strategy details.

## Workflow

### Step 1: Fetch Issue & Feedback

```
forge_issues → get → { documentId: "<id>" }
forge_comments → list → { filters: { issue: "<documentId>" } }
```

Verify status is `reopen`. Find the latest rejection comment — either:
- **Code Review** (starts with `## Code Review`) — from forge-review, has severity table
- **QA Test Report** (starts with `**QA Test Report**`) — from forge-test, has pass/fail table

If feedback is unclear or missing → post clarifying comment, set `needs_info`, stop.

### Step 2: Understand the Feedback

Parse the rejection:
- **From review:** extract Bug and Minor severity findings. Ignore Low items.
- **From QA:** extract FAIL test cases and failure descriptions.

Each finding = one fix task. Don't invent additional fixes.

### Step 3: Enter Worktree & Check Out Branch

**Always use a worktree** to isolate from parallel agent sessions:

```
EnterWorktree → { name: "ISS-XX-fix" }
```

Then check out the ISS-* branch inside the worktree:

```bash
git fetch origin
git checkout ISS-XX-*
git pull origin ISS-XX-*
```

### Step 4: Apply Scoped Fixes

For each finding:
1. Read the affected file at the mentioned line
2. Fix the specific issue
3. Move to the next finding

**Do not:** refactor adjacent code, add new features, or change the overall approach.

### Step 5: Test

If the fix touches API endpoints: run real API tests. Frontend-only fixes: build is sufficient.

### Step 6: Commit & Push

```bash
git add <specific files>
git commit -m "fix: address review feedback — <summary>"
git push
```

Separate fix commit — don't amend or squash into the original.

### Step 7: Exit Worktree

After pushing, exit and remove the worktree (changes are already on remote):

```
ExitWorktree → { action: "remove" }
```

### Step 8: Deploy

```
forge_coolify_deploy → deploy → {}
```

### Step 8: Post Comment & Set Status

**Status update must be the LAST action.**

Set status based on complexity:
- **Simple / Medium:** `deploying`
- **Complex:** `developed`

## Output Rules (Save Tokens)

- **Zero narration.** Tool calls are self-documenting.
- **Code only.** When fixing, output only tool calls. No explanations between edits.
- **No preamble.** Don't restate the feedback — just fix.
- **One-line status only.** "Build passed" or "Fix applied, pushing." — nothing more.
- **Never repeat file contents.** After reading a file, just edit it.
- **Skip the recap.** The commit message and comment cover it.
