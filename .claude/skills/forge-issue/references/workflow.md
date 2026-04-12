# Issue Resolution Workflow

Step-by-step process for resolving a Forge issue.

## Step 1: Fetch Issue Data

For each documentId provided, fetch the full issue and its comments in parallel:

```
forge_issues → get → { documentId: "<id>" }
forge_comments → list → { filters: { issue: "<documentId>" } }
```

Review all returned data:
- **Core**: title, description, status, priority, category
- **Criteria**: acceptanceCriteria, aiAcceptanceCriteria
- **Solutions**: suggestedSolution, aiSuggestedSolution
- **Plan**: plan (pre-approved implementation plan, if set)
- **Context**: attachments, changeHistory, comments

## Step 2: Triage — Is This Actionable?

If the issue is too generic to act on:

```
forge_issues → update → { documentId: "<id>", data: { status: "needs_info" } }
forge_comments → create → { data: { body: "Moved to needs_info — ...", issue: "<documentId>" } }
```

Then **stop** — do not create a branch or proceed further.

## Step 3: Create Branch

```bash
git checkout -b ISS-XX-short-title
```

## Step 4: Read Codebase Context

- **`.forge/knowledge.json`** — project structure, conventions, available commands
- **`.forge/lessons.md`** — previous learnings, gotchas

## Step 5: Update Status

```
forge_issues → update → { documentId: "<id>", data: { status: "in_progress" } }
```

## Step 6: Plan or Execute

**If the issue has a `plan` field:** Execute it directly — do not re-plan.

**If no plan and task is complex:** Enter plan mode first. After planning, save to issue.

**If no plan and task is simple:** Proceed directly to implementation.

## Step 7: Implement

- Stay on the feature branch — never run `git checkout`
- Follow acceptance criteria
- Match existing patterns from knowledge.json
- Minimal changes only — don't refactor outside scope

## Step 8: Verify

Run the appropriate test command. Fix any test failures before proceeding.

## Step 9: Commit

- Conventional commit format: `feat:`, `fix:`, `refactor:`, etc.
- Reference the issue ID: `Resolves ISS-25`
- Stage specific files — avoid `git add .`

## Step 10: Code Review

Run `git diff HEAD~1` and review every changed file for bugs, dead code, edge cases, type issues.

## Step 11: Code Simplifier

Launch a code simplifier subagent to polish the implementation.

## Step 12: Post Comment

```
forge_comments → create → { data: { body: "<markdown>", issue: "<documentId>" } }
```

## Step 13: Resolve

```
forge_issues → update → { documentId: "<id>", data: { status: "resolved" } }
```

## Step 14: Capture Learnings

If you discovered anything useful, append to `.forge/lessons.md`.
