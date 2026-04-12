#!/usr/bin/env python3
"""
Add a feature to modular PRD.

Usage:
  python3 add_prd_feature.py <name> [--path ./docs]

Examples:
  python3 add_prd_feature.py auth
  python3 add_prd_feature.py documents
  python3 add_prd_feature.py workflow --path ./my-project/docs
"""

import argparse
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates" / "prd"


def kebab_to_title(s: str) -> str:
    """Convert kebab-case to Title Case."""
    return " ".join(word.capitalize() for word in s.split("-"))


def get_feature_template() -> str:
    """Load feature template."""
    template_path = TEMPLATES_DIR / "features" / "feature.md"
    if template_path.exists():
        return template_path.read_text()
    return "# [Feature Name] Features\n\nTODO: Add content\n"


def update_readme_index(readme_path: Path, feature_name: str, title: str):
    """Add feature to README index."""
    if not readme_path.exists():
        return

    content = readme_path.read_text()

    # Find the features table and add new entry
    new_entry = f"| [{feature_name}.md](features/{feature_name}.md) | {title} |"

    # Check if already in index
    if f"features/{feature_name}.md" in content:
        print(f"  Feature already in README index")
        return

    # Find where to insert (after last feature entry)
    lines = content.split("\n")
    insert_idx = None

    for i, line in enumerate(lines):
        if line.startswith("| [") and "features/" in line:
            insert_idx = i + 1

    if insert_idx:
        lines.insert(insert_idx, new_entry)
        readme_path.write_text("\n".join(lines))
        print(f"  Updated: {readme_path}")


def main():
    parser = argparse.ArgumentParser(description="Add PRD feature")
    parser.add_argument("name", help="Feature name (kebab-case)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    prd_path = docs_path / "requirements" / "prd"
    features_path = prd_path / "features"

    # Check if PRD initialized
    if not prd_path.exists():
        print(f"❌ PRD not initialized. Run add_prd.py first")
        return

    # Create features dir if needed
    features_path.mkdir(parents=True, exist_ok=True)

    # Create feature file
    feature_file = features_path / f"{args.name}.md"

    if feature_file.exists():
        print(f"❌ Feature already exists: {feature_file}")
        return

    title = kebab_to_title(args.name)

    print(f"\n📄 Creating Feature: {title}\n")

    # Get template and replace placeholders
    content = get_feature_template()
    content = content.replace("[Feature Name]", title)
    content = content.replace("[feature]", args.name)

    # Write feature file
    feature_file.write_text(content)
    print(f"  Created: {feature_file}")

    # Update README index
    update_readme_index(prd_path / "README.md", args.name, title)

    print(f"\n✅ Feature '{args.name}' created")
    print(f"\nNext steps:")
    print(f"  1. Define feature description")
    print(f"  2. Add user flow")
    print(f"  3. Document acceptance criteria")


if __name__ == "__main__":
    main()
