# Code Implementation Workflow

## Step 1: Fetch Issue Data

Fetch the issue and its comments in parallel:

```
forge_issues → get → { documentId: "<id>" }
forge_comments → list → { filters: { issue: "<documentId>" } }
```

## Step 2: Determine Mode & Complexity

Check comments for pipeline artifacts:
- **Triage comment** (starts with `**Triage**`) → pipeline mode, extract complexity (Simple/Medium/Complex)
- **Plan comment** (starts with `**Plan**`) → has forge-plan output

**Pipeline mode:** No preview deploy → merge to baseBranch, `forge_coolify_deploy`, `deploying`; Complex → `developed`
**Standalone mode:** exit as `closed`

## Step 3: Check Actionability

If no plan AND the issue is vague → set `needs_info`, post comment, stop.

## Step 4: Enter Worktree & Create Feature Branch

**Always use a worktree** to isolate from parallel agent sessions:

```
EnterWorktree → { name: "ISS-XX" }
```

Then fetch config and create the feature branch inside the worktree:

```
forge_config → get → {}
```

Use the `baseBranch` value (defaults to `main`):

```bash
git checkout <baseBranch> && git pull origin <baseBranch> && git checkout -b ISS-XX-short-title
```

## Step 5: Read Context (Conditional)

**If plan has file paths** (pipeline mode): Skip knowledge.json. Go straight to implementation.
**If no plan** (standalone mode): Read `.forge/knowledge.json` and `.forge/lessons.md`.
**Attachments**: If the issue has `attachments`, read them using the Read tool (images) or WebFetch (URLs).

## Step 6: Set In Progress

```
forge_issues → update → { documentId: "<id>", data: { status: "in_progress" } }
```

## Step 7: Implement

Follow the plan step-by-step. Read each file as you get to it. Minimal changes only.

## Step 8: Build

`npm run build` from the correct package directory. Fix any build errors.

## Step 9: Test

If the plan has an **API Test Plan**, run those tests against the local dev server. Skip for frontend-only.

## Step 10: Review (Tiered by Complexity)

**Simple:** Self-review only — re-read diff, check for obvious mistakes.
**Medium:** Quick review agent — Bug-severity findings only.
**Complex:** Full review agent — Bug + Minor findings + simplifier.

## Step 11: Simplify (Complex Only)

Launch code-simplifier subagent for Complex issues only.

## Step 12: Commit

```bash
git add <specific files>
git commit -m "feat: <what changed> — Resolves ISS-XX"
```

## Step 13: Push & Deploy

**Simple / Medium:** Push ISS-* branch, merge to baseBranch, trigger Coolify deploy.
**Complex:** Push feature branch only — do NOT merge to baseBranch.

## Step 14: Post Comment

Post a summary on the issue (see `references/comments.md` for style).

## Step 15: Set Status (LAST)

**Status update must be the LAST action.** It triggers downstream pipeline steps.

**No preview deploy or Simple/Medium:**
```
forge_issues → update → { documentId: "<id>", data: { status: "deploying" } }
```

**Complex:**
```
forge_issues → update → { documentId: "<id>", data: { status: "developed" } }
```

## Step 16: Exit Worktree

After pushing, exit and remove the worktree (changes are already on remote):

```
ExitWorktree → { action: "remove" }
```

## Step 17: Capture Learnings

If you discovered anything useful, append to `.forge/lessons.md`.
