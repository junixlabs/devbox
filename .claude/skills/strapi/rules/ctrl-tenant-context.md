# ctrl-tenant-context

**REQUIRED**: Use `withTenantContext` wrapper for tenant-scoped actions.

```typescript
import { withTenantContext, buildTenantFilter } from '../../../utils';

const UID = 'api::item.item' as const;

export default {
  find: withTenantContext(async (ctx, tenantId) => {
    const items = await strapi.documents(UID).findMany({
      filters: buildTenantFilter(tenantId),
      populate: ['tenant'],
    });
    return { data: items };
  }),

  create: withTenantContext(async (ctx, tenantId) => {
    const { data } = ctx.request.body as { data: Record<string, unknown> };
    const item = await strapi.documents(UID).create({
      data: { ...data, tenant: tenantId },
    });
    return { data: item };
  }),
}
```
