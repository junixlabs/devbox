# Architecture

## System Context

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Client    │─────▶│   Server    │─────▶│  Database   │
└─────────────┘      └─────────────┘      └─────────────┘
```

---

## Technology Stack

| Layer | Technology | Version |
|-------|------------|---------|
| Frontend | [Framework] | x.x |
| Backend | [Framework] | x.x |
| Database | [Database] | x.x |

---

## Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                      Application                         │
├──────────────────┬──────────────────┬───────────────────┤
│   Presentation   │    Business      │   Data Access     │
└──────────────────┴──────────────────┴───────────────────┘
```

---

## Multi-Tenant Strategy

[If applicable - describe tenant isolation approach]

---

## Entity Relationship Overview

```
[Entity] ─▶ [Entity] ─▶ [Entity]
```

Full entity definitions: [data-model.md](data-model.md)
