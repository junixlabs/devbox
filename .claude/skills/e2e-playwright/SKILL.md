---
name: e2e-playwright
description: |
  E2E testing with Playwright. Use for: creating tests from user stories,
  fixing failing tests, reviewing test coverage. Reports bugs to /docs/bugs
  and integrates with nextjs skill for fixes.
---

# E2E Playwright Testing

## Quick Reference

| Item | Location |
|------|----------|
| Tests | `e2e/tests/[NN-module]/*.spec.ts` |
| Page objects | `e2e/tests/pages/*.page.ts` |
| Data factories | `e2e/tests/data/factories.ts` |
| Bug reports | `docs/bugs/` |

## Running Tests

**Always use `test:reuse` to reuse existing test users (faster):**

```bash
cd e2e
npm run test:reuse                      # Run all tests
npm run test:reuse -- -g "EMP-001"      # Single test by name
npm run test:reuse -- --project=04-employees  # Single module
```

## Bug Reporting Workflow

When E2E tests find frontend bugs:

1. Write to `docs/bugs/FE-BUG-[DATE].md` with test, location, severity, issue, expected, actual, suggested fix
2. Invoke nextjs skill to fix: `Use /nextjs skill to fix the bug documented in docs/bugs/FE-BUG-[DATE].md`

## Core Rules

1. **Frontend-First** — Read frontend code before writing tests
2. **Error-First Assertions** — Check errors BEFORE success. Never hide failures with try/catch
3. **Verify After Create** — Confirm created data appears in UI
4. **Report Bugs** — When tests fail due to frontend issues, create bug report in `docs/bugs/`

## Common Selectors

| Component | Selector |
|-----------|----------|
| Input | `getByLabel('Label')` |
| Button | `getByRole('button', { name: /text/i })` |
| Custom Select | `getByLabel('Label').click()` → `getByRole('option', { name })` |
| Modal | `getByRole('dialog')` |
| Table row | `getByRole('row').filter({ hasText })` |

## Quick Fixes

| Issue | Fix |
|-------|-----|
| Strict mode | Add `.first()` |
| Custom Select | Use `click()` + `getByRole('option')` |
| Silent redirect | Accept toast OR redirect |
| Test passes but broken | Remove try/catch |

## References

- `references/patterns.md` — Test structure, assertions, flows
- `references/selectors.md` — UI selectors, page objects
- `references/troubleshooting.md` — Issues and debugging
- `scripts/coverage_check.py` — E2E coverage analysis
- `scripts/generate_tests.py` — Generate tests from user stories
- `scripts/report_bug.py` — Generate bug reports from failures
