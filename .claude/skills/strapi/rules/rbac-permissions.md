# rbac-permissions

Role-Based Access Control (RBAC) permissions for tenant users. These are different from API permissions (see `api-permissions.md`).

## Architecture

```
Permission (database)
    ↓
Platform Role Template (Super Admin manages)
    ↓ sync
Custom Role (tenant-scoped, isSystemRole=true for defaults)
    ↓
User (assigned roles)
```

## Adding New Permissions

**File**: `src/bootstrap/seeds/permissions.ts`

```typescript
export const permissionsData: PermissionData[] = [
  // Add new permission
  {
    module: 'myfeature',      // Module category
    action: 'create',         // create|read|update|delete|approve|export|manage
    code: 'myfeature.create', // Unique code (module.action)
    name: 'Create my feature',
    requiredPlan: 'starter'   // starter|professional|enterprise
  },
];
```

After adding:
1. Restart server → permission auto-seeded
2. Super Admin assigns to role templates via UI
3. Super Admin syncs to tenants

## Managing Role Permissions (Super Admin)

Permissions are managed via UI, not code. No bootstrap sync needed.

### Endpoints

| Action | Endpoint | Description |
|--------|----------|-------------|
| List templates | `GET /platform-role-templates` | All templates with permissions |
| View template | `GET /platform-role-templates/:id` | Single template |
| Update template | `PUT /platform-role-templates/:id` | Edit permissions |
| Preview sync | `GET /platform-role-templates/sync-preview` | What will change |
| Sync to tenants | `POST /platform-role-templates/sync` | Push to all tenants |
| Read-only view | `GET /platform-role-templates/public` | For tenant users |

### Update Role Permissions

```typescript
// PUT /platform-role-templates/:id
{
  "data": {
    "permissions": ["perm-doc-id-1", "perm-doc-id-2", ...]  // Permission documentIds
  }
}
```

### Sync to Tenants

```typescript
// POST /platform-role-templates/sync
{
  "roleCode": "hr_admin"  // Optional: sync specific role, omit for all
}
```

## System Roles

These roles are created per-tenant and managed by Super Admin:

| Code | Description |
|------|-------------|
| `company_admin` | Full access (always gets ALL permissions) |
| `hr_admin` | Full HR module access |
| `hr_manager` | Limited HR with approve permissions |
| `manager` | Team management and approvals |
| `employee` | Self-service only |
| `payroll_admin` | Payroll module access |

## Tenant User Access

Tenant users see `isReadOnly: true` for system roles:

```typescript
// GET /custom-roles response for tenant
{
  "data": [{
    "code": "hr_admin",
    "isSystemRole": true,
    "isReadOnly": true,  // Cannot modify
    "permissions": [...]
  }]
}
```

## Checking Permissions in Code

```typescript
// In controller
const user = ctx.state.user;
const fullUser = await strapi.db.query('plugin::users-permissions.user').findOne({
  where: { id: user.id },
  populate: ['customRoles', 'customRoles.permissions'],
});

const hasPermission = fullUser.customRoles.some(role =>
  role.permissions.some(p => p.code === 'leave.approve')
);
```

## Workflow: Adding New Feature Permissions

1. **Add permission codes** to `src/bootstrap/seeds/permissions.ts`
2. **Restart server** → permissions auto-seeded to database
3. **Super Admin** opens role management UI
4. **Edit role template** → add new permissions
5. **Sync to tenants** → all tenant roles updated
