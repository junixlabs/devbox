# Troubleshooting E2E Tests

## Quick Fixes

| Issue | Fix |
|-------|-----|
| Selector not found | Read frontend code, verify element exists |
| Strict mode violation | Add `.first()` or scope to container |
| Custom Select fails | Use `click()` + `getByRole('option')`, not `selectOption()` |
| Modal doesn't close | Check for validation errors, increase timeout |
| Test passes but FE broken | Remove try/catch, use error-first assertions |
| Silent redirect | Accept toast OR redirect as success |
| Missing required fields | Read form component, update factory |

---

## Detailed Solutions

### 1. Strict Mode Violation

**Error**: `strict mode violation: locator resolved to N elements`

```typescript
// Fix: Add .first()
get addButton() {
    return this.page.getByRole('button', { name: /add/i }).first();
}

// Or: Scope to container
const headerButton = page.locator('header').getByRole('button', { name: /add/i });
```

### 2. Modal Doesn't Close After Submit

**Cause**: Backend validation error keeps modal open.

```typescript
await modal.getByRole('button', { name: /create/i }).click();

// Check if modal closed or has error
await expect(async () => {
    const hasError = await modal.getByText(/error|failed/i).first().isVisible().catch(() => false);
    if (hasError) {
        const msg = await modal.getByText(/error|failed/i).first().textContent();
        throw new Error(`Modal error: ${msg}`);
    }
    const closed = await modal.isHidden().catch(() => false);
    expect(closed).toBe(true);
}).toPass({ timeout: 10000 });
```

### 3. Login Conflicts Between Tests

**Cause**: `beforeEach` logs in one user, but test needs another.

```typescript
// DON'T auto-login if tests need different users
test.beforeEach(async ({ page }) => {
    employeesPage = new EmployeesPage(page);
    // Don't login here
});

// Login in each test
test('employee view', async ({ page, employeeUser }) => {
    await employeeUser.login(page);
});

test('admin view', async ({ page, hrAdminUser }) => {
    await hrAdminUser.login(page);
});
```

### 4. Timing/Flaky Tests

```typescript
// Wait for network after navigation
await page.waitForLoadState('networkidle');

// Wait for element, not fixed timeout
await expect(element).toBeVisible({ timeout: 5000 });

// Debounce for search
await searchInput.fill('query');
await page.waitForTimeout(500);
```

### 5. Table Row Not Found After Create

```typescript
// Wait for table to refresh
await page.waitForLoadState('networkidle');

// Or wait for row to appear
await expect(
    page.getByRole('row').filter({ hasText: 'New Entity' })
).toBeVisible({ timeout: 5000 });
```

### 6. Form Validation Errors

**Cause**: Missing required fields.

```typescript
// Check screenshot for red validation messages
// Identify missing required fields (*)
// Update test to fill ALL required fields

await page.getByLabel('Name').fill(data.name);
await page.getByLabel('Code').fill(data.code);       // Required!
await page.getByLabel('Phone').fill(data.phone);     // Required!
```

### 7. Screenshot Path Wrong

```typescript
// WRONG - path includes e2e/ prefix
await page.screenshot({ path: 'e2e/tests/screenshots/test.png' });

// CORRECT - relative to e2e/ working directory
await page.screenshot({ path: 'tests/screenshots/test.png' });
```

---

## Debugging Commands

```bash
# Run all tests (reuse test users - faster)
npm run test:reuse

# Run single test by name
npm run test:reuse -- -g "EMP-001-AC-1"

# Run single module
npm run test:reuse -- --project=04-employees

# Run with debugger
npm run test:debug

# View trace after failure
npx playwright show-trace test-results/[folder]/trace.zip

# Only use test:clean when users are corrupted
npm run test:clean
```
