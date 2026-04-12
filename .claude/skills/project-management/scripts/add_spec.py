#!/usr/bin/env python3
"""
Initialize modular Technical Specification structure.

Usage:
  python3 add_spec.py <name> [--path ./docs]

Examples:
  python3 add_spec.py docuvault
  python3 add_spec.py my-project --path ./my-project/docs
"""

import argparse
import shutil
from datetime import datetime
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates" / "tech-specs"


def kebab_to_title(s: str) -> str:
    """Convert kebab-case to Title Case."""
    return " ".join(word.capitalize() for word in s.split("-"))


def copy_template(src: Path, dst: Path, replacements: dict):
    """Copy template file with replacements."""
    content = src.read_text()
    for old, new in replacements.items():
        content = content.replace(old, new)
    dst.write_text(content)


def main():
    parser = argparse.ArgumentParser(description="Initialize tech spec structure")
    parser.add_argument("name", help="Project name (kebab-case)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    specs_path = docs_path / "requirements" / "tech-specs"
    core_path = specs_path / "core"
    modules_path = specs_path / "modules"

    # Check if already initialized
    if (specs_path / "README.md").exists():
        print(f"❌ Tech specs already initialized at: {specs_path}")
        print("   Use add_spec_module.py to add new modules")
        return

    title = kebab_to_title(args.name)
    today = datetime.now().strftime("%Y-%m-%d")

    print(f"\n📄 Initializing Tech Spec: {title}\n")

    # Create directories
    for path in [specs_path, core_path, modules_path]:
        path.mkdir(parents=True, exist_ok=True)
        print(f"  Created: {path}")

    # Replacements
    replacements = {
        "[Project Name]": title,
        "[project]": args.name,
        "[Date]": today,
    }

    # Copy README
    copy_template(
        TEMPLATES_DIR / "README.md",
        specs_path / "README.md",
        replacements
    )
    print(f"  Created: {specs_path / 'README.md'}")

    # Copy core specs
    for template in ["architecture.md", "data-model.md", "security.md", "infrastructure.md"]:
        src = TEMPLATES_DIR / "core" / template
        if src.exists():
            copy_template(src, core_path / template, replacements)
            print(f"  Created: {core_path / template}")

    print(f"\n✅ Tech Spec structure initialized for '{args.name}'")
    print(f"\nNext steps:")
    print(f"  1. Edit core specs in {core_path}")
    print(f"  2. Add module specs with: python3 add_spec_module.py <module-name>")
    print(f"  3. Update README.md with decisions")


if __name__ == "__main__":
    main()
