---
name: forge-code
description: "Implement code changes for Forge issues. Use this skill whenever approved issues need to be coded — creates branch, follows the plan, implements changes, builds, reviews, commits, and pushes. Triggers on: /forge-code, coding issues, implementing approved issues, writing code for an issue, building features from a plan. Also use when the pipeline needs to move an issue from approved to deploying."
user_invocable: true
arguments: "documentId1 documentId2 ..."
---

# Forge Code

The coding step in the issue pipeline: `approved → developed`. Implements code, validates it locally (build + test), then pushes. An independent review step follows.

When a plan exists (from forge-plan), this skill should be fast and focused — the plan already identified the files, the approach, and the patterns. Don't re-explore. Follow the plan, edit the files, test, commit.

## Usage

```
/forge-code <documentId>
/forge-code <documentId1> <documentId2>
```

## Tools

- **forge_issues** — get issue data, update status
- **forge_comments** — read plan/triage comments, post completion comment
- **forge_coolify_deploy** — trigger deployment after push
- **Codebase tools** — Read, Edit, Write, Glob, Grep, Bash


## Pre-check: Review Issue Detection

Before coding, check if this is a **review/approval issue** (not a code task):

If `issue.category === "review"` AND title starts with `"Review & Approve"`:
1. Extract target issue documentId from description or `depends_on` relations
2. Fetch target issue + all comments via `forge_issues` get + `forge_comments` list
3. **Check target plan quality:** approach clear, files listed, steps defined, risks noted
4. **Check blockers:** fetch each `blocked_by` issue — all must be at `tested`/`pass`/`closed`
5. **If blocker not done:** diagnose by status — `open` → kick off `/forge-triage`, `confirmed` → `/forge-plan`, `approved` → `/forge-code`, `reopen` → `/forge-fix`, `on_hold`/`needs_info` → escalate to owner. Comment on blocker requesting notification. Set review issue to `on_hold`.
6. **If all criteria pass:** set target to `approved`, comment approval summary, set review issue to `tested`
7. **If plan has issues** (architecture change, new deps, security): comment concerns, set review issue to `needs_info`
8. **Phase gates** (ISS-30, 38, 45, 50, 54): NEVER auto-approve — create summary, recommend to owner
9. **STOP here** — do NOT create branch, write code, or push. Return.

See `forge-approve` skill and its `references/decision-framework.md` for full criteria.

## Quick Start (Pipeline Mode)

When the issue has a plan and triage/plan comments from Forge AI:

1. Fetch issue + comments → extract plan and complexity from triage
2. **Enter worktree** for isolation: `EnterWorktree → { name: "ISS-XX" }` — prevents conflicts with parallel agent sessions
3. `forge_config → get` to read `baseBranch`, then `git checkout <baseBranch> && git pull && git checkout -b ISS-XX-short-title`
4. Set `in_progress`
5. Follow plan step-by-step — read each file as you reach it in the plan, edit, move on
6. Run build (`npm run build`) — catch compile/type errors
7. Test API (if plan has API Test Plan) — curl affected endpoints, verify responses. Skip for frontend-only.
8. Review (tiered — see below) — catch logic bugs
9. Fix any review findings, re-build, re-test
10. Commit
11. Push & deploy: push ISS-* branch, merge to baseBranch (Simple/Medium), trigger `forge_coolify_deploy`
12. Post comment
13. Set status (LAST — triggers next pipeline step): No preview deploy → `deploying`; Simple (staging URL) → `deploying` with previewUrl; Simple (no staging) / Medium → `deploying`; Complex → `developed`
14. **Exit worktree:** `ExitWorktree → { action: "remove" }` — changes are already pushed to remote

**Do NOT:** re-read knowledge.json (plan has the file paths), re-explore the codebase, second-guess the plan, read files that aren't in the plan.

Build and review happen BEFORE push. Only clean, reviewed code gets pushed and deployed.

Read `references/workflow.md` for the full step-by-step including standalone mode.

## Tiered Review

Review effort should match the risk. Over-reviewing trivial changes wastes tokens.

| Complexity | Review | Simplifier |
|-----------|--------|------------|
| **Simple** | Self-review: read your diff, check for obvious mistakes | Skip |
| **Medium** | Quick review agent: Bug-severity only, skip style | Skip |
| **Complex** | Full review agent: Bug + Minor findings | Run simplifier |

Complexity comes from the triage comment (extracted in Step 2 of the workflow).

## Key Rules

1. **Fetch issue first** — never assume data from prompt
2. **Worktree first** — `EnterWorktree → { name: "ISS-XX" }` before any git operations. This isolates your work from parallel agent sessions working on other issues.
3. **Plan = source of truth** — don't re-explore or re-plan
4. **Branch from baseBranch** — `forge_config → get` for `baseBranch`, then `git checkout <baseBranch> && git pull && git checkout -b ISS-XX-short-title`
5. **Stay on branch** — never switch mid-work
6. **Build + review before push** — never push unvalidated code
7. **Post a comment** — see `references/comments.md`
8. **Exit worktree after push** — `ExitWorktree → { action: "remove" }` to clean up

## Output Rules (Save Tokens)

- **Zero narration.** Do not say what you're about to do or what you just did. The tool calls are self-documenting.
- **Code only.** When implementing, output only tool calls (Edit/Write/Bash). No explanations between edits.
- **No preamble.** Don't explain the plan back — you already read it. Just execute.
- **One-line status only.** If you must communicate, keep it to one line: "Build failed: missing import in X.ts" or "All 5 steps done, pushing."
- **Never repeat file contents.** After reading a file, don't quote it back. Just edit it.
- **Skip the recap.** Don't summarize changes at the end. The commit message and comment cover it.
