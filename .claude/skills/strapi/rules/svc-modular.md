# svc-modular

Split large services (>200 lines) into domain-specific modules with facade pattern.

**Incorrect:**
```typescript
// services/report.ts - 723 lines with all reports inline
export default ({ strapi }) => ({
  getHeadcountReport: async (tenantId, filters) => { /* 100 lines */ },
  getTurnoverReport: async (tenantId, filters) => { /* 80 lines */ },
  getAttendanceSummary: async (tenantId, filters) => { /* 150 lines */ },
});
```

**Correct:**
```typescript
// services/report.ts - Facade (~50 lines)
import { getHeadcountReport, getTurnoverReport } from './employee-reports';
import { getAttendanceSummary } from './attendance-reports';

export default ({ strapi }) => ({
  getHeadcountReport: getHeadcountReport(strapi),
  getTurnoverReport: getTurnoverReport(strapi),
  getAttendanceSummary: getAttendanceSummary(strapi),
});
```

**Structure:**
```
services/
├── report.ts             # Facade composing modules
├── types.ts              # Shared interfaces
├── employee-reports.ts   # Domain-specific functions
├── attendance-reports.ts
└── leave-reports.ts
```
