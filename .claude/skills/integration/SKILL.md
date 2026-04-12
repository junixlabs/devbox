---
name: integration
description: |
  Strapi 5 + Next.js 16 integration skill for consistent API bridging. Use when:
  (1) checking frontend-backend integration for mismatches and anti-patterns,
  (2) generating frontend types from Strapi schemas, (3) creating API client functions,
  (4) adding React Query hooks for data fetching, (5) connecting frontend to existing
  backend endpoints. Covers integration auditing, type generation, API client patterns.
version: 1.1.0
---

# Integration Skill

Bridges existing Strapi 5 backend APIs to Next.js 16 frontend with consistent patterns.

## When to Use

- **Checking integration** for mismatches between frontend and backend
- Generating TypeScript types from Strapi schema.json
- Creating frontend API functions for existing endpoints
- Adding React Query hooks for data fetching
- Connecting new frontend features to backend APIs

## Quick Reference

### Project Structure
```
backend/src/api/{resource}/content-types/{resource}/schema.json  # Source
frontend/src/features/{feature}/types.ts      # Generated types
frontend/src/features/{feature}/api/index.ts  # API functions
frontend/src/features/{feature}/hooks/        # React Query hooks
frontend/src/lib/api/                         # Shared utilities
```

### Key Patterns
- **documentId**: Strapi's unique identifier (not numeric id)
- **Response format**: `{ data: T, meta?: { pagination } }`
- **Form data**: Relations use string documentId
- **API calls**: Always use `api` client, never direct `fetch()`

## Scripts

### Check Integration (Audit)

```bash
# Check for integration issues
python3 .claude/skills/integration/scripts/check_integration.py

# Verbose output with all suggestions
python3 .claude/skills/integration/scripts/check_integration.py --verbose
```

**Detects:**
- Direct `fetch()` calls instead of API client
- Calls to non-existent Next.js `/api/` routes
- **Frontend API calls with no backend implementation** (ERROR)
  - Parses both explicit routes and `createCoreRouter` implicit CRUD routes
  - Uses `pluralName` from schema.json for accurate route matching
- Enum value mismatches between frontend types and backend schemas
- Missing fields in frontend types
- Strapi response unwrapping issues (when `{ data: T }` not handled)

### Generate Code

```bash
# Generate TypeScript types from Strapi schema
python3 .claude/skills/integration/scripts/generate_types.py <resource-name>

# Generate API functions from Strapi routes
python3 .claude/skills/integration/scripts/generate_api.py <resource-name>
```

## References

- `references/frontend-patterns.md` - API client, hooks, type definitions
- `references/type-conventions.md` - Type naming and structure conventions
- `references/crud-factory.md` - Using crud-factory for boilerplate reduction

## Workflow: Connect Frontend to Backend

1. **Check integration**: Run `check_integration.py` to audit existing code
2. **Generate types**: Run `generate_types.py` with resource name
3. **Generate API**: Run `generate_api.py` with resource name
4. **Create hooks**: Add React Query hooks using generated API
5. **Integrate**: Use hooks in components
6. **Verify**: Run `check_integration.py` again to ensure no issues
