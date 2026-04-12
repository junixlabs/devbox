---
name: forge-test
description: "QA test Forge issue changes against preview deployments. Use this skill to test like a human QA — hitting the preview backend API and navigating the preview frontend to verify acceptance criteria are met. Triggers on: /forge-test, testing an issue, QA testing, verifying changes on preview, checking if acceptance criteria pass. Also use when the pipeline needs to verify changes at testing status."
user_invocable: true
arguments: "documentId"
---

# Forge Test

Automated QA agent that tests the issue's actual output against the preview or staging deployment — like a human tester would. Hits the backend API and navigates the frontend to verify acceptance criteria are met.

This is NOT a test runner (vitest/playwright). It's a manual QA replacement that uses live URLs to verify the change works end-to-end.

## Usage

```
/forge-test <documentId>
```

## Tools

- **forge_issues** — get issue data (acceptance criteria, preview URLs, plan, changeHistory)
- **forge_comments** — list previous comments + post test report
- **forge_config** — get project config (staging URLs, test credentials)
- **forge_coolify_deploy** — check deployment status before testing
- **Browser** — `mcp__claude-in-chrome__*` for frontend testing
- **HTTP** — WebFetch / Bash (curl) for API testing

Read `references/test-approach.md` for testing patterns, `references/result-format.md` for report template, and `references/browser-playbook.md` for browser interaction guides.

## Workflow

1. **Fetch Issue + Pipeline Context** — read title, description, acceptanceCriteria, plan, previewUrl, comments
2. **Enter Worktree** — `EnterWorktree → { name: "ISS-XX-test" }` then `git fetch origin && git checkout ISS-XX-*` to isolate from parallel agent sessions
3. **Detect Reopen Cycle** — check `changeHistory` for prior `testing → reopen`. Extract regression tests from last QA report.
4. **Wait for Deployment Readiness** — `forge_coolify_deploy → status → {}`. Wait up to 3 min if building. For CLI projects (no preview URL), skip this step.
5. **Get Test URLs & Credentials** — `forge_config → get`. Priority: issue `previewUrl` → project `stagingUrl`. For CLI projects, test locally via build/test commands.
5. **Build Test Cases** — from Plan QA Scenarios → Acceptance Criteria → Review findings → Regression
6. **Test Backend API** — authenticate, curl/fetch endpoints, verify status codes + response shape
7. **Test Frontend UI** — navigate, login, follow user flows, check elements visible/correct
8. **Post Test Report** — `forge_comments → create` with table of results
9. **Set Status LAST** — all pass: `tested`; any fail: `reopen`
10. **Exit Worktree** — `ExitWorktree → { action: "remove" }` — testing is done, clean up
11. **Cascade: Unblock downstream issues** — after setting `tested`, search for issues that have `blocked_by` this issue. For each:
    - Fetch the blocked issue via `forge_issues get`
    - Check if ALL its blockers are now `tested`/`pass`/`staging`/`released`/`closed`
    - If fully unblocked AND status is `open`: start `forge_agent_sessions` → `/forge-triage <documentId>`
    - If fully unblocked AND status is `waiting` (has plan): start `forge_agent_sessions` → `/forge-approve <documentId>` (or `/forge-code` if not complex)
    - Comment on unblocked issue: "Blocker ISS-XX completed. Pipeline resumed."

## Report Format

```markdown
**QA Test Report**

**Test environment:** {testUrl} (preview|staging)

| # | Test Case | Source | Result | Notes |
|---|-----------|--------|--------|-------|
| 1 | Description | AC #1 | PASS/FAIL | Details |

**Summary:** X/Y passed
**Verdict:** PASS / FAIL
```

## Output Rules (Save Tokens)

- **Zero narration.** Don't announce each test case before running it.
- **Report goes to the comment, not to chat.**
- **One-line status only.** "QA done: 5/6 passed, 1 FAIL. Set reopen." — nothing more.
