---
name: forge-issue
description: "Resolve Forge issues end-to-end. Invoke with: /forge-issue documentId1 documentId2. Fetches issue data via forge_issues MCP tool, creates branch, implements changes, posts comment, and resolves. Works from desktop app or Claude CLI interactive."
version: 2.2.0
user_invocable: true
arguments: "documentId1 documentId2 ..."
---

# Forge Issue

Self-contained skill for resolving issues from the Forge project management platform.

## Usage

```
/forge-issue <documentId>
/forge-issue <documentId1> <documentId2>
```

Arguments are issue documentIds. The skill fetches all issue data via MCP tools.

## Workflow

Read `references/workflow.md` for the step-by-step resolution process.

## MCP Tools

- **forge_issues** — list, get, create, update issues
- **forge_comments** — list, create comments on issues
- **forge_tasks** — list, get, create, update tasks
- **forge_memory** — store/retrieve facts across sessions

## Issue Lifecycle

```
open → approved → in_progress → resolved → confirmed → closed
                              ↘ failed → reopen (back to open)
```

## Key Rules

1. **Always fetch issue data first** via `forge_issues → get` — never assume data from prompt
2. **Triage before working** — if the issue is too vague to act on, set `needs_info` with a comment explaining what's missing, then stop
3. **Create a branch** via `git checkout -b` before starting work
4. **Execute approved plans directly** — if issue has a `plan`, do not re-plan
5. **Update status to `in_progress`** before writing any code
6. **Stay on your branch** — never `git checkout`
7. **Code review before finishing** — review all changes yourself for bugs, dead code, edge cases, type issues. Fix and commit separately
8. **Code simplify after review** — launch `code-simplifier` agent to clean up redundancy and complexity
9. **Post a comment** when done — see `references/comments.md`
10. **Mark `resolved`** only after tests pass, review is clean, and changes are committed
11. **Capture learnings** — append to `.forge/lessons.md`
