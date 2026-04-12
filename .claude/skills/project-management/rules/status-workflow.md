# Story Status Workflow

Standard status transitions for user stories.

---

## Status Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         STORY STATUS WORKFLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐    Sprint     ┌──────────┐    Start    ┌─────────────┐       │
│   │ Backlog  │──────────────▶│ Sprint   │────────────▶│ In Progress │       │
│   │    📋    │   Planning    │    🔜    │    Work     │     🔨      │       │
│   └──────────┘               └──────────┘             └──────┬──────┘       │
│        ▲                                                     │              │
│        │                                              Open PR│              │
│        │                                                     ▼              │
│        │                                              ┌─────────────┐       │
│        │                                              │  In Review  │       │
│        │                                              │     🔍      │       │
│        │                                              └──────┬──────┘       │
│        │                          Request                    │              │
│        │                          Changes             PR Merge│              │
│        └──────────────────────────────────────────────┐      ▼              │
│                                                       │┌─────────────┐      │
│                                                       ││    Done     │      │
│                                                       ││     ✅      │      │
│                                                       │└──────┬──────┘      │
│                                                       │       │             │
│                                                       │ Deploy│             │
│                                                       │       ▼             │
│                                                       │┌─────────────┐      │
│                                                       ││  Deployed   │      │
│                                                       ││     🚀      │      │
│                                                       │└─────────────┘      │
│                                                       │                     │
│   BLOCKED STATE (can occur at any stage)              │                     │
│   ──────────────────────────────────────              │                     │
│   Any status ──▶ 🔴 Blocked ──▶ Original status       │                     │
│                                                       │                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Status Definitions

| Status | Emoji | Description |
|--------|-------|-------------|
| **Backlog** | 📋 | Story defined but not yet planned |
| **Sprint** | 🔜 | Committed to current sprint |
| **In Progress** | 🔨 | Active development |
| **In Review** | 🔍 | PR opened, awaiting review |
| **Done** | ✅ | PR merged, feature complete |
| **Deployed** | 🚀 | Live in production |
| **Blocked** | 🔴 | Cannot proceed (add blocker reason) |

---

## Valid Transitions

| From | To | Trigger |
|------|----|---------|
| Backlog | Sprint | Sprint planning |
| Sprint | In Progress | Developer starts work |
| In Progress | In Review | PR opened |
| In Review | In Progress | Changes requested |
| In Review | Done | PR merged |
| Done | Deployed | Production deploy |
| Any | Blocked | Dependency/issue |
| Blocked | Previous | Blocker resolved |

---

## Update Commands

```bash
# Start working on a story
python3 scripts/update_status.py DOC-001 in-progress

# Open PR for review
python3 scripts/update_status.py DOC-001 review --pr 123

# Mark as done after merge
python3 scripts/update_status.py DOC-001 done --pr 123

# Mark as deployed
python3 scripts/update_status.py DOC-001 deployed

# Mark as blocked
python3 scripts/update_status.py DOC-001 blocked --reason "Waiting for API"
```

---

## Story File Status Section

```markdown
## Status

| Field | Value |
|-------|-------|
| **Status** | 🔨 In Progress |
| **Sprint** | Sprint 02 |
| **PR** | [#123](link) |
| **Deployed** | 2026-02-15 |
| **Blocked** | - |
```
