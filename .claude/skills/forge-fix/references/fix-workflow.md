# Fix Workflow

## Core Principle

Fix what the feedback says, nothing more. The plan was already approved and the code was already reviewed — expanding scope during a fix cycle introduces new bugs and triggers infinite review loops.

## Parsing Rejection Feedback

### From Code Review (forge-review)

Starts with `## Code Review`. Contains a severity table:

```
| # | File | Line | Severity | Finding |
```

- **Bug** — must fix. Incorrect behavior.
- **Minor** — should fix. Problematic pattern.
- **Low** — skip unless trivial. Style/naming.

Fix Bug and Minor items. Skip Low unless it's a one-line change.

### From QA Report (forge-test)

Starts with `**QA Test Report**`. Contains a pass/fail table:

```
| # | Test Case | Source | Result | Notes |
```

Look at FAIL rows. The **Notes** column and **Failures** section describe what went wrong.

### From CI Build Failure

Starts with `**Preview deploy failed**`. Contains build output in a code block. Parse the error — usually a compile error, missing import, or test failure.

## Branch Handling

**Always use a worktree** to isolate from parallel agent sessions:

```
EnterWorktree → { name: "ISS-XX-fix" }
```

Then check out the ISS-* feature branch inside the worktree (never directly on baseBranch):

```bash
git fetch origin
git checkout ISS-XX-*
git pull origin ISS-XX-*
```

After fixing and committing, push ISS-*. Only merge to baseBranch if staging deploys from baseBranch.

## Fix Strategy

For each finding:
1. Read the affected file at the mentioned line
2. Understand the surrounding context (read 20-30 lines around it)
3. Apply the minimal fix that addresses the finding
4. Move to the next finding

**Do not:**
- Refactor adjacent code that wasn't mentioned
- Add new features or "improvements"
- Change the overall approach or architecture
- Touch files that aren't related to the findings

## Build + Test Before Push

After all fixes applied:
1. `npm run build` — verify no compile errors
2. If API endpoints were changed: curl affected endpoints to verify responses
3. Fix any failures before pushing

## Commit Convention

Separate fix commit — never amend or squash into the original:

```bash
git add <specific files>
git commit -m "fix: address review feedback — <1-line summary>"
git push
```

## When Feedback is Unclear

If the rejection comment doesn't provide enough detail:
- Post a clarifying comment asking specific questions
- Set `status: needs_info`
- Stop — don't guess at fixes
