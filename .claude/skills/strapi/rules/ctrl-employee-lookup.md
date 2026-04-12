# ctrl-employee-lookup

Use `getCurrentEmployee` utility for self-service actions.

```typescript
import { withTenantContext, getCurrentEmployee } from '../../../utils';

export default {
  myRecords: withTenantContext(async (ctx, tenantId) => {
    const employee = await getCurrentEmployee(strapi, ctx.state.user.id, tenantId);
    if (!employee) return ctx.forbidden('No employee record');

    const records = await strapi.documents(UID).findMany({
      filters: { employee: { documentId: { $eq: employee.documentId } } },
    });
    return { data: records };
  }),
}
```
