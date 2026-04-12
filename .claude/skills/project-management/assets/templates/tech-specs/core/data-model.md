# Data Model

## Entities

### [Entity Name]
```typescript
interface EntityName {
  id: number;
  field1: string;
  field2?: Type;
  createdAt: Date;
  updatedAt: Date;
}
```

### [Entity Name 2]
```typescript
interface EntityName2 {
  id: number;
  field1: string;
  relation: EntityName;
  createdAt: Date;
  updatedAt: Date;
}
```

---

## Relationships

```
┌──────────┐     1:N     ┌──────────┐
│ Entity1  │────────────▶│ Entity2  │
└──────────┘             └──────────┘
```

---

## Database Schema

```sql
CREATE TABLE entity_name (
  id SERIAL PRIMARY KEY,
  field1 VARCHAR(255) NOT NULL,
  field2 INTEGER,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_entity_field1 ON entity_name(field1);
```
