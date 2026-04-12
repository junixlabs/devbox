# Documentation Flow

How documents flow from business requirements to implementation.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DOCUMENTATION FLOW                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────┐      ┌─────────┐      ┌─────────────┐      ┌──────────┐       │
│   │   BRD   │─────▶│   PRD   │─────▶│ User Stories│─────▶│Tech Spec │       │
│   │ Business│      │ Product │      │   Feature   │      │Technical │       │
│   │ Context │      │  Goals  │      │   Details   │      │  Design  │       │
│   └─────────┘      └─────────┘      └─────────────┘      └────┬─────┘       │
│                                                                │             │
│                                                                ▼             │
│                                                         ┌──────────┐        │
│                                                         │   Code   │        │
│                                                         │          │        │
│                                                         └──────────┘        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Document Purposes

| Document | Purpose | Audience | Timing |
|----------|---------|----------|--------|
| **BRD** | Define business need and objectives | Stakeholders, Management | Project inception |
| **PRD** | Translate business needs to product features | Product, Design, Engineering | After BRD approval |
| **User Stories** | Break features into implementable units | Engineering, QA | Sprint planning |
| **Tech Spec** | Define technical implementation approach | Engineering | Before implementation |

---

## 1. BRD (Business Requirements Document)

**Location:** `docs/brd/{project}-brd.md`

**Contains:**
- Business objectives and success metrics
- Problem statement and gap analysis
- Scope, assumptions, constraints
- High-level requirements
- Risk assessment
- Timeline and budget

**Created by:** Product Owner / Business Analyst

**Approved by:** Stakeholders, Sponsors

**Triggers PRD creation** when approved.

---

## 2. PRD (Product Requirements Document)

**Location:** `docs/prd/{feature}-prd.md`

**Contains:**
- Problem and solution overview
- Target users and personas
- Goals and KPIs
- Functional requirements
- User experience details
- Non-functional requirements
- Release plan

**Created by:** Product Manager

**Approved by:** Product Owner, Tech Lead

**Triggers user story creation** when approved.

---

## 3. User Stories

**Location:** `docs/user-stories/{module}/{prefix}-{nnn}-{name}.md`

**Contains:**
- User story statement (As a... I want... So that...)
- Acceptance criteria
- Business rules
- UI/UX requirements
- API endpoints
- Test scenarios

**Created by:** Product Manager / Business Analyst

**Estimated by:** Engineering team

**Assigned to:** Sprints during planning

---

## 4. Tech Spec (Technical Specification)

**Location:** `docs/tech-specs/{feature}-spec.md`

**Contains:**
- Architecture and system design
- Data model and schema
- API design and contracts
- Security considerations
- Performance requirements
- Testing strategy
- Migration and rollout plan

**Created by:** Tech Lead / Senior Engineer

**Reviewed by:** Engineering team

**Guides implementation** of user stories.

---

## Workflow Example

### Phase 1: Project Initiation

```bash
# 1. Initialize project docs
python3 scripts/init_docs.py ./project --new

# 2. Create BRD
python3 scripts/add_brd.py employee-portal
# Fill in: docs/brd/employee-portal-brd.md
```

### Phase 2: Feature Planning

```bash
# 3. Create PRD (after BRD approval)
python3 scripts/add_prd.py user-authentication --brd employee-portal
# Fill in: docs/prd/user-authentication-prd.md

# 4. Create user story module
python3 scripts/add_module.py authentication --prefix AUTH
# Fill in: docs/user-stories/authentication/
```

### Phase 3: Technical Design

```bash
# 5. Create tech spec (for complex features)
python3 scripts/add_spec.py user-authentication --prd user-authentication
# Fill in: docs/tech-specs/user-authentication-spec.md
```

### Phase 4: Sprint Execution

```bash
# 6. Create sprint
python3 scripts/add_sprint.py 01 --start 2026-02-01 --end 2026-02-14

# 7. Update story status as work progresses
python3 scripts/update_status.py AUTH-001 in-progress
python3 scripts/update_status.py AUTH-001 review --pr 123
python3 scripts/update_status.py AUTH-001 done --pr 123
```

---

## Document Relationships

```
BRD: employee-portal-brd.md
├── PRD: user-authentication-prd.md
│   ├── Story: auth-001-user-login.md
│   ├── Story: auth-002-user-registration.md
│   └── Story: auth-003-password-reset.md
├── PRD: dashboard-prd.md
│   └── Story: dash-001-overview.md
└── Tech Spec: user-authentication-spec.md
```

---

## When to Create Each Document

| Scenario | Documents Needed |
|----------|------------------|
| New project | BRD → PRD → Stories → Spec |
| New feature (large) | PRD → Stories → Spec |
| New feature (small) | Stories only |
| Bug fix | None (use bug report) |
| Technical improvement | Spec only |

---

## Best Practices

1. **Don't skip steps** - Even for small features, at least create user stories
2. **Link documents** - Reference related documents in each file
3. **Keep in sync** - Update documents when requirements change
4. **Version control** - Track changes with git
5. **Review regularly** - Ensure documents match implementation
