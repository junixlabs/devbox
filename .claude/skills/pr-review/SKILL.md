---
name: pr-review
description: "Independent code review agent for forge-issue workflow. Reviews git diff with fresh context, checks against project skills and conventions."
version: 1.0.0
---

# PR Review

Independent code review agent launched as a subagent during forge-issue Step 10. Reviews the git diff with a fresh context window — no bias from implementation.

## How It's Called

From forge-issue workflow (Step 10), launched via Task tool:

```
Task → subagent_type: "general-purpose"
  mode: "bypassPermissions"
  prompt: "You are a code reviewer. Read the pr-review skill at .claude/skills/pr-review/SKILL.md and follow it exactly. Review the changes in the last N commits on this branch."
```

## Review Process

### 1. Load Context

Before reviewing any code, load project-specific rules:

1. **Read `CLAUDE.md`** — project-wide instructions and patterns
2. **Read `.forge/knowledge.json`** — project structure, conventions (if exists)
3. **Read `.forge/lessons.md`** — past gotchas (if exists)
4. **Detect tech stack from changed files** — run `git diff HEAD~N --name-only`:
   - Strapi files → read `.claude/skills/strapi/SKILL.md`
   - Next.js files → read `.claude/skills/nextjs/SKILL.md`
5. **Check for package-level CLAUDE.md** — read if exists

### 2. Get the Diff

```bash
git diff HEAD~N --stat          # overview of files changed
git diff HEAD~N                 # full diff
git log --oneline HEAD~N..HEAD  # commit messages
```

### 3. Review Checklist

**Bugs & Logic** — incorrect logic, off-by-one errors, null/undefined risks, race conditions, missing error handling
**Security** — injection, XSS, SQL injection, secrets in code, missing auth, unsanitized input
**Performance** — N+1 queries, unnecessary re-renders, memory leaks, large payloads without pagination
**TypeScript** — incorrect casts, `any` leaks, missing type narrowing, wrong generic params
**React** (if detected) — `useEffect` deps, unmounted state updates, unstable keys, missing cleanup
**Strapi** (if detected) — document service vs entity service, lifecycle infinite loops, missing populate, filter syntax
**Consistency** — follows project patterns, web/dev parity if both changed

### 4. Output Format

```markdown
## Code Review — ISS-XX

| # | File | Line | Severity | Finding |
|---|------|------|----------|---------|
| 1 | path/to/file.ts | 42 | Bug | Description |
| 2 | path/to/file.ts | 88 | Minor | Description |
| 3 | path/to/component.tsx | 15 | Low | Description |

### Summary
- X bugs found (must fix)
- Y minor issues (should fix)
- Z low-priority suggestions (optional)
```

Severity levels:
- **Bug** — incorrect behavior, will cause issues in production. Must fix.
- **Minor** — not a bug but problematic. Should fix.
- **Low** — style, naming, minor improvement. Optional.

### 5. Return Findings (Do NOT Fix)

**Important:** The review agent only reports findings — it does NOT fix code.

If no issues found:

```
## Code Review — ISS-XX
No issues found. Implementation looks clean.
```
