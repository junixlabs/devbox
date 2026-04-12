---
name: project-management
description: |
  Project documentation and tracking. Use when: (1) initializing project docs,
  (2) creating BRD/PRD/stories/specs, (3) managing sprints and roadmap,
  (4) updating story status, (5) maintaining changelog. Full SDLC documentation.
---

# Project Management Skill

Manage project documentation from business requirements to deployment.

## Initialize Project

```bash
python3 scripts/init_docs.py <path> --new   # Full structure
python3 scripts/init_git.py <path>          # Add git
```

## Create Documents

| Document | Command |
|----------|---------|
| BRD | `python3 scripts/add_brd.py <name>` |
| PRD (init) | `python3 scripts/add_prd.py <name>` |
| PRD Feature | `python3 scripts/add_prd_feature.py <name>` |
| User Story Module | `python3 scripts/add_module.py <name>` |
| Tech Spec (init) | `python3 scripts/add_spec.py <name>` |
| Tech Spec Module | `python3 scripts/add_spec_module.py <name>` |
| Sprint | `python3 scripts/add_sprint.py <number>` |

## Test Specs

| Layer | Source | Format |
|-------|--------|--------|
| Backend | `tech-specs/modules/*.md` | API contracts |
| Frontend | `test-specs/journeys/*.feature` | Gherkin |

```bash
# Frontend E2E: user journey tests
python3 scripts/add_journey.py admin
python3 scripts/add_journey.py editor
```

→ See `references/gherkin-guide.md` for Gherkin syntax

## Update Status

```bash
python3 scripts/update_status.py <story-id> <status>

# Status options: backlog, sprint, in-progress, review, done, deployed, blocked

# Examples:
python3 scripts/update_status.py DOC-001 in-progress
python3 scripts/update_status.py DOC-001 review --pr 123
python3 scripts/update_status.py DOC-001 done --pr 123
python3 scripts/update_status.py DOC-001 deployed
python3 scripts/update_status.py DOC-001 blocked --reason "Waiting for API"
```

## Documentation Structure

```
docs/
├── requirements/
│   ├── brd/                   # Business Requirements (single file)
│   │   └── {project}-brd.md
│   │
│   ├── prd/                   # Product Requirements (modular)
│   │   ├── README.md          # Index + overview
│   │   ├── features/          # Feature specs
│   │   │   ├── auth.md
│   │   │   ├── documents.md
│   │   │   └── {feature}.md
│   │   ├── ux.md              # User experience
│   │   └── nfr.md             # Non-functional requirements
│   │
│   ├── user-stories/          # User Stories (by module)
│   │   ├── README.md
│   │   └── {module}/
│   │       └── {prefix}-{nnn}-*.md
│   │
│   ├── tech-specs/            # Technical Specs (modular)
│   │   ├── README.md          # Index + decisions
│   │   ├── core/              # Foundation specs
│   │   └── modules/           # Feature specs
│   │
│   └── test-specs/            # Test Specs
│       ├── README.md
│       └── journeys/          # Frontend E2E (Gherkin)
│           └── {role}.feature
│
├── tracking/
│   ├── roadmap.md
│   ├── changelog.md
│   └── sprints/
│
├── guidelines/
│
├── bugs/
│
└── project-structure.md
```

## Rules

| Rule | File |
|------|------|
| When to update tracking | `rules/tracking-rules.md` |
| Status workflow | `rules/status-workflow.md` |
| Changelog format | `rules/changelog-rules.md` |

## Documentation Flow

→ See `references/documentation-flow.md`

```
BRD → PRD → Tech Spec → Test Specs → Implementation
```

## Templates

Templates in `assets/templates/`:

| Template | Purpose |
|----------|---------|
| `brd.md` | Business Requirements Document |
| `prd.md` | Product Requirements Document |
| `user-story.md` | User story format |
| `tech-spec.md` | Technical specification |
| `roadmap.md` | Project roadmap |
| `sprint.md` | Sprint tracking |
| `changelog.md` | Release changelog |

## References

- Documentation flow: `references/documentation-flow.md`
- Template guide: `references/templates-guide.md`
- User story format: `references/user-story-format.md`
- Gherkin syntax: `references/gherkin-guide.md`
