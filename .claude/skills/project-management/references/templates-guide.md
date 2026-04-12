# Templates Guide

Documentation templates for consistent project documentation.

## Template Files

| Template | Location | Purpose |
|----------|----------|---------|
| `project-structure.md` | `assets/templates/` | Project architecture |
| `design-system.md` | `assets/templates/` | UI design tokens |
| `code-standards.md` | `assets/templates/` | Coding rules |
| `user-stories-readme.md` | `assets/templates/` | User stories index |

## Customization

### Project Structure Template

Replace placeholders:
- `frontend/` and `backend/` directories match your setup
- Add/remove route groups as needed
- Update file naming conventions

### Design System Template

Fill in:
- Color palette with Tailwind classes and hex values
- Typography with font family and sizes
- Component variants matching your UI library
- Spacing and radius values

### Code Standards Template

Customize:
- Stack references for your technologies
- Commit message format
- Pre-commit checklist items

### User Stories README

Add:
- Module list with folders
- User roles and codes
- Priority definitions if different

## Directory Structure

After initialization:

```
docs/
├── project-structure.md
├── design-guidelines/
│   └── design-system.md
├── rules/
│   └── code-standards.md
├── user-stories/
│   ├── README.md
│   └── {module}/
│       ├── README.md
│       └── {prefix}-001-*.md
└── bugs/
```
