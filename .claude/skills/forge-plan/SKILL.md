---
name: forge-plan
description: "Write implementation plans for confirmed Forge issues. Use this skill whenever an issue needs a plan before coding begins — exploring the codebase, identifying affected files, and writing step-by-step implementation instructions into the issue's plan field. Triggers on: /forge-plan, planning issues, writing implementation plans, exploring codebase for an issue, preparing issues for development, moving issues from confirmed to approved. Also use when the pipeline needs to advance an issue from confirmed status."
user_invocable: true
arguments: "documentId"
---

# Forge Plan

This is the second step in the issue pipeline: `confirmed → approved`. Its job is to turn a triaged issue into a concrete implementation plan that a coding agent (or developer) can follow without re-exploring the codebase.

Planning is the highest-value step in the pipeline. A good plan saves the coding step from wasting tokens on exploration, wrong turns, and rework.

## Usage

```
/forge-plan <documentId>
```

## Tools

- **forge_issues** — get issue data, write plan back to `plan` field
- **forge_comments** — read triage comment (complexity), post plan comment
- **Codebase tools** — Read, Glob, Grep for exploring the actual code

## Two-Tier Planning

**Lightweight plan (Simple/Medium):** Use `knowledge.json` + issue description + targeted Glob to identify files and write the plan. Read at most 1-2 source files.

**Deep plan (Complex):** Full codebase exploration. Read all affected files, trace dependencies, verify patterns.

The tier is determined by the triage comment's complexity classification.

## Workflow

1. **Fetch Issue & Triage Context** — verify status is `confirmed`, extract complexity from triage comment
2. **Checkout latest baseBranch** — `git fetch origin && git checkout <baseBranch> && git pull`
3. **Understand the Issue** — read title, description, acceptance criteria, attachments, relations
4. **Build the File Map** — read `.forge/knowledge.json`, use targeted Glob to confirm file paths
5. **Explore** — depth depends on tier (lightweight vs deep)
6. **Write the Plan** — `forge_issues → update → { plan: "<markdown>" }`
7. **Validate** — verify all files exist, steps cover all acceptance criteria
8. **Post Comment & Set Status** — Simple/Medium: auto-approve → `approved`; Complex: `waiting`
9. **Complex issues only** — after setting `waiting`, create a review issue via `forge_issues create`:
   - Title: `Review & Approve ISS-XX — {original title}`
   - Category: `review`, same priority, **status: `approved`** (skip triage/plan)
   - Relations: `depends_on <targetDocumentId>` + copy target's `blocked_by` relations
   - Description: reference target documentId and checklist from `forge-approve` skill
   - Plan: "Run /forge-approve <targetDocumentId> to review and approve. No code changes."
   This enables the `forge-approve` skill to auto-process the approval via normal pipeline.
   Setting `approved` directly lets Forge cascade to the code step immediately.

## Plan Format

```markdown
## Approach
<1-3 sentences: solution strategy>

## Affected Files
- `path/to/file.ts` — <what changes and why>

## Implementation Steps
1. <concrete action with file path> — <why>

## API Test Plan
<Only for backend changes — include curl examples>

## QA Scenarios
<Scenarios for forge-test: Setup → Action → Verify → Contrast>

## Risks
<Non-obvious risks — omit if none>
```

Read `references/plan-format.md` for full format details and `references/exploration-guide.md` for codebase exploration patterns.

## Output Rules (Save Tokens)

- **Zero narration.** Tool calls are self-documenting.
- **No quoting files.** After reading, don't repeat contents. Just write the plan.
- **Plan goes to the API, not to chat.** Write via `forge_issues → update`. Don't also print it.
