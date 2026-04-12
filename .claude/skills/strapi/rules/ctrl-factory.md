# ctrl-factory

Use `factories.createCoreController` to get Strapi's built-in sanitization and response handling.

```typescript
import { factories } from '@strapi/strapi';
import type { Context } from 'koa';

const UID = 'api::employee.employee' as const;

export default factories.createCoreController(UID, ({ strapi }) => ({
  // Override find with pagination (see ctrl-pagination rule)
  async find(ctx: Context) {
    const tenantId = ctx.state.tenantId;
    const { page, pageSize, limit, offset } = parsePagination(ctx.query);
    const filters = withTenantFilter(ctx.query.filters, tenantId);

    const [data, total] = await Promise.all([
      strapi.documents(UID).findMany({
        filters,
        populate: ['department', 'position'],
        limit,
        offset,
      }),
      strapi.documents(UID).count({ filters }),
    ]);

    return { data, meta: buildPaginationMeta(total, page, pageSize) };
  },

  // Custom action
  async hierarchy(ctx: Context) {
    const { id } = ctx.params;

    const employee = await strapi.documents(UID).findOne({
      documentId: id,
      populate: ['manager', 'directReports'],
    });

    if (!employee) return ctx.notFound();

    return { data: employee };
  },
}));
```

> **Note**: Return `{ data }` directly — Strapi's `returnBodyMiddleware` sets `ctx.body` automatically.
