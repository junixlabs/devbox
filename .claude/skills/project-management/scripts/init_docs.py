#!/usr/bin/env python3
"""
Initialize documentation structure for a project.

Usage:
  python3 init_docs.py <project-path> --new    # Create new docs structure
  python3 init_docs.py <project-path>          # Add docs to existing project
"""

import argparse
import sys
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates"

DOCS_STRUCTURE = {
    "requirements": {
        "brd": {},
        "prd": {},
        "user-stories": {},
        "tech-specs": {},
    },
    "tracking": {
        "sprints": {},
    },
    "guidelines": {},
    "bugs": {},
}


def get_template(name: str) -> str:
    """Load template from assets/templates."""
    template_path = TEMPLATES_DIR / name
    if template_path.exists():
        return template_path.read_text()
    return f"# {name.replace('.md', '').replace('-', ' ').title()}\n\nTODO: Add content\n"


def create_structure(base_path: Path, structure: dict):
    """Recursively create directory structure."""
    for name, children in structure.items():
        path = base_path / name
        path.mkdir(parents=True, exist_ok=True)
        print(f"  Created: {path}")
        if children:
            create_structure(path, children)


def create_docs_files(docs_path: Path):
    """Create documentation files from templates."""
    files = {
        "project-structure.md": "project-structure.md",
        "tracking/roadmap.md": "roadmap.md",
        "tracking/changelog.md": "changelog.md",
        "guidelines/design-system.md": "design-system.md",
        "guidelines/code-standards.md": "code-standards.md",
        "requirements/user-stories/README.md": "user-stories-readme.md",
    }

    for file_path, template_name in files.items():
        full_path = docs_path / file_path
        if not full_path.exists():
            full_path.parent.mkdir(parents=True, exist_ok=True)
            content = get_template(template_name)
            full_path.write_text(content)
            print(f"  Created: {full_path}")


def main():
    parser = argparse.ArgumentParser(description="Initialize project documentation")
    parser.add_argument("path", help="Project path")
    parser.add_argument("--new", action="store_true", help="Create new docs structure")
    args = parser.parse_args()

    project_path = Path(args.path).resolve()
    docs_path = project_path / "docs"

    print(f"\n📚 Initializing documentation for: {project_path}\n")

    if args.new:
        if docs_path.exists():
            print(f"⚠️  docs/ already exists at {docs_path}")
            response = input("Overwrite? [y/N]: ").strip().lower()
            if response != "y":
                print("Aborted.")
                sys.exit(0)

    # Create directory structure
    print("📁 Creating directory structure...")
    create_structure(docs_path, DOCS_STRUCTURE)

    # Create template files
    print("\n📄 Creating documentation files...")
    create_docs_files(docs_path)

    print(f"\n✅ Documentation initialized at {docs_path}")
    print("\nNext steps:")
    print("  1. Edit docs/project-structure.md with your architecture")
    print("  2. Update docs/tracking/roadmap.md with milestones")
    print("  3. Create BRD: python3 scripts/add_brd.py <project-name>")
    print("  4. Create sprint: python3 scripts/add_sprint.py 01")
    print("  5. Add user story modules: python3 scripts/add_module.py <name>")


if __name__ == "__main__":
    main()
