# [Module Name] Module

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/resources` | List resources |
| GET | `/api/resources/:id` | Get resource |
| POST | `/api/resources` | Create resource |
| PUT | `/api/resources/:id` | Update resource |
| DELETE | `/api/resources/:id` | Delete resource |

---

## Frontend Structure

```
app/(dashboard)/[module]/
├── page.tsx
└── [id]/page.tsx

features/[module]/
├── api/[module]-api.ts
├── hooks/use-[module].ts
└── components/
    ├── [module]-list.tsx
    └── [module]-form.tsx
```

---

## Backend Structure

```
src/api/[module]/
├── content-types/[module]/
│   ├── schema.json
│   └── lifecycles.ts
├── controllers/[module].ts
├── services/[module].ts
└── routes/[module].ts
```

---

## Key Flows

### Create Flow
```
1. User submits form
2. Validate input
3. Create record
4. Return response
```

---

## Service Examples

```typescript
// src/api/[module]/services/[module].ts
export default ({ strapi }) => ({
  async findAll(tenantId: string) {
    return strapi.documents('api::[module].[module]').findMany({
      filters: { tenant: { documentId: tenantId } },
    });
  },
});
```
