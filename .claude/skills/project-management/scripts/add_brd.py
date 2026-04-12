#!/usr/bin/env python3
"""
Create a Business Requirements Document (BRD).

Usage:
  python3 add_brd.py <name> [--path ./docs]

Examples:
  python3 add_brd.py employee-portal
  python3 add_brd.py hrms-system --path ./my-project/docs
"""

import argparse
from datetime import datetime
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates"


def kebab_to_title(s: str) -> str:
    """Convert kebab-case to Title Case."""
    return " ".join(word.capitalize() for word in s.split("-"))


def get_brd_template() -> str:
    """Load BRD template."""
    template_path = TEMPLATES_DIR / "brd.md"
    if template_path.exists():
        return template_path.read_text()
    return "# Business Requirements Document\n\nTODO: Add content\n"


def main():
    parser = argparse.ArgumentParser(description="Create BRD document")
    parser.add_argument("name", help="Project/feature name (kebab-case)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    brd_path = docs_path / "requirements" / "brd"

    # Ensure BRD directory exists
    if not brd_path.exists():
        brd_path.mkdir(parents=True)
        print(f"  Created: {brd_path}")

    # Create BRD file
    brd_file = brd_path / f"{args.name}-brd.md"

    if brd_file.exists():
        print(f"❌ BRD already exists: {brd_file}")
        return

    title = kebab_to_title(args.name)
    today = datetime.now().strftime("%Y-%m-%d")

    print(f"\n📄 Creating BRD: {title}\n")

    # Get template and replace placeholders
    content = get_brd_template()
    content = content.replace("[Project Name]", title)
    content = content.replace("| **Created** | [Date] |", f"| **Created** | {today} |")
    content = content.replace("| **Last Updated** | [Date] |", f"| **Last Updated** | {today} |")

    # Write BRD file
    brd_file.write_text(content)
    print(f"  Created: {brd_file}")

    print(f"\n✅ BRD '{args.name}' created")
    print(f"\nNext steps:")
    print(f"  1. Fill in executive summary and objectives")
    print(f"  2. Define scope and requirements")
    print(f"  3. Get stakeholder approval")
    print(f"  4. Create PRD: python3 scripts/add_prd.py {args.name}")


if __name__ == "__main__":
    main()
