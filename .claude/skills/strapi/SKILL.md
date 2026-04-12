---
name: strapi
description: Strapi 5 toolbox. Scripts for project/API initialization, rules for development patterns.
---

# Strapi Skill

Toolbox for Strapi 5 backend development.

## Scripts

| Command | Description |
|---------|-------------|
| `python3 scripts/init_project.py <name> --new` | Create new Strapi project |
| `python3 scripts/init_project.py <name>` | Add structure to existing |
| `python3 scripts/init_api.py <name>` | Create API module |
| `python3 scripts/check_permissions.py` | Audit route → permission coverage |

## Rules

Development patterns in `rules/`:

| File | Topic |
|------|-------|
| `_index.md` | Table of contents |
| `document-api.md` | Document API usage |
| `controllers.md` | Controller patterns index |
| `authorization.md` | Auth patterns |
| `relations.md` | Relation filters |
| `lifecycle.md` | Lifecycle hooks |

## Related Skills

| Skill | Purpose |
|-------|---------|
| `strapi-server` | Start/stop dev server |
| `tester` | Run tests |

## References

- Project structure: `docs/project-structure.md`
