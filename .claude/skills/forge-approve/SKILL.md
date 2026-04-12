---
name: forge-approve
description: "Review and approve complex Forge issues at waiting status. Checks plan quality against decision framework, verifies blockers resolved, diagnoses stalled blockers and kicks off if needed. Auto-activates when forge-code processes review/approval issues. Triggers on: /forge-approve, approve issue, review waiting issue, Review & Approve."
user_invocable: true
arguments: "<documentId> — target issue documentId at waiting status"
---

# Forge Approve — Complex Issue Reviewer

Review complex issues at `waiting` status and decide: **approve**, **assist blockers**, or **escalate**.

## Activation

- Direct: `/forge-approve <documentId>` (target issue)
- Auto: when processing a "Review & Approve ISS-XX" issue via forge-code pipeline

## Process

1. **Resolve target issue** — direct invocation: `documentId` = target; review issue: extract from `depends_on` relations
2. **Fetch target issue + all comments** via `forge_issues` get + `forge_comments` list
3. **Verify plan exists** — check issue `plan` field is non-empty
4. **Check blockers** — fetch each `blocked_by` / `depends_on` issue, check status
5. **Evaluate against decision framework** (see `references/decision-framework.md`)

## Decision: Auto-approve

Set target status → `approved` when ALL true:
- Complexity is Medium or Lightweight (from triage comment)
- All blockers resolved (at `tested`/`pass`/`staging`/`released`/`closed`)
- Plan has: clear approach, affected files, steps, risks
- No architecture changes, no new external dependencies, no security concerns

**After approve:** comment summary → update status → review issue set `tested`

## Decision: Blocker Not Resolved

When blockers exist and are NOT complete, **actively diagnose and assist**:

1. **Fetch each blocker issue** — check status and recent comments
2. **Diagnose and act by status:**

| Blocker Status | Diagnosis | Action |
|----------------|-----------|--------|
| `open` | Not started — forgotten or just unblocked | Kick off: `forge_agent_sessions` → `/forge-triage <blockerDocId>` |
| `confirmed` | Triaged but no plan | Kick off: `/forge-plan <blockerDocId>` |
| `approved` | Planned but not coded | Kick off: `/forge-code <blockerDocId>` |
| `reopen` | Was rejected, needs fix | Kick off: `/forge-fix <blockerDocId>` |
| `in_progress` / `developed` / `testing` | Actively progressing | Wait — comment on blocker requesting notification |
| `on_hold` / `needs_info` | **Stalled** | **Escalate to owner** — report stall with details |
| `waiting` | Blocker itself needs approval | Recursive: run `/forge-approve <blockerDocId>` |

3. **Comment on blocker issue:** "ISS-{reviewId} is waiting for this issue to complete. Please update status when done."
4. **Set review issue to `on_hold`** with comment explaining which blocker and what action was taken
5. **Forge will re-trigger** when blocker status changes (via `blocked_by` relation)

## Decision: Escalate

Leave target at `waiting`, review issue at `needs_info` when ANY true:
- Plan changes architecture or adds new dependencies
- Issue scope grew beyond original description
- Security concerns detected
- Blocker is stalled (`on_hold`/`needs_info`) with no clear resolution path

**After escalate:** comment specific concerns on target + review issue

## Phase Gate Issues (NEVER auto-approve)

ISS-30, ISS-38, ISS-45, ISS-50, ISS-54 — create phase summary, recommend to owner, await explicit approval.

## Review Issue Format

Created by forge-plan at step 9 (Complex issues). Status: `approved` (skip triage/plan).
```
Title: "Review & Approve ISS-XX — {original title}"
Category: review | Priority: same as target | Status: approved
Relations: depends_on <target>, blocked_by <target's blockers>
```

## Tools

- **forge_issues** — get/update issues
- **forge_comments** — list/create comments
- **forge_agent_sessions** — kick off stalled/forgotten pipeline steps
