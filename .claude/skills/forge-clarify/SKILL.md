---
name: forge-clarify
description: "Clarify and validate Forge issues before planning — reproduce bugs via browser, verify UX expectations for features, capture evidence screenshots. Use this skill after triage (confirmed status) to ensure the issue is well-understood before writing an implementation plan. Triggers on: /forge-clarify, clarifying issues, reproducing bugs, validating UX, verifying issue understanding. Also use when the pipeline needs to move an issue from confirmed to clarified status."
user_invocable: true
arguments: "documentId"
---

# Forge Clarify

This is the step between triage and plan: `confirmed → clarified`. Its job is to validate understanding — reproduce bugs in a live environment, verify UX expectations for features, and capture visual evidence.

Simple issues are auto-skipped (the lifecycle hook advances them to `clarified` without running this skill).

## Usage

```
/forge-clarify <documentId>
```

## Tools

- **forge_issues** — get/update issues
- **forge_comments** — list/create comments
- **Browser** — `mcp__claude-in-chrome__*` tools for live environment testing
- **WebFetch** — for API endpoint testing

## Workflow

### Step 1: Fetch Issue & Triage Context

Find the triage comment (starts with `**Triage**`) and extract **complexity** and **category**.

### Step 2: Resolve Live Environment

Priority: issue `previewUrl` / `previewApiUrl` → project staging URLs → note if unavailable

### Step 3a: Bug Investigation

1. Read reproduction steps from description/attachments
2. Open the live URL in Chrome
3. Follow reported steps exactly, screenshot each step
4. Assess: Reproduced or Cannot reproduce

### Step 3b: Feature/Improvement Investigation

1. Navigate to the area being changed
2. Screenshot current state
3. Compare with mockups/designs in attachments
4. Identify existing UX patterns

### Step 4: Post Comment & Set Status

Upload screenshots, post clarify comment, then **set status LAST**:
- Clear → `clarified`
- Ambiguous → `needs_info`

## Comment Formats

**Bug — Reproduced:**
```markdown
**Clarify** — Reproduced: <one-line summary>
**Environment:** <URL tested>
**Reproduced:** Yes
**Steps Verified:** 1. <step> → ✅/❌ <observation>
**Root Cause Hypothesis:** <likely code-level issue>
```

**Feature — Clear:**
```markdown
**Clarify** — UX validated: <one-line summary>
**Current State:** <what exists>
**Desired State:** <what should change>
**Existing Patterns:** <similar UI patterns>
```

**Always write clarify comments in English.**

## Output Rules (Save Tokens)

- **Zero narration.** Do not say what you're about to do.
- **Screenshots are evidence.** Capture, upload, attach to comment.
- **One-line status only.** "Clarified, setting status." — nothing more.
