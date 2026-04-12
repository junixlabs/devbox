# ctrl-pagination

All list (`find`) endpoints MUST use `parsePagination` + `buildPaginationMeta` from `utils/controller`.

## Required pattern

```typescript
import { parsePagination, buildPaginationMeta, withTenantFilter } from '../../../utils/controller';

async find(ctx) {
  const tenantId = ctx.state.tenantId;
  const { filters, sort } = ctx.query;
  const { page, pageSize, limit, offset } = parsePagination(ctx.query);

  const tenantFilters = withTenantFilter(filters, tenantId);

  const [data, total] = await Promise.all([
    strapi.documents(UID).findMany({
      filters: tenantFilters,
      populate: POPULATE,
      sort: sort || DEFAULT_SORT,
      limit,
      offset,
    }),
    strapi.documents(UID).count({ filters: tenantFilters }),
  ]);

  return { data, meta: buildPaginationMeta(total, page, pageSize) };
}
```

## Rules

1. **Never hardcode** pagination values — always import `PAGINATION` from `config/constants`
2. **Never spread `ctx.query`** into `findMany()` — it contains `page`/`pageSize` which Strapi ignores; use `parsePagination` to convert to `limit`/`offset`
3. **Always return full meta** — `{ page, pageSize, pageCount, total }` via `buildPaginationMeta`
4. **Always use `Promise.all`** for data + count queries (parallel fetch)
5. **Factory controllers** (`createApprovalController`, `createCrudController`) must also use `parsePagination` + `buildPaginationMeta` in their `find` methods

## Wrong

```typescript
// Spreading ctx.query — page/pageSize silently ignored by findMany
const data = await strapi.documents(UID).findMany({ ...ctx.query, filters });
return { data, meta: { pagination: { total } } }; // incomplete meta

// Hardcoded values
const pageSize = Math.min(Number(ctx.query.pageSize) || 25, 100);
const offset = (page - 1) * pageSize;
```

## Correct

```typescript
const { page, pageSize, limit, offset } = parsePagination(ctx.query);
// ...
return { data, meta: buildPaginationMeta(total, page, pageSize) };
```
