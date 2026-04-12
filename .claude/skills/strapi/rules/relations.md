# Relations

## rel-nested-filter

Use nested filters for relations (never flat).

**Wrong:**
```typescript
filters: { teamId: teamId }
filters: { 'team.id': teamId }
```

**Correct:**
```typescript
filters: {
  team: { documentId: { $eq: teamId } }
}

// Multiple conditions
filters: {
  team: { documentId: { $eq: teamId } },
  status: { $eq: 'active' }
}
```

## rel-documentId

Always use documentId for relation references.

**Wrong:**
```typescript
data: { team: 1 }
data: { team: { id: 1 } }
```

**Correct:**
```typescript
data: { team: 'abc123documentId' }

// Or connect syntax
data: {
  team: { connect: ['abc123documentId'] }
}
```

## rel-schema

Define relations in schema correctly.

```json
{
  "attributes": {
    "tenant": {
      "type": "relation",
      "relation": "manyToOne",
      "target": "api::tenant.tenant"
    },
    "employees": {
      "type": "relation",
      "relation": "oneToMany",
      "target": "api::employee.employee",
      "mappedBy": "department"
    },
    "permissions": {
      "type": "relation",
      "relation": "manyToMany",
      "target": "api::permission.permission"
    }
  }
}
```

## rel-operators

Filter operators for relations.

```typescript
filters: {
  // Equality
  tenant: { documentId: { $eq: tenantId } },

  // Not equal
  status: { $ne: 'deleted' },

  // In array
  status: { $in: ['draft', 'review'] },

  // Starts with
  action: { $startsWith: 'api::employee' },

  // Null check
  deletedAt: { $null: true }
}
```

> **Note**: For populate patterns, see `document-api.md` → doc-populate.
