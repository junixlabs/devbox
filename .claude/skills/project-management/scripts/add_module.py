#!/usr/bin/env python3
"""
Add a user story module to the documentation.

Usage:
  python3 add_module.py <module-name> [--prefix ABC] [--path ./docs]

Examples:
  python3 add_module.py employee-management --prefix EMP
  python3 add_module.py leave-management --prefix LVE
"""

import argparse
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates"


def kebab_to_title(s: str) -> str:
    """Convert kebab-case to Title Case."""
    return " ".join(word.capitalize() for word in s.split("-"))


def get_module_readme_template(module_name: str, prefix: str) -> str:
    """Generate module README content."""
    title = kebab_to_title(module_name)
    return f"""# {title} Module

## Overview
The {title} module handles [TODO: describe module purpose].

> **🏢 Multi-Tenant Context**: All data is isolated per company (tenant).

---

## Stories by Priority

### 🔴 MUST HAVE (MVP Critical)

| ID | Story | File |
|----|-------|------|
| {prefix}-001 | [TODO: First story] | [{prefix.lower()}-001-first-story.md](./{prefix.lower()}-001-first-story.md) |

### 🟡 SHOULD HAVE (Important)

| ID | Story | File |
|----|-------|------|
| {prefix}-002 | [TODO: Second story] | [{prefix.lower()}-002-second-story.md](./{prefix.lower()}-002-second-story.md) |

### 🟢 NICE TO HAVE (Future)

| ID | Story | File |
|----|-------|------|
| {prefix}-003 | [TODO: Third story] | [{prefix.lower()}-003-third-story.md](./{prefix.lower()}-003-third-story.md) |

---

## Dependencies

```
[TODO: Add dependency diagram]
```

---

*Last Updated: [DATE]*
"""


def get_story_template(module_name: str, prefix: str, number: int) -> str:
    """Generate user story template."""
    title = kebab_to_title(module_name)
    story_id = f"{prefix}-{number:03d}"
    return f"""# {story_id}: [TODO: Story Title]

## Story Details

| Field | Value |
|-------|-------|
| **Story ID** | {story_id} |
| **Module** | {title} |
| **Priority** | 🔴 MUST HAVE |
| **Story Points** | [TODO] |
| **Sprint** | - |

---

## Status

| Field | Value |
|-------|-------|
| **Status** | 📋 Backlog |
| **Assignee** | - |
| **PR** | - |
| **Deployed** | - |
| **Blocked** | - |

---

## User Story

**As a** [role],
**I want to** [action],
**So that** [benefit].

---

## Acceptance Criteria

### AC-1: [TODO: First criterion]
- [ ] [TODO: Acceptance condition]
- [ ] [TODO: Acceptance condition]

### AC-2: [TODO: Second criterion]
- [ ] [TODO: Acceptance condition]

---

## Business Rules

| Rule ID | Description |
|---------|-------------|
| BR-001 | [TODO: Business rule] |

---

## UI/UX Requirements

### Form Layout
```
[TODO: ASCII wireframe or description]
```

### Interactions
- [TODO: Interaction pattern]

---

## Data Model

```
[TODO: Entity definition]
EntityName {{
  id: UUID
  field: Type
}}
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/resource` | [TODO] |
| POST | `/api/v1/resource` | [TODO] |

---

## Dependencies

| Story ID | Dependency Type | Description |
|----------|-----------------|-------------|
| [TODO] | Prerequisite | [TODO] |

---

## Test Scenarios

| Test ID | Scenario | Expected Result |
|---------|----------|-----------------|
| TC-001 | [TODO: Scenario] | [TODO: Result] |

---

*Created: [DATE]*
*Author: [AUTHOR]*
*Version: 1.0*
"""


def main():
    parser = argparse.ArgumentParser(description="Add user story module")
    parser.add_argument("name", help="Module name (kebab-case)")
    parser.add_argument("--prefix", default=None, help="Story ID prefix (e.g., EMP)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    user_stories_path = docs_path / "requirements" / "user-stories"

    if not user_stories_path.exists():
        print(f"❌ User stories directory not found at {user_stories_path}")
        print("Run init_docs.py first to create the docs structure.")
        return

    # Generate prefix from module name if not provided
    prefix = args.prefix
    if not prefix:
        # Take first 3 letters of first word
        words = args.name.split("-")
        prefix = words[0][:3].upper()

    module_path = user_stories_path / args.name

    if module_path.exists():
        print(f"⚠️  Module already exists: {module_path}")
        return

    print(f"\n📚 Creating module: {args.name} (prefix: {prefix})\n")

    # Create module directory
    module_path.mkdir(parents=True)
    print(f"  Created: {module_path}")

    # Create README
    readme_path = module_path / "README.md"
    readme_path.write_text(get_module_readme_template(args.name, prefix))
    print(f"  Created: {readme_path}")

    # Create first story template
    story_path = module_path / f"{prefix.lower()}-001-first-story.md"
    story_path.write_text(get_story_template(args.name, prefix, 1))
    print(f"  Created: {story_path}")

    print(f"\n✅ Module '{args.name}' created successfully")
    print("\nNext steps:")
    print(f"  1. Edit {readme_path} with module overview")
    print(f"  2. Rename and edit {story_path} with first user story")
    print(f"  3. Add more stories: {prefix.lower()}-002-*.md, etc.")


if __name__ == "__main__":
    main()
