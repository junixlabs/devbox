---
name: tester
description: |
  Unified test runner for backend and frontend. Run tests, check coverage,
  and generate test files. Auto-detects target based on context or use
  explicit flags: be (backend), fe (frontend), all (both).
---

# Unified Test Runner

Run tests, check coverage for both backend (Strapi/Jest) and frontend (Playwright E2E).

## Quick Start

```bash
# Auto-detect and run tests
python3 .claude/skills/tester/scripts/runner.py run

# Explicit target
python3 .claude/skills/tester/scripts/runner.py run --target be    # Backend only
python3 .claude/skills/tester/scripts/runner.py run --target fe    # Frontend E2E only
python3 .claude/skills/tester/scripts/runner.py run --target all   # Both
```

## Run Tests

### Backend Tests (Jest)

```bash
# Run all backend tests
python3 .claude/skills/tester/scripts/runner.py run --target be

# Run specific test file
python3 .claude/skills/tester/scripts/runner.py run --target be --file auth

# Run with coverage
python3 .claude/skills/tester/scripts/runner.py run --target be --coverage

# Watch mode
python3 .claude/skills/tester/scripts/runner.py run --target be --watch
```

### Frontend E2E Tests (Playwright)

```bash
# Run all E2E tests
python3 .claude/skills/tester/scripts/runner.py run --target fe

# Run specific test file
python3 .claude/skills/tester/scripts/runner.py run --target fe --file login

# Run with visible browser
python3 .claude/skills/tester/scripts/runner.py run --target fe --headed

# Debug mode
python3 .claude/skills/tester/scripts/runner.py run --target fe --debug

# Playwright UI mode
python3 .claude/skills/tester/scripts/runner.py run --target fe --ui

# View HTML report
python3 .claude/skills/tester/scripts/runner.py report
```

### Bug Log & Failure Reports

When tests fail, detailed logs are saved to `.tmp/` folder for easy debugging:

```bash
# View quick failure summary (recommended)
python3 .claude/skills/tester/scripts/runner.py bugs --summary

# View full bug log with error traces
python3 .claude/skills/tester/scripts/runner.py bugs

# Get path to JSON failures (for programmatic access)
python3 .claude/skills/tester/scripts/runner.py failures-json

# Clear all bug logs
python3 .claude/skills/tester/scripts/runner.py clear
```

**Output files (in `.tmp/`):**
| File | Description |
|------|-------------|
| `test-failures-summary.txt` | Quick summary - file, test name, key error |
| `test-failures.json` | Detailed JSON for programmatic access |
| `test-bugs.log` | Full log with complete error traces |

## Check Coverage

Find untested endpoints/pages:

```bash
# Auto-detect
python3 .claude/skills/tester/scripts/coverage.py

# Backend API coverage
python3 .claude/skills/tester/scripts/coverage.py --target be

# Frontend E2E page coverage
python3 .claude/skills/tester/scripts/coverage.py --target fe

# Show all (covered + uncovered)
python3 .claude/skills/tester/scripts/coverage.py --all

# Output as JSON
python3 .claude/skills/tester/scripts/coverage.py --json
```

## Generate Tests

### Backend (Smart Generator)

Reads schema.json and routes to generate comprehensive tests:

```bash
# Generate smart tests for a resource
python3 .claude/skills/tester/scripts/generate_smart.py training-program

# Preview without writing
python3 .claude/skills/tester/scripts/generate_smart.py leave-request --dry-run

# Generate for all resources
python3 .claude/skills/tester/scripts/generate_smart.py --all --force
```

## Options Summary

| Script | Option | Description |
|--------|--------|-------------|
| runner.py | `--target be\|fe\|all` | Target platform |
| runner.py | `--file NAME` | Run only tests matching NAME |
| runner.py | `--coverage` | Generate coverage (BE only) |
| runner.py | `--watch` | Watch mode (BE only) |
| runner.py | `--headed` | Run with visible browser (FE) |
| runner.py | `--debug` | Debug mode (FE) |
| runner.py | `--ui` | Playwright UI mode (FE) |
| coverage.py | `--target be\|fe` | Target platform |
| coverage.py | `--all` | Show all endpoints/pages |
| coverage.py | `--json` | Output as JSON |

## Test Structure

```
backend/tests/
├── api/                      # API integration tests
│   ├── helpers/              # Test utilities
│   ├── auth.test.ts
│   ├── employee.test.ts
│   └── ...
└── setup.ts                  # Test setup

e2e/                          # Playwright E2E tests
├── tests/
│   ├── login.spec.ts
│   ├── dashboard.spec.ts
│   ├── employees.spec.ts
│   └── ...
├── playwright.config.ts
└── package.json
```

## Backend Testing Rules

### HTTP Status Codes

#### 401 Unauthorized
Use when authentication fails:
```typescript
expect(response.status).toBe(401);
```

#### 403 Forbidden - MUST Verify Source

**CRITICAL**: Always verify the source of 403 responses based on JWT presence:

**WITHOUT JWT (`apiRequest()`)** - Strapi permission layer catches first:
```typescript
// No JWT → Strapi's permission layer returns "Forbidden"
const response = await apiRequest().get('/api/some-endpoint');
expect(response.status).toBe(403);
expect(response.body.error?.message).toBe('Forbidden');
```

**WITH JWT (`authRequest(jwt)`)** - Policy/controller layer catches:
```typescript
// With JWT → Request passes Strapi layer, reaches policy/controller
const response = await authRequest(jwt).get('/api/some-endpoint');
expect(response.status).toBe(403);
// Message depends on what rejected it:
```

Message sources when JWT is present:

1. **`is-super-admin` policy**:
   ```typescript
   expect(response.body.error?.message).toBe('Super admin access required');
   ```

2. **Controller authorization** (cross-tenant/permission check):
   ```typescript
   expect(response.body.error?.message).toBe('Access denied to this resource');
   ```

3. **Feature gate** (plan-based access):
   ```typescript
   expect(response.body.error?.message).toContain('Feature not available');
   ```

4. **Controller calls `ctx.forbidden()` without message**:
   ```typescript
   expect(response.body.error?.message).toBe('Forbidden');
   ```

**Validation Script**: Check 403 message consistency:
```bash
python3 .claude/skills/tester/scripts/check_403.py
python3 .claude/skills/tester/scripts/check_403.py --fix  # Show suggested fixes
```

## Frontend E2E Testing Rules

### Playwright Test Structure

```typescript
import { test, expect } from '@playwright/test';

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login form', async ({ page }) => {
    await expect(page.getByLabel(/Email/i)).toBeVisible();
    await expect(page.getByLabel(/Password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /Sign in/i })).toBeVisible();
  });

  test('should login successfully', async ({ page }) => {
    await page.getByLabel(/Email/i).fill('test@example.com');
    await page.getByLabel(/Password/i).fill('password123');
    await page.getByRole('button', { name: /Sign in/i }).click();

    await expect(page).toHaveURL(/dashboard/);
  });
});
```

### Critical Rules (Lessons Learned)

#### 1. Multiple Elements with Same Text → Scope Locators

When text appears multiple times (e.g., in heading AND step indicator):

```typescript
// WRONG - "strict mode violation: resolved to 2 elements"
await expect(page.getByText('Choose Plan')).toBeVisible();

// CORRECT - Scope to specific element type
await expect(page.getByRole('heading', { name: 'Choose Plan' })).toBeVisible();
// OR for step indicators
await expect(page.locator('span').filter({ hasText: 'Choose Plan' })).toBeVisible();
```

#### 2. Duplicate Links → Scope to Page Section

When same link appears in navigation AND content:

```typescript
// WRONG - "strict mode violation: resolved to 2 elements"
await page.getByRole('link', { name: /Start free trial/i }).click();

// CORRECT - Scope to main content area
await page.getByRole('main').getByRole('link', { name: /Start free trial/i }).click();
// OR scope to navigation
await page.getByRole('navigation').getByRole('link', { name: /Start free trial/i }).click();
```

#### 3. Custom Checkbox/UI Components → Click Label Wrapper

Custom checkboxes often have hidden inputs with visual overlays that intercept clicks:

```typescript
// WRONG - "element intercepts pointer events"
await page.getByRole('checkbox', { name: /Terms of Service/i }).check();

// CORRECT - Click the label wrapper
await page.locator('label').filter({ hasText: /Terms of Service/i }).click();
```

#### 4. Dynamic Content Loading → Wait for Loading State

Always wait for loading states to complete before interacting:

```typescript
// WRONG - May interact before data loads
await page.getByRole('button', { name: /Starter/i }).click();

// CORRECT - Wait for loading to finish first
await expect(page.getByText('Loading plans...')).toBeHidden({ timeout: 10000 });
await page.getByRole('button', { name: /Starter/i }).click();
```

#### 5. Credential-Dependent Tests → Skip Gracefully

Tests requiring login credentials should skip when not configured:

```typescript
test('should login and access dashboard', async ({ page }) => {
  const testEmail = process.env.TEST_USER_EMAIL;
  const testPassword = process.env.TEST_USER_PASSWORD;

  // Skip if no credentials configured
  test.skip(!testEmail || !testPassword,
    'Requires TEST_USER_EMAIL and TEST_USER_PASSWORD environment variables');

  await page.getByLabel(/Email/i).fill(testEmail!);
  // ... rest of test
});
```

#### 6. Form Validation → Don't Rely on Custom Error Messages

HTML5 form validation may prevent submission before custom errors show:

```typescript
// WRONG - Custom error may not appear if HTML5 validation blocks
await expect(page.getByText('Please enter your email')).toBeVisible();

// CORRECT - Check form state or required attributes
const emailInput = page.getByLabel(/Email/i);
const hasRequiredAttr = await emailInput.getAttribute('required');
expect(hasRequiredAttr).not.toBeNull();
```

### Best Practices

1. **Use semantic locators with regex for flexibility**:
   ```typescript
   // Good - case insensitive, partial match
   page.getByRole('button', { name: /Sign in/i })
   page.getByLabel(/Email/i)

   // Avoid - brittle exact match
   page.getByRole('button', { name: 'Sign In' })
   ```

2. **Wait for navigation with URL pattern**:
   ```typescript
   await page.getByRole('button', { name: /Submit/i }).click();
   await expect(page).toHaveURL(/dashboard|first-login/, { timeout: 15000 });
   ```

3. **Take screenshots at key steps**:
   ```typescript
   await page.screenshot({ path: 'tests/screenshots/step-2-company.png' });
   ```

4. **Use timeouts for async operations**:
   ```typescript
   await expect(page.getByText(/is available/i)).toBeVisible({ timeout: 5000 });
   ```

## Requirements

### Backend
- Strapi server running: `python3 .claude/skills/strapi-server/scripts/server.py start`
- Dependencies: `cd backend && npm install`

### Frontend E2E
- Dependencies auto-installed on first run
- Playwright browsers: `npx playwright install chromium`
