---
name: nextjs
description: Next.js development patterns, conventions, and best practices for App Router projects. Use when building Next.js pages, components, hooks, or reviewing Next.js code.
---

# Next.js Skill

Next.js 16 App Router development patterns and conventions.

## Project Structure

```
src/
  app/           # Pages and layouts (App Router)
  components/    # Shared UI components
  features/      # Feature modules (co-locate by domain)
  hooks/         # Shared React hooks
  lib/           # Utilities, API clients
```

## Key Patterns

### Data Fetching
- Use React Query for client-side data fetching
- Use Server Components for initial data when possible
- Never call APIs directly from components — use hooks

### Component Conventions
- Keep components under 200 lines — extract to sub-components
- Co-locate styles, tests, and types with components
- Use `"use client"` only when needed (interactivity, browser APIs)

### Routing
- File-based routing via `app/` directory
- Dynamic routes: `[id]/page.tsx`
- Route groups: `(group)/page.tsx` (don't affect URL)
- Loading states: `loading.tsx`
- Error boundaries: `error.tsx`

## Rules

See `rules/` directory for specific patterns:
- `rules/component-size.md` — Component size limits
- `rules/hooks-modular.md` — Hook patterns
- `rules/page-extraction.md` — Page component patterns

## Common Commands

```bash
npm run dev      # Development server
npm run build    # Production build
npm run lint     # ESLint
npx tsc --noEmit # TypeScript check
```
