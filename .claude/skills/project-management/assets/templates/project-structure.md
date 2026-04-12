# Project Structure

## Overview

```
project/
├── frontend/          # Next.js App Router
├── backend/           # Strapi CMS
├── docs/              # Documentation
└── openspec/          # Change proposals
```

---

## Frontend (Next.js)

```
frontend/src/
├── app/                          # App Router (routes only)
│   ├── (tenant)/                 # Tenant-scoped routes
│   │   ├── dashboard/
│   │   └── ...
│   ├── (auth)/                   # Auth routes
│   └── layout.tsx
│
├── features/                     # Feature modules
│   └── {name}/
│       ├── components/
│       ├── hooks/
│       ├── api/
│       └── types.ts
│
├── components/ui/                # Shared UI components
│
├── lib/                          # Core utilities
│   ├── api/
│   ├── hooks/
│   ├── constants/
│   ├── utils/
│   └── types/
│
└── providers/                    # Context providers
```

---

## Backend (Strapi)

```
backend/src/
├── api/{name}/
│   ├── content-types/{name}/schema.json
│   ├── controllers/{name}.ts
│   ├── routes/{name}.ts
│   └── services/{name}.ts
│
├── services/                     # Shared services
├── config/                       # Configuration
├── utils/                        # Utilities
├── middlewares/
├── policies/
└── bootstrap/
```

---

## File Naming

| Type | Convention | Example |
|------|------------|---------|
| Components | `kebab-case.tsx` | `employee-card.tsx` |
| Hooks | `use-{name}.ts` | `use-employees.ts` |
| Controllers | `{name}.ts` | `employee.ts` |
| Services | `{name}.ts` | `employee.ts` |

---

## File Size Guidelines

| Type | Max Lines | Action |
|------|-----------|--------|
| Component | 200 | Split into subcomponents |
| Hook file | 100 | Split by domain |
| Controller | 300 | Extract to files |
| Service | 200 | Split into modules |
