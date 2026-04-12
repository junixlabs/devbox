#!/usr/bin/env python3
"""
Initialize modular PRD structure.

Usage:
  python3 add_prd.py <name> [--path ./docs]

Examples:
  python3 add_prd.py docuvault
  python3 add_prd.py my-project --path ./my-project/docs
"""

import argparse
from datetime import datetime
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates" / "prd"


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
    parser = argparse.ArgumentParser(description="Initialize PRD structure")
    parser.add_argument("name", help="Project name (kebab-case)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    prd_path = docs_path / "requirements" / "prd"
    features_path = prd_path / "features"

    # Check if already initialized
    if (prd_path / "README.md").exists():
        print(f"❌ PRD already initialized at: {prd_path}")
        print("   Use add_prd_feature.py to add new features")
        return

    title = kebab_to_title(args.name)
    today = datetime.now().strftime("%Y-%m-%d")

    print(f"\n📄 Initializing PRD: {title}\n")

    # Create directories
    for path in [prd_path, features_path]:
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
        prd_path / "README.md",
        replacements
    )
    print(f"  Created: {prd_path / 'README.md'}")

    # Copy ux.md and nfr.md
    for template in ["ux.md", "nfr.md"]:
        src = TEMPLATES_DIR / template
        if src.exists():
            copy_template(src, prd_path / template, replacements)
            print(f"  Created: {prd_path / template}")

    print(f"\n✅ PRD structure initialized for '{args.name}'")
    print(f"\nNext steps:")
    print(f"  1. Edit README.md with overview and goals")
    print(f"  2. Add features: python3 add_prd_feature.py <feature-name>")
    print(f"  3. Edit ux.md with screens and navigation")
    print(f"  4. Edit nfr.md with requirements")


if __name__ == "__main__":
    main()
