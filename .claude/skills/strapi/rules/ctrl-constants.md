# ctrl-constants

Use centralized constants instead of magic numbers.

```typescript
import { PAGINATION, PAYROLL, LEAVE_STATUS } from '../../../config/constants';
import { EMPLOYEE_POPULATE } from '../../../config/populate';

// Pagination
const pageSize = Number(ctx.query.pageSize) || PAGINATION.DEFAULT_PAGE_SIZE;

// Status checks
if (request.status === LEAVE_STATUS.APPROVED) { ... }

// Payroll calculations
const pfAmount = grossSalary * PAYROLL.DEFAULT_PF_RATE;

// Populate arrays
const employees = await strapi.documents(UID).findMany({
  populate: EMPLOYEE_POPULATE,
});
```

## Date Utilities

```typescript
import { getTodayISO, hasDateOverlap, daysBetween } from '../../../utils';

const today = getTodayISO(); // '2026-01-25'

if (hasDateOverlap(start1, end1, start2, end2)) {
  return ctx.badRequest('Date range overlaps');
}

const leaveDays = daysBetween(startDate, endDate);
```
