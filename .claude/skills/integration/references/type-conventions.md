# Type Conventions

## Type File Location

Types at `frontend/src/features/{feature}/types.ts`

## Entity Type (Backend Response)

```typescript
// Full entity from backend
export interface Resource {
  documentId: string;  // Always use documentId, not id
  name: string;
  status: 'active' | 'inactive';
  createdAt: string;
  updatedAt: string;

  // Relations - include only displayed fields
  tenant: { documentId: string };
  category?: {
    documentId: string;
    name: string;
  };
  createdBy?: {
    documentId: string;
    firstName: string;
    lastName: string;
  };
}
```

## List Item Type (Optimized for Lists)

```typescript
// Lighter type for table/list views
export interface ResourceListItem {
  documentId: string;
  name: string;
  status: 'active' | 'inactive';
  categoryName?: string;  // Flattened
}
```

## Form Data Type (Mutations)

```typescript
// Input for create/update - uses string IDs for relations
export interface ResourceFormData {
  name: string;
  status: 'active' | 'inactive';
  category?: string;  // documentId as string
}

// Partial for updates
export type ResourceUpdateData = Partial<ResourceFormData>;
```

## API Response Types

```typescript
export interface Pagination {
  page: number;
  pageSize: number;
  pageCount: number;
  total: number;
}

export interface ListResponse<T> {
  data: T[];
  meta: {
    pagination: Pagination;
  };
}

export interface SingleResponse<T> {
  data: T;
}
```

## Naming Conventions

| Pattern | Example | Use Case |
|---------|---------|----------|
| `{Entity}` | `Employee` | Full backend entity |
| `{Entity}ListItem` | `EmployeeListItem` | Table/list display |
| `{Entity}FormData` | `EmployeeFormData` | Create mutation input |
| `{Entity}UpdateData` | `EmployeeUpdateData` | Update mutation input |
| `{Entity}Filters` | `EmployeeFilters` | Query parameters |

## Status Enums

```typescript
// Define as union types, not enums
export type ResourceStatus = 'draft' | 'active' | 'archived';

// For display labels, use constants
export const RESOURCE_STATUS_LABELS: Record<ResourceStatus, string> = {
  draft: 'Draft',
  active: 'Active',
  archived: 'Archived',
};
```

## Nested Relation Types

```typescript
// Inline for simple relations
manager?: {
  documentId: string;
  firstName: string;
  lastName: string;
};

// Shared type for complex/reused relations
export interface EmployeeRef {
  documentId: string;
  firstName: string;
  lastName: string;
  email: string;
}
```
