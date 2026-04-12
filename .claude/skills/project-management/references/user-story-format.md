# User Story Format

Standard format for user stories in `docs/user-stories/`.

## File Naming

```
{prefix}-{nnn}-{slug}.md

Examples:
  emp-001-create-employee.md
  lve-002-apply-leave.md
  pay-003-process-payroll.md
```

## Story Structure

### 1. Story Details Table

```markdown
| Field | Value |
|-------|-------|
| **Story ID** | {PREFIX}-{NNN} |
| **Module** | {Module Name} |
| **Priority** | 🔴 MUST HAVE / 🟡 SHOULD HAVE / 🟢 NICE TO HAVE |
| **Story Points** | {N} |
| **Sprint Target** | Sprint {N} |
```

### 2. User Story Statement

```markdown
**As a** {role},
**I want to** {action},
**So that** {benefit}.
```

### 3. Acceptance Criteria

```markdown
### AC-1: {Criterion Name}
- [ ] {Testable condition}
- [ ] {Testable condition}
```

### 4. Business Rules

```markdown
| Rule ID | Description |
|---------|-------------|
| BR-001 | {Rule description} |
```

### 5. UI/UX Requirements

Include ASCII wireframe or description of layout and interactions.

### 6. Data Model

```markdown
EntityName {
  id: UUID (primary key)
  field: Type (constraints)
}
```

### 7. API Endpoints

```markdown
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/resource` | List resources |
| POST | `/api/v1/resource` | Create resource |
```

### 8. Dependencies

```markdown
| Story ID | Type | Description |
|----------|------|-------------|
| {ID} | Prerequisite | {Why needed} |
```

### 9. Test Scenarios

```markdown
| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| TC-001 | {Scenario} | {Result} |
```

## Module README

Each module folder has a README with:

1. **Overview** - Module purpose
2. **Stories by Priority** - Tables for MUST/SHOULD/NICE TO HAVE
3. **Data Model** - SQL or schema overview
4. **Dependencies** - Dependency diagram

## Priority Guidelines

| Priority | Description |
|----------|-------------|
| 🔴 MUST HAVE | MVP requirement. Launch blocked without. |
| 🟡 SHOULD HAVE | Important for release. Can launch without. |
| 🟢 NICE TO HAVE | Future enhancement. |
