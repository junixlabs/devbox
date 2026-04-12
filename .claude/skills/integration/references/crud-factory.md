# CRUD Factory Usage

Factories reduce boilerplate for standard CRUD operations.

## API Factory

Location: `frontend/src/lib/api/crud-factory.ts`

```typescript
import { createCrudApi } from '@/lib/api/crud-factory';
import type { Resource, ResourceFormData } from '../types';

// Creates: list(), get(), create(), update(), delete()
export const resourcesApi = createCrudApi<Resource, ResourceFormData>({
  endpoint: '/resources',
});

// Usage
const items = await resourcesApi.list({ page: 1, pageSize: 10 });
const item = await resourcesApi.get(documentId);
const created = await resourcesApi.create(formData);
const updated = await resourcesApi.update(documentId, partialData);
await resourcesApi.delete(documentId);
```

## Hooks Factory

Location: `frontend/src/lib/hooks/crud-hooks-factory.ts`

```typescript
import { createCrudHooks } from '@/lib/hooks/crud-hooks-factory';
import { resourcesApi } from '../api';

export const {
  useList: useResources,
  useOne: useResource,
  useCreate: useCreateResource,
  useUpdate: useUpdateResource,
  useDelete: useDeleteResource,
} = createCrudHooks({
  queryKey: ['resources'],
  api: resourcesApi,
  invalidateKeys: [['related-resources']], // Optional: cross-invalidation
});
```

## When to Use Factories

**Use factories when:**
- Standard CRUD operations only
- No complex query parameters
- No custom endpoints

**Use custom API when:**
- Complex filters (nested, $or, $containsi)
- State transitions (approve, reject, submit)
- Aggregation endpoints (summary, stats)
- Non-standard responses

## Extending Factory API

```typescript
// Create base with factory
export const resourcesApi = createCrudApi<Resource, ResourceFormData>({
  endpoint: '/resources',
});

// Add custom methods
export async function getResourceSummary(id: string) {
  return api.get<{ data: ResourceSummary }>(`/resources/${id}/summary`);
}

export async function approveResource(id: string, comments?: string) {
  return api.post<{ data: Resource }>(`/resources/${id}/approve`, {
    data: { comments },
  });
}

// Export combined API
export const resourceApi = {
  ...resourcesApi,
  getSummary: getResourceSummary,
  approve: approveResource,
};
```

## Factory with Custom Hooks

```typescript
const baseHooks = createCrudHooks({ queryKey: ['resources'], api: resourcesApi });

// Add custom hook
export function useApproveResource() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resourceApi.approve,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['resources'] });
      queryClient.invalidateQueries({ queryKey: ['pending-approvals'] });
    },
  });
}

export const resourceHooks = {
  ...baseHooks,
  useApprove: useApproveResource,
};
```
