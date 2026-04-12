# api-permissions

Strapi 5 uses two layers for API access:

1. **Users-permissions plugin** — JWT auth + CASL ability from role permissions
2. **Route policies** — custom authorization (e.g., `is-super-admin`, `check-permission`)

Both layers must allow access for a request to succeed.

> For middleware execution order and PolicyError details, see `authorization.md`.

## Seeding API Permissions

Custom API actions must be registered in the bootstrap permissions seed:

**File**: `src/bootstrap/seeds/api-permissions.ts`

```typescript
const apiPermissions = [
  { controller: 'import', actions: ['getTypes', 'getTemplate', 'validate', 'execute'] },
  { controller: 'report', actions: ['headcount', 'turnover', 'dashboardMetrics'] },
];
```

## Adding New API Endpoints

When adding a new controller action:

1. Create the route in `routes/`
2. Create the handler in `controllers/`
3. **Add the action to `api-permissions.ts`**

## Common 403 Errors

If authenticated requests return 403:

1. Check `api-permissions.ts` includes the action
2. Restart server to trigger bootstrap
3. Verify permission was created in database

If policy custom messages return as generic "Forbidden":

1. Use `errors.PolicyError` instead of `errors.ForbiddenError` in policies
2. See `authorization.md` → auth-policies section

## When to Use `auth: false`

Only for truly public endpoints (no authentication needed):

```typescript
// Public endpoints - no auth required
// In routes: config: { auth: false }

// Authenticated endpoints - use policy + seed permission
// In routes: config: { policies: ['global::is-authenticated'] }
```
