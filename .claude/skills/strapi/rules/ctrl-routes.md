# ctrl-routes

Register custom routes with policies and middlewares.

```typescript
// src/api/leave-request/routes/leave-request.ts
export default {
  routes: [
    // Authenticated user routes
    {
      method: 'GET',
      path: '/leave-requests',
      handler: 'leave-request.find',
      config: {
        policies: ['global::is-authenticated'],
        middlewares: ['global::tenant-context'],
      },
    },
    // Self-service routes
    {
      method: 'GET',
      path: '/leave-requests/my-requests',
      handler: 'leave-request.myRequests',
      config: {
        policies: ['global::is-authenticated'],
        middlewares: ['global::tenant-context'],
      },
    },
    // RBAC-protected routes
    {
      method: 'POST',
      path: '/leave-requests/:id/approve',
      handler: 'leave-request.approve',
      config: {
        policies: [
          'global::is-authenticated',
          { name: 'global::check-permission', config: { permission: 'leave.approve' } },
        ],
        middlewares: ['global::tenant-context'],
      },
    },
    // Super admin only
    {
      method: 'POST',
      path: '/leave-requests/bulk-delete',
      handler: 'leave-request.bulkDelete',
      config: {
        policies: ['global::is-super-admin'],
      },
    },
    // Feature-gated routes
    {
      method: 'POST',
      path: '/leave-requests/:id/workflow',
      handler: 'leave-request.startWorkflow',
      config: {
        policies: ['global::is-tenant-owner'],
        middlewares: [
          'global::tenant-context',
          { name: 'global::feature-gate', config: { feature: 'approvals' } },
        ],
      },
    },
  ],
};
```

## Route naming

- Use kebab-case for paths: `/leave-requests`, `/shift-assignments`
- Use camelCase for handlers: `leave-request.myRequests`
- Group by purpose: CRUD first, then custom actions
