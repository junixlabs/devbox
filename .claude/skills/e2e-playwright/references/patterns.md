# E2E Test Patterns

## Test Structure

```typescript
import { test, expect } from '../fixtures/test-users';
import { FeaturePage } from '../pages';

test.describe('[Feature] - [Actor] Workflow', () => {
    let featurePage: FeaturePage;

    test.beforeEach(async ({ page, hrAdminUser }) => {
        featurePage = new FeaturePage(page);
        await hrAdminUser.login(page);
        await featurePage.goto();
    });

    test('[STORY-ID]-AC-N: Description', async ({ page }) => {
        // GIVEN: Precondition
        // WHEN: Action
        // THEN: Assertion
    });
});
```

## Naming Convention

| Element | Format | Example |
|---------|--------|---------|
| Test file | `[feature].spec.ts` | `employees.spec.ts` |
| Test describe | `[Feature] - [Actor] Workflow` | `Employee Management - HR Admin Workflow` |
| Test name | `[STORY-ID]-AC-N: Description` | `EMP-001-AC-1: Should create employee` |

---

## Assertion Pattern (Error-First)

**CRITICAL**: Tests must FAIL when frontend has bugs. Never hide failures.

```typescript
// After form submit
await page.getByRole('button', { name: /save/i }).click();

await expect(async () => {
    // 1. Check ERRORS first - fail fast
    const hasError = await page.getByText(/error|failed/i).first().isVisible().catch(() => false);
    if (hasError) {
        const msg = await page.getByText(/error|failed/i).first().textContent();
        throw new Error(`Form failed: ${msg}`);
    }

    // 2. Check success (toast OR redirect)
    const hasSuccess = await page.getByText(/success|created|updated/i).first().isVisible().catch(() => false);
    const redirected = !page.url().includes('/new') && !page.url().includes('/edit');

    expect(hasSuccess || redirected).toBe(true);
}).toPass({ timeout: 10000 });
```

### Anti-Patterns

```typescript
// WRONG - hides bugs
try {
    await expect(element).toBeVisible();
} catch {
    test.skip();  // Bug hidden!
}

// WRONG - only checks toast (fails on silent redirect)
await expect(page.getByText(/success/i)).toBeVisible();

// WRONG - no verification after create
await page.getByRole('button', { name: /save/i }).click();
// Test ends - no proof it worked!
```

---

## Create-Verify Flow

Always create data via UI and verify it exists.

```typescript
test('Should create and verify employee', async ({ page, hrAdminUser }) => {
    const employee = createTestEmployee();
    await hrAdminUser.login(page);

    // CREATE via UI
    await page.goto('/employees/new');
    await page.getByLabel('First Name').fill(employee.firstName);
    await page.getByLabel('Last Name').fill(employee.lastName);
    await page.getByLabel('Work Email').fill(employee.email);
    await page.getByLabel('Phone').fill(employee.phone);  // Required!
    await page.getByRole('button', { name: /create/i }).click();

    // VERIFY success (error-first pattern)
    await expect(async () => {
        const hasError = await page.getByText(/error|failed/i).first().isVisible().catch(() => false);
        if (hasError) throw new Error('Creation failed');
        const hasSuccess = await page.getByText(/success/i).first().isVisible().catch(() => false);
        const redirected = !page.url().includes('/new');
        expect(hasSuccess || redirected).toBe(true);
    }).toPass({ timeout: 10000 });

    // VERIFY in list
    await page.goto('/employees');
    await expect(page.getByText(employee.firstName)).toBeVisible({ timeout: 5000 });
});
```

---

## Multi-Step Workflow

```typescript
test('Leave request approval workflow', async ({ page, employeeUser, hrAdminUser }) => {
    const leave = createTestLeaveRequest();

    // Step 1: Employee submits
    await employeeUser.login(page);
    await page.goto('/leave');
    await page.getByRole('button', { name: /apply/i }).click();
    await page.getByLabel('Leave Type').click();
    await page.getByRole('option', { name: /annual/i }).click();
    await page.getByLabel('Start Date').fill(leave.startDate);
    await page.getByLabel('End Date').fill(leave.endDate);
    await page.getByRole('button', { name: /submit/i }).click();
    await expect(page.getByText(/submitted|pending/i)).toBeVisible();

    // Step 2: HR approves
    await page.goto('/logout');
    await hrAdminUser.login(page);
    await page.goto('/leave/approvals');
    await page.getByRole('row').filter({ hasText: employeeUser.firstName })
        .getByRole('button', { name: /approve/i }).click();
    await expect(page.getByText(/approved/i)).toBeVisible();
});
```

---

## Data Factories

Location: `e2e/tests/data/factories.ts`

```typescript
import { createTestEmployee, createTestDepartment, createTestLeaveRequest } from '../data';

// All factories generate unique data with timestamp + random string
const employee = createTestEmployee({ firstName: 'John' });
// → { firstName: 'John', lastName: 'Employeeab12', email: 'john.1706547891234-x7k9m@test.local', phone: '+1234567890' }
```

| Factory | Required Fields |
|---------|----------------|
| `createTestEmployee()` | firstName, lastName, email, phone |
| `createTestDepartment()` | name |
| `createTestPosition()` | title |
| `createTestLeaveRequest()` | startDate, endDate, reason |
| `createTestLeaveType()` | name, code, defaultDays |

---

## Screenshots

```typescript
// Path relative to e2e/ working directory
await page.screenshot({ path: 'tests/screenshots/[test-name].png' });
```
