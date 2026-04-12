# Technical Specification: [Feature Name]

## Document Information

| Field | Value |
|-------|-------|
| **Version** | 1.0 |
| **Author** | [Author] |
| **Created** | [Date] |
| **Last Updated** | [Date] |
| **Status** | Draft / In Review / Approved |
| **Related PRD** | [Link to PRD] |

---

## Overview

### Summary

[Brief technical summary of what this spec covers]

### Goals

1. [Technical goal 1]
2. [Technical goal 2]

### Non-Goals

1. [What this spec explicitly does NOT cover]
2. [Out of scope technical decisions]

---

## Architecture

### System Context

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Client    │─────▶│   Server    │─────▶│  Database   │
└─────────────┘      └─────────────┘      └─────────────┘
                            │
                            ▼
                     ┌─────────────┐
                     │  External   │
                     │   Service   │
                     └─────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                      Application                         │
├──────────────────┬──────────────────┬───────────────────┤
│   Presentation   │    Business      │   Data Access     │
│      Layer       │     Layer        │      Layer        │
├──────────────────┼──────────────────┼───────────────────┤
│ - Components     │ - Services       │ - Repositories    │
│ - Pages          │ - Validators     │ - Models          │
│ - Hooks          │ - Transformers   │ - Queries         │
└──────────────────┴──────────────────┴───────────────────┘
```

---

## Data Model

### Entity Definitions

```typescript
interface Entity {
  id: string;           // UUID, primary key
  field1: Type;         // Description
  field2: Type;         // Description
  createdAt: Date;      // Auto-generated
  updatedAt: Date;      // Auto-updated
}
```

### Database Schema

```sql
CREATE TABLE entity (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  field1 VARCHAR(255) NOT NULL,
  field2 INTEGER,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_entity_field1 ON entity(field1);
```

### Entity Relationships

```
┌─────────┐     1:N     ┌─────────┐
│  User   │────────────▶│  Order  │
└─────────┘             └─────────┘
                             │
                             │ N:M
                             ▼
                        ┌─────────┐
                        │ Product │
                        └─────────┘
```

---

## API Design

### Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/resources` | List resources | Required |
| GET | `/api/v1/resources/:id` | Get single resource | Required |
| POST | `/api/v1/resources` | Create resource | Required |
| PUT | `/api/v1/resources/:id` | Update resource | Required |
| DELETE | `/api/v1/resources/:id` | Delete resource | Required |

### Request/Response Examples

#### GET /api/v1/resources

**Request:**
```http
GET /api/v1/resources?page=1&limit=10
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "field1": "value",
      "field2": 123
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 100
  }
}
```

#### POST /api/v1/resources

**Request:**
```json
{
  "field1": "value",
  "field2": 123
}
```

**Response (201):**
```json
{
  "data": {
    "id": "uuid",
    "field1": "value",
    "field2": 123,
    "createdAt": "2026-01-31T00:00:00Z"
  }
}
```

### Error Responses

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid input |
| 401 | `UNAUTHORIZED` | Missing/invalid auth |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 500 | `INTERNAL_ERROR` | Server error |

---

## Implementation Details

### File Structure

```
src/
├── features/[feature]/
│   ├── components/
│   │   └── [component].tsx
│   ├── hooks/
│   │   └── use-[feature].ts
│   ├── api/
│   │   └── [feature]-api.ts
│   ├── types.ts
│   └── index.ts
```

### Key Components

#### Component: [Name]

**Purpose:** [What this component does]

**Props:**
```typescript
interface Props {
  prop1: Type;    // Description
  prop2?: Type;   // Optional, description
  onEvent: () => void;
}
```

**Usage:**
```tsx
<Component prop1={value} onEvent={handleEvent} />
```

### State Management

| State | Location | Update Triggers |
|-------|----------|-----------------|
| [State 1] | [Where stored] | [When updated] |
| [State 2] | [Where stored] | [When updated] |

---

## Security Considerations

### Authentication

- [Auth method: JWT/Session/etc.]
- [Token storage strategy]
- [Refresh mechanism]

### Authorization

| Role | Permissions |
|------|-------------|
| Admin | Full access |
| User | Read/Write own data |
| Guest | Read-only public data |

### Data Protection

- [ ] Input validation
- [ ] SQL injection prevention
- [ ] XSS prevention
- [ ] CSRF protection
- [ ] Rate limiting

---

## Performance

### Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| API Response Time | < 200ms p95 | Monitoring |
| Page Load | < 2s | Lighthouse |
| Database Query | < 50ms | Query logs |

### Optimization Strategies

1. **Caching:** [Strategy]
2. **Database:** [Indexing strategy]
3. **Frontend:** [Bundle optimization]

---

## Testing Strategy

### Unit Tests

| Component | Coverage Target |
|-----------|-----------------|
| Services | 90% |
| Utils | 100% |
| Components | 80% |

### Integration Tests

| Scenario | Priority |
|----------|----------|
| [Happy path] | High |
| [Error handling] | High |
| [Edge cases] | Medium |

### E2E Tests

| Flow | Test File |
|------|-----------|
| [User flow 1] | `tests/e2e/flow1.spec.ts` |
| [User flow 2] | `tests/e2e/flow2.spec.ts` |

---

## Migration Plan

### Database Migrations

```sql
-- Migration: 001_create_table
-- Up
CREATE TABLE ...

-- Down
DROP TABLE ...
```

### Data Migration

1. [Step 1]
2. [Step 2]
3. [Rollback procedure]

---

## Monitoring & Observability

### Logging

| Level | Use Case |
|-------|----------|
| ERROR | Exceptions, failures |
| WARN | Degraded service |
| INFO | Key operations |
| DEBUG | Development only |

### Metrics

| Metric | Type | Alert Threshold |
|--------|------|-----------------|
| API latency | Histogram | p99 > 1s |
| Error rate | Counter | > 1% |
| Active users | Gauge | - |

---

## Rollout Plan

### Feature Flags

| Flag | Default | Rollout |
|------|---------|---------|
| `feature_enabled` | false | 10% → 50% → 100% |

### Rollback Procedure

1. Disable feature flag
2. Revert database migration (if needed)
3. Deploy previous version

---

## Open Questions

| # | Question | Owner | Decision |
|---|----------|-------|----------|
| 1 | [Question] | [Name] | Pending |
| 2 | [Question] | [Name] | [Decision] |

---

## References

- PRD: `docs/prd/[feature]-prd.md`
- User Stories: `docs/user-stories/[module]/`
- Design: [Link]
- API Docs: [Link]
