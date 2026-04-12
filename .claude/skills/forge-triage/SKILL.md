---
name: forge-triage
description: "Triage and validate Forge project management issues before development begins. Use this skill whenever issues need to be reviewed for completeness, classified by complexity, or assigned category/priority. Triggers on: /forge-triage, triaging issues, validating issue quality, classifying issue complexity, setting issue priority, reviewing new issues, checking if issues are actionable. Also use when the pipeline needs to move an issue from open to confirmed status. Even if the user just says 'triage this' or 'check if this issue is ready', use this skill."
user_invocable: true
arguments: "documentId1 documentId2 ..."
---

# Forge Triage

Triage gates the pipeline — it catches incomplete issues before they waste expensive planning and coding cycles.

Operate purely on issue data via MCP tools. Do not read the codebase — triage should be fast and cheap.

## Usage

```
/forge-triage <documentId>
/forge-triage <documentId1> <documentId2>
```

## Tools

- **forge_issues** — get/update issues
- **forge_comments** — list/create comments

## Workflow

### Step 1: Fetch Issue Data

```
forge_issues → get → { documentId: "<id>" }
forge_comments → list → { filters: { issue: "<documentId>" } }
```

### Step 2: Evaluate Completeness

Read `references/triage-criteria.md` for full criteria. Core question: **can a developer understand what to change and what the result should be?**

If incomplete → set `needs_info` with specific questions, then **stop**.

### Step 3: Classify Complexity

Read `references/complexity-rules.md`:
- **Simple** — single file/component, isolated change
- **Medium** — 2-5 files, single package
- **Complex** — cross-package, schema changes, new APIs

### Step 4: Set Category

If missing, infer: bug/feature/improvement/task. Only set if missing — preserve reporter's choice.

### Step 5: Set Priority

If `none`, infer: critical/high/medium/low. Only set if `none`.

### Step 6: Detect Related Issues

Search for 2-3 key terms from the issue. Link related issues with `relations` field.

### Step 7: Save Classifications, Post Comment & Set Status

Save fields first, post comment, then **set `confirmed` LAST** (triggers next pipeline step).

## Triage Comment Format

```markdown
**Triage** — <one-line summary>

**Complexity:** <Simple/Medium/Complex> — <brief justification>
**Category:** <category> <(inferred) if missing>
**Priority:** <priority> <(inferred) if none>
**Relations:** <list or "None detected">
```

**Always write triage comments in English**, regardless of issue language.
