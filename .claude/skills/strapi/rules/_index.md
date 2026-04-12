# Strapi Rules Index

## Controllers
| Rule | Description |
|------|-------------|
| `ctrl-tenant-context` | withTenantContext wrapper for multi-tenant |
| `ctrl-employee-lookup` | getCurrentEmployee for self-service |
| `ctrl-factory` | createCoreController patterns |
| `ctrl-response` | Response format and error handling |
| `ctrl-constants` | Centralized constants and date utilities |
| `ctrl-routes` | Custom route registration with policies |

## Services
| Rule | Description |
|------|-------------|
| `svc-modular` | Facade pattern for large services |
| `svc-workflow-engine` | Complex service structure |

## Data Access
| Rule | Description |
|------|-------------|
| `document-api` | Document API: CRUD, populate, plugin queries |
| `relations` | Relation filters, documentId, schema |
| `lifecycle` | Lifecycle hooks |

## Auth & Permissions
| Rule | Description |
|------|-------------|
| `authorization` | Policies, PolicyError, middleware order |
| `api-permissions` | Seeding API permissions for authenticated users |
| `rbac-permissions` | RBAC permissions for tenant roles (Super Admin managed) |
