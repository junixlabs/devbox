# Plan Format

The plan field is the primary artifact of this skill. It's read by forge-code (follows it step-by-step), humans (review for complex issues), and forge-review (compares diff against plan).

## Template

```markdown
## Approach
<1-3 sentences: what's the solution strategy and why this approach over alternatives>

## Affected Files
- `path/to/file.ts` — <what changes and why>
- `path/to/other.ts` — <what changes and why>

## Implementation Steps
1. <concrete action with file path> — <why this step>
2. <concrete action with file path> — <why this step>
...

## API Test Plan
<Only include if the change affects backend API endpoints. Omit entirely for frontend-only changes.>

1. **<Test name>**
   - `<METHOD> <path>` — <body/params if any>
   - Expected: <status code> + <key response fields>
   - Example: `curl -X POST http://localhost:1337/api/... -H 'Authorization: Bearer $TOKEN' -d '...'`

## QA Scenarios
<Test scenarios for forge-test: Setup → Action → Verify → Contrast>

1. **<Scenario name>** (AC #N)
   - Setup: <preconditions — login role, data needed>
   - Action: <what to do — navigate, click, call API>
   - Verify: <expected result>
   - Contrast: <opposite case that proves the logic>

## Risks
- <non-obvious risks or edge cases> (omit section entirely if none)
```

## Writing Good Plans

**Approach section** — explain the *why*, not just the *what*. If you considered alternatives, briefly note why you chose this one.

**Affected Files** — list every file that needs modification. Missing a file means the coding agent won't know to touch it. Include files for types, tests, and re-exports.

**Implementation Steps** — each step should be one atomic change. Include file path, what to change, why, and pattern to follow.

**API Test Plan** — only include for changes that touch backend API endpoints. Each test should specify the endpoint, method, request body, **and the expected response**. Include a curl example. **Omit entirely for frontend-only changes.**

**QA Scenarios** — consumed by `forge-test`. Each scenario: Setup (role, time override, data state), Action (concrete user steps), Verify (expected observable result), Contrast (opposite case that proves the logic).

**Risks** — only include if there are genuine concerns.

## Anti-Patterns

- **Vague steps** — "Update the component" tells the coding agent nothing
- **Missing file paths** — "Add a new hook" without specifying where
- **Over-planning** — don't specify exact variable names or line numbers
- **Copy-pasting the issue** — the plan should add value beyond the description
