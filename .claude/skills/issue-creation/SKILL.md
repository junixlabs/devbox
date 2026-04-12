---
target: cloud
---
# Issue Creation Guidelines

When creating issues, follow these standards to ensure quality and actionability.

## Confirmation Rule
IMPORTANT: Always present the full issue draft in markdown format and ask the user to confirm BEFORE calling forge_issues to create it. Only create after explicit user approval (e.g. "yes", "go ahead", "create it", "looks good"). If the user requests changes, revise and present again.

## Tool Call Format
When creating, call forge_issues with exactly this structure:
```json
{
  "action": "create",
  "data": {
    "title": "Verb-first, specific, max 80 chars",
    "description": "Markdown context — why needed, what triggered it.\n\nFor bugs:\n- **Expected:** what should happen\n- **Actual:** what happens instead",
    "category": "bug | feature | improvement | task",
    "priority": "critical | high | medium | low",
    "acceptanceCriteria": "- [ ] Condition one\n- [ ] Condition two\n- [ ] Condition three",
    "suggestedSolution": "### Approach\nDescribe the UX/UI approach and user-facing behavior changes. Do NOT reference specific code files or implementation details — the agent solving the issue will determine those.",
    "plan": "### Steps\n1. Step one\n2. Step two\n\n(omit for simple issues)"
  }
}
```

Rules:
- ALL fields go inside `data` object — not at top level
- ALL content fields use **markdown** formatting (headings, lists, code blocks, bold, etc.)
- `description` = context ONLY. Do NOT put acceptance criteria or solution here
- `acceptanceCriteria` = markdown checklist (`- [ ] ...`), one condition per line
- `suggestedSolution` = UX/UI approach and user-facing behavior only. Do NOT mention specific code files, libraries, or implementation details
- `plan` = markdown with numbered steps (optional, for complex issues only)
- `status` is auto-set to "open" — do not include it

## Draft Format
Present draft before creating:

---
**Title:** {title}
**Category:** {category} | **Priority:** {priority}

**Description:**
{description}

**Acceptance Criteria:**
{acceptanceCriteria as bullet list}

**Suggested Solution:**
{suggestedSolution, or "N/A"}

---
Shall I create this issue?

## Priority Guidelines
- **critical** — Production broken, data loss, security vulnerability
- **high** — Major feature blocked, significant UX degradation
- **medium** — Important but not urgent, planned improvements
- **low** — Nice-to-have, minor polish, tech debt

## Category Guidelines
- **bug** — Something broken that worked before
- **feature** — New capability that doesn't exist yet
- **improvement** — Enhancement to existing functionality
- **task** — Technical work, refactoring, infrastructure

## Quality Checklist
Before creating, verify:
- Title is actionable and specific
- Description has context (not acceptance criteria — that goes in its own field)
- acceptanceCriteria has measurable conditions
- Not a duplicate — search existing issues first using forge_issues list
- Scope is single-deliverable — split large work into multiple issues
