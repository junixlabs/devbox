# Authorization

## Middleware Execution Order

Strapi processes requests in this order:
1. **Authenticate** — JWT verified, CASL ability generated from role permissions
2. **Authorize** — checks `route.config.auth.scope` against ability
   - No scope defined → authenticated users pass, unauthenticated get 401
   - Scope defined → `ability.can(scope)` must return true
3. **Policies** — `is-super-admin`, `is-authenticated`, etc.
4. **Middlewares** — `tenant-context`, `feature-gate`, etc.
5. **Controller** — handler executes

The authorize middleware **catches all errors** from steps 2-4:
- `ForbiddenError` → converted to generic `ctx.forbidden()` (message lost)
- `PolicyError` → re-thrown with custom message preserved
- `UnauthorizedError` → converted to generic `ctx.unauthorized()`

## auth-policies — CRITICAL: PolicyError vs ForbiddenError

Strapi's authorize middleware **strips custom messages** from `ForbiddenError`.
Only `PolicyError` preserves custom messages in the API response.

```typescript
// WRONG - message becomes generic "Forbidden"
throw new errors.ForbiddenError('Super admin access required');

// CORRECT - message "Super admin access required" reaches the client
throw new errors.PolicyError('Super admin access required');
```

### Policy patterns

```typescript
// Throw PolicyError for custom denial messages
import { errors } from '@strapi/utils';

export default (policyContext) => {
  const user = policyContext.state.user;

  if (!user) {
    throw new errors.UnauthorizedError('Authentication required');
  }

  if (!user.isSuperAdmin) {
    throw new errors.PolicyError('Super admin access required');
  }

  return true;
};

// Return false for generic "Forbidden" (no custom message needed)
export default async (ctx, config, { strapi }) => {
  const user = ctx.state.user;
  if (!user) return false;

  const resource = await strapi.documents(config.uid).findOne({
    documentId: ctx.params.id,
    populate: ['tenant']
  });

  return resource?.tenant?.documentId === ctx.state.tenantId;
};
```

## auth-user

Prefer policies over inline auth checks. When inline checks are needed:

```typescript
async myAction(ctx) {
  const user = ctx.state.user;
  if (!user) return ctx.unauthorized();

  // For tenant-scoped actions, use withTenantContext instead
  // See ctrl-tenant-context.md
}
```

## Project policies

| Policy | Use for | Error |
|--------|---------|-------|
| `global::is-authenticated` | Any logged-in user | 401 |
| `global::is-super-admin` | Platform admin only | PolicyError 403 |
| `global::is-tenant-owner` | User belongs to tenant | false → 403 |
| `global::check-permission` | RBAC permission check | false → 403 |
