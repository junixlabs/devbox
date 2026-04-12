# [ID]: [Story Title]

## Story Details

| Field | Value |
|-------|-------|
| **Story ID** | [PREFIX-NNN] |
| **Module** | [Module Name] |
| **Priority** | 🔴 MUST HAVE / 🟡 SHOULD HAVE / 🟢 NICE TO HAVE |
| **Story Points** | [N] |
| **Sprint** | Sprint [N] |

---

## Status

| Field | Value |
|-------|-------|
| **Status** | 📋 Backlog / 🔜 Sprint / 🔨 In Progress / 🔍 In Review / ✅ Done / 🚀 Deployed |
| **Assignee** | [Name] |
| **PR** | [#NNN](link) |
| **Deployed** | [Date] |
| **Blocked** | - |

---

## User Story

**As a** [role],
**I want to** [action],
**So that** [benefit].

---

## Acceptance Criteria

### AC-1: [Criterion Name]
- [ ] [Testable condition]
- [ ] [Testable condition]

### AC-2: [Criterion Name]
- [ ] [Testable condition]
- [ ] [Testable condition]

### AC-3: [Criterion Name]
- [ ] [Testable condition]

---

## Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-001 | [Business rule] |
| BR-002 | [Business rule] |

---

## UI/UX Requirements

### Wireframe

```
┌─────────────────────────────────────┐
│  [Component Layout]                 │
├─────────────────────────────────────┤
│                                     │
│  [Content Area]                     │
│                                     │
│  ┌─────────┐  ┌─────────┐          │
│  │ Action  │  │ Action  │          │
│  └─────────┘  └─────────┘          │
│                                     │
└─────────────────────────────────────┘
```

### Interactions

| Interaction | Behavior |
|-------------|----------|
| [Action] | [Result] |
| [Action] | [Result] |

### Validation

| Field | Rules |
|-------|-------|
| [Field] | Required, [constraints] |
| [Field] | [constraints] |

### Error States

| Error | Message |
|-------|---------|
| [Condition] | [User-facing message] |
| [Condition] | [User-facing message] |

---

## Data Model

```typescript
interface Entity {
  id: string;           // UUID
  field1: Type;         // Description
  field2: Type;         // Description
  createdAt: Date;
  updatedAt: Date;
}
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/resource` | List resources |
| POST | `/api/v1/resource` | Create resource |
| PUT | `/api/v1/resource/:id` | Update resource |
| DELETE | `/api/v1/resource/:id` | Delete resource |

### Request/Response

**POST /api/v1/resource**

Request:
```json
{
  "field1": "value",
  "field2": 123
}
```

Response (201):
```json
{
  "data": {
    "id": "uuid",
    "field1": "value",
    "field2": 123
  }
}
```

---

## Dependencies

| Story ID | Type | Description |
|----------|------|-------------|
| [ID] | Prerequisite | [Why needed] |
| [ID] | Related | [How related] |

---

## Test Scenarios

| Test ID | Scenario | Steps | Expected Result |
|---------|----------|-------|-----------------|
| TC-001 | [Happy path] | [Steps] | [Result] |
| TC-002 | [Error case] | [Steps] | [Result] |
| TC-003 | [Edge case] | [Steps] | [Result] |

---

## Technical Notes

[Implementation notes, considerations, or decisions]

---

## Change Log

| Date | Author | Change |
|------|--------|--------|
| [Date] | [Name] | Initial creation |

---

*Created: [DATE]*
*Author: [AUTHOR]*
*Version: 1.0*
