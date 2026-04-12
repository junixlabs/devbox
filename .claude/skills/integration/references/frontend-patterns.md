# Frontend Patterns (Next.js 16)

## API Client Usage

Base client at `frontend/src/lib/api/client.ts`:

```typescript
import { api } from '@/lib/api/client';

// GET with query params
const response = await api.get<ResourceResponse>('/resources', {
  'pagination[page]': page,
  'pagination[pageSize]': pageSize,
  'filters[status][$eq]': 'active',
  'populate[relation][fields][0]': 'name',
});

// POST
const created = await api.post<Resource>('/resources', { data: input });

// PUT
const updated = await api.put<Resource>(`/resources/${id}`, { data: input });

// DELETE
await api.delete(`/resources/${id}`);
```

## API Functions Pattern

Feature API at `frontend/src/features/{feature}/api/index.ts`:

```typescript
import { api } from '@/lib/api/client';
import type { Resource, ResourceFormData } from '../types';

interface ListParams {
  page?: number;
  pageSize?: number;
  status?: string;
}

export async function getResources(params?: ListParams) {
  const query: Record<string, string> = {};
  if (params?.page) query['pagination[page]'] = String(params.page);
  if (params?.pageSize) query['pagination[pageSize]'] = String(params.pageSize);
  if (params?.status) query['filters[status][$eq]'] = params.status;

  return api.get<{ data: Resource[]; meta: { pagination: Pagination } }>(
    '/resources',
    query
  );
}

export async function getResource(id: string) {
  return api.get<{ data: Resource }>(`/resources/${id}`);
}

export async function createResource(data: ResourceFormData) {
  return api.post<{ data: Resource }>('/resources', { data });
}

export async function updateResource(id: string, data: Partial<ResourceFormData>) {
  return api.put<{ data: Resource }>(`/resources/${id}`, { data });
}

export async function deleteResource(id: string) {
  return api.delete(`/resources/${id}`);
}
```

## React Query Hooks Pattern

Hooks at `frontend/src/features/{feature}/hooks/index.ts`:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as resourceApi from '../api';

const RESOURCE_KEY = 'resources';

export function useResources(params?: ListParams) {
  return useQuery({
    queryKey: [RESOURCE_KEY, 'list', params],
    queryFn: () => resourceApi.getResources(params),
  });
}

export function useResource(id: string | undefined) {
  return useQuery({
    queryKey: [RESOURCE_KEY, 'detail', id],
    queryFn: () => resourceApi.getResource(id!),
    enabled: !!id,
  });
}

export function useCreateResource() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resourceApi.createResource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [RESOURCE_KEY] });
    },
  });
}

export function useUpdateResource() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<ResourceFormData> }) =>
      resourceApi.updateResource(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: [RESOURCE_KEY] });
    },
  });
}

export function useDeleteResource() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: resourceApi.deleteResource,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [RESOURCE_KEY] });
    },
  });
}
```

## Query Key Conventions

```typescript
// List queries
[RESOURCE_KEY, 'list']
[RESOURCE_KEY, 'list', { status: 'active' }]
[RESOURCE_KEY, 'list', { page: 1, status: 'active' }]

// Detail queries
[RESOURCE_KEY, 'detail', documentId]

// Special queries
[RESOURCE_KEY, 'my']  // Current user's records
[RESOURCE_KEY, 'pending']  // Filtered subset
```

## Cross-Resource Invalidation

```typescript
export function useApproveRequest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: approveRequest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['requests'] });
      queryClient.invalidateQueries({ queryKey: ['balances'] }); // Related
    },
  });
}
```
