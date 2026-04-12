# ctrl-response

Return objects directly — Strapi's `returnBodyMiddleware` sets `ctx.body` automatically.

```typescript
// Success (200)
return { data: result };
return { data: result, meta: buildPaginationMeta(total, page, pageSize) }; // see ctrl-pagination rule

// Errors
return ctx.badRequest('Invalid input');
return ctx.unauthorized();
return ctx.forbidden();
return ctx.notFound('Resource not found');

// Created (201)
ctx.status = 201;
return { data: result };

// No content (204)
ctx.status = 204;
ctx.body = null;
```

## Null check pattern

Always check `findOne` results before returning.

**Wrong:**
```typescript
async findOne(ctx) {
  const data = await strapi.documents(UID).findOne({ documentId: ctx.params.id });
  return { data }; // Returns { data: null } instead of 404
}
```

**Correct:**
```typescript
async findOne(ctx) {
  const data = await strapi.documents(UID).findOne({ documentId: ctx.params.id });
  if (!data) return ctx.notFound();
  return { data };
}
```
