#!/usr/bin/env python3
"""
Create a new sprint tracking document.

Usage:
  python3 add_sprint.py <number> [--start DATE] [--end DATE] [--path ./docs]

Examples:
  python3 add_sprint.py 02
  python3 add_sprint.py 02 --start 2026-02-10 --end 2026-02-21
"""

import argparse
import os
from datetime import datetime, timedelta
from pathlib import Path

SKILL_DIR = Path(__file__).parent.parent
TEMPLATES_DIR = SKILL_DIR / "assets" / "templates"


def get_sprint_template() -> str:
    """Load sprint template."""
    template_path = TEMPLATES_DIR / "sprint.md"
    if template_path.exists():
        return template_path.read_text()
    return "# Sprint [XX]\n\nTODO: Add content\n"


def format_date(date_str: str | None, default: datetime) -> str:
    """Parse date string or use default."""
    if date_str:
        return date_str
    return default.strftime("%Y-%m-%d")


def main():
    parser = argparse.ArgumentParser(description="Create new sprint document")
    parser.add_argument("number", help="Sprint number (e.g., 02)")
    parser.add_argument("--start", help="Start date (YYYY-MM-DD)")
    parser.add_argument("--end", help="End date (YYYY-MM-DD)")
    parser.add_argument("--path", default="./docs", help="Docs path")
    parser.add_argument("--name", default="", help="Sprint name/theme")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()
    sprints_path = docs_path / "tracking" / "sprints"

    # Ensure sprints directory exists
    if not sprints_path.exists():
        sprints_path.mkdir(parents=True)
        print(f"  Created: {sprints_path}")

    # Format sprint number
    sprint_num = args.number.zfill(2)
    sprint_file = sprints_path / f"sprint-{sprint_num}.md"

    if sprint_file.exists():
        print(f"❌ Sprint file already exists: {sprint_file}")
        return

    # Calculate dates
    today = datetime.now()
    start_date = format_date(args.start, today)

    if args.end:
        end_date = args.end
    else:
        # Default 2-week sprint
        start_dt = datetime.strptime(start_date, "%Y-%m-%d")
        end_dt = start_dt + timedelta(days=13)  # 2 weeks - 1 day
        end_date = end_dt.strftime("%Y-%m-%d")

    sprint_name = args.name or f"Sprint {sprint_num}"

    print(f"\n📅 Creating Sprint {sprint_num}\n")

    # Get template and replace placeholders
    content = get_sprint_template()
    content = content.replace("[XX]", sprint_num)
    content = content.replace("[Sprint Name]", sprint_name)
    content = content.replace("| **Start Date** | [YYYY-MM-DD] |", f"| **Start Date** | {start_date} |")
    content = content.replace("| **End Date** | [YYYY-MM-DD] |", f"| **End Date** | {end_date} |")
    content = content.replace("| **Status** | 🔨 Active / ✅ Complete / ⏳ Planned |", "| **Status** | ⏳ Planned |")

    # Write sprint file
    sprint_file.write_text(content)
    print(f"  Created: {sprint_file}")

    # Update current symlink
    current_link = sprints_path / "current.md"
    if current_link.is_symlink():
        current_link.unlink()

    # Create relative symlink
    os.symlink(f"sprint-{sprint_num}.md", current_link)
    print(f"  Updated: {current_link} -> sprint-{sprint_num}.md")

    print(f"\n✅ Sprint {sprint_num} created")
    print(f"\nDetails:")
    print(f"  Start: {start_date}")
    print(f"  End:   {end_date}")
    print(f"\nNext steps:")
    print(f"  1. Edit {sprint_file} with sprint goal")
    print(f"  2. Add stories to sprint backlog")
    print(f"  3. Update story status in user-stories/")


if __name__ == "__main__":
    main()
