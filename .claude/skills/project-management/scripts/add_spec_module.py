#!/usr/bin/env python3
"""
Add a module to modular Technical Specification.

Usage:
  python3 add_spec_module.py <name> [--path ./docs]

Examples:
  python3 add_spec_module.py auth
  python3 add_spec_module.py documents
  python3 add_spec_module.py workflow --path ./my-project/docs
"""

import argparse
from datetime import datetime
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates" / "tech-specs"


def kebab_to_title(s: str) -> str:
    """Convert kebab-case to Title Case."""
    return " ".join(word.capitalize() for word in s.split("-"))


def get_module_template() -> str:
    """Load module template."""
    template_path = TEMPLATES_DIR / "modules" / "module.md"
    if template_path.exists():
        return template_path.read_text()
    return "# [Module Name] Module\n\nTODO: Add content\n"


def update_readme_index(readme_path: Path, module_name: str, title: str):
    """Add module to README index."""
    if not readme_path.exists():
        return

    content = readme_path.read_text()

    # Find the modules table and add new entry
    new_entry = f"| [{module_name}.md](modules/{module_name}.md) | {title} module |"

    # Check if already in index
    if f"modules/{module_name}.md" in content:
        print(f"  Module already in README index")
        return

    # Find where to insert (after last module entry)
    lines = content.split("\n")
    insert_idx = None

    for i, line in enumerate(lines):
        if line.startswith("| [") and "modules/" in line:
            insert_idx = i + 1

    if insert_idx:
        lines.insert(insert_idx, new_entry)
        readme_path.write_text("\n".join(lines))
        print(f"  Updated: {readme_path}")


def main():
    parser = argparse.ArgumentParser(description="Add tech spec module")
    parser.add_argument("name", help="Module name (kebab-case)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    specs_path = docs_path / "requirements" / "tech-specs"
    modules_path = specs_path / "modules"

    # Check if tech-specs initialized
    if not specs_path.exists():
        print(f"❌ Tech specs not initialized. Run add_spec.py first")
        return

    # Create modules dir if needed
    modules_path.mkdir(parents=True, exist_ok=True)

    # Create module file
    module_file = modules_path / f"{args.name}.md"

    if module_file.exists():
        print(f"❌ Module already exists: {module_file}")
        return

    title = kebab_to_title(args.name)

    print(f"\n📄 Creating Module Spec: {title}\n")

    # Get template and replace placeholders
    content = get_module_template()
    content = content.replace("[Module Name]", title)
    content = content.replace("[module]", args.name)

    # Write module file
    module_file.write_text(content)
    print(f"  Created: {module_file}")

    # Update README index
    update_readme_index(specs_path / "README.md", args.name, title)

    print(f"\n✅ Module '{args.name}' created")
    print(f"\nNext steps:")
    print(f"  1. Define API endpoints")
    print(f"  2. Add frontend/backend structure")
    print(f"  3. Document key flows and services")


if __name__ == "__main__":
    main()
