# Triage Criteria

The core question: **can a developer (or coding agent) understand what to change and what the result should be?**

## All Issues Must Have

1. **Clear scope** — what area of the system is affected (feature, page, API, component)
2. **Expected outcome** — what should be true after the change
3. **Enough context** — sufficient detail to start without guessing

## Bug-Specific Requirements

Bugs need at least ONE of:
- Steps to reproduce
- Observable symptom (error message, screenshot, wrong behavior)
- Specific conditions (browser, device, data state)

"Login is broken" → `needs_info` (which login? what happens? when?)
"Login returns 500 when email contains a + character" → actionable

## Feature-Specific Requirements

Features need at least ONE of:
- User-facing description of the desired outcome
- Acceptance criteria that define "done"
- Suggested solution detailed enough to derive expected behavior

"Improve the dashboard" → `needs_info` (improve how? which metrics? for whom?)
"Add a filter dropdown to the issues list that filters by priority" → actionable

## When to Confirm vs Needs Info

**Confirm** when:
- You can roughly describe what areas would need to change
- Expected behavior is clear enough to verify when done
- Acceptance criteria exist (author-written OR AI-generated)

**Needs Info** when:
- Scope is too broad to plan ("improve performance")
- Multiple valid interpretations exist — choosing wrong wastes effort
- Cannot determine what "done" looks like
- Missing critical context (which page? which endpoint? which user role?)

## Edge Cases

- **AI-generated criteria** (`aiAcceptanceCriteria`, `aiSuggestedSolution`) count as sufficient detail, even if the human description is vague.
- **Plan already populated** — if the `plan` field has content, the issue is past triage. Confirm immediately.
- **Short but clear** — "Fix typo: 'Submited' → 'Submitted' on settings page" is one line but fully actionable. Length ≠ quality.
- **References external context** — if the issue mentions a Slack thread without including the content, ask for the relevant details to be inlined.
