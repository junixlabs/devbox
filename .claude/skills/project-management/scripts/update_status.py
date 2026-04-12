#!/usr/bin/env python3
"""
Update user story status.

Usage:
  python3 update_status.py <story-id> <status> [--pr NUMBER] [--reason TEXT] [--path ./docs]

Examples:
  python3 update_status.py DOC-001 in-progress
  python3 update_status.py DOC-001 review --pr 123
  python3 update_status.py DOC-001 done --pr 123
  python3 update_status.py DOC-001 deployed
  python3 update_status.py DOC-001 blocked --reason "Waiting for API"

Status options: backlog, sprint, in-progress, review, done, deployed, blocked
"""

import argparse
import re
from datetime import datetime
from pathlib import Path

STATUS_MAP = {
    "backlog": "📋 Backlog",
    "sprint": "🔜 Sprint",
    "in-progress": "🔨 In Progress",
    "review": "🔍 In Review",
    "done": "✅ Done",
    "deployed": "🚀 Deployed",
    "blocked": "🔴 Blocked",
}


def find_story_file(docs_path: Path, story_id: str) -> Path | None:
    """Find story file by ID."""
    user_stories_path = docs_path / "requirements" / "user-stories"

    if not user_stories_path.exists():
        return None

    # Search all module directories
    for module_dir in user_stories_path.iterdir():
        if module_dir.is_dir():
            for story_file in module_dir.glob("*.md"):
                if story_file.name == "README.md":
                    continue
                # Check if file starts with story ID prefix
                prefix = story_id.lower().replace("-", "")
                file_prefix = story_file.stem.split("-")[0]
                file_num = story_file.stem.split("-")[1] if "-" in story_file.stem else ""

                # Match ID pattern (e.g., doc-001 matches doc-001-*.md)
                story_id_lower = story_id.lower()
                if story_file.stem.startswith(story_id_lower):
                    return story_file

                # Also check inside file for Story ID
                content = story_file.read_text()
                if f"**Story ID** | {story_id}" in content:
                    return story_file

    return None


def update_story_status(
    file_path: Path,
    new_status: str,
    pr_number: str | None = None,
    reason: str | None = None
) -> bool:
    """Update status in story file."""
    content = file_path.read_text()

    status_emoji = STATUS_MAP.get(new_status, new_status)

    # Update status field
    # Pattern: | **Status** | <anything> |
    status_pattern = r'\| \*\*Status\*\* \| [^|]+ \|'
    if re.search(status_pattern, content):
        content = re.sub(status_pattern, f'| **Status** | {status_emoji} |', content)
    else:
        print(f"  ⚠️  Status field not found in standard format")
        return False

    # Update PR field if provided
    if pr_number:
        pr_pattern = r'\| \*\*PR\*\* \| [^|]+ \|'
        pr_value = f"[#{pr_number}](https://github.com/org/repo/pull/{pr_number})"
        if re.search(pr_pattern, content):
            content = re.sub(pr_pattern, f'| **PR** | {pr_value} |', content)

    # Update blocked reason if status is blocked
    if new_status == "blocked" and reason:
        blocked_pattern = r'\| \*\*Blocked\*\* \| [^|]+ \|'
        if re.search(blocked_pattern, content):
            content = re.sub(blocked_pattern, f'| **Blocked** | {reason} |', content)

    # Clear blocked reason if not blocked
    if new_status != "blocked":
        blocked_pattern = r'\| \*\*Blocked\*\* \| [^|]+ \|'
        if re.search(blocked_pattern, content):
            content = re.sub(blocked_pattern, '| **Blocked** | - |', content)

    # Update deployed date if deployed
    if new_status == "deployed":
        today = datetime.now().strftime("%Y-%m-%d")
        deployed_pattern = r'\| \*\*Deployed\*\* \| [^|]+ \|'
        if re.search(deployed_pattern, content):
            content = re.sub(deployed_pattern, f'| **Deployed** | {today} |', content)

    file_path.write_text(content)
    return True


def main():
    parser = argparse.ArgumentParser(description="Update story status")
    parser.add_argument("story_id", help="Story ID (e.g., DOC-001)")
    parser.add_argument("status", choices=list(STATUS_MAP.keys()), help="New status")
    parser.add_argument("--pr", help="PR number")
    parser.add_argument("--reason", help="Blocked reason")
    parser.add_argument("--path", default="./docs", help="Docs path")
    args = parser.parse_args()

    docs_path = Path(args.path).resolve()

    print(f"\n🔄 Updating {args.story_id} → {args.status}\n")

    # Find story file
    story_file = find_story_file(docs_path, args.story_id)

    if not story_file:
        print(f"❌ Story not found: {args.story_id}")
        print(f"\nSearched in: {docs_path / 'requirements' / 'user-stories'}")
        return

    print(f"  Found: {story_file}")

    # Update status
    if update_story_status(story_file, args.status, args.pr, args.reason):
        print(f"  Updated: {STATUS_MAP[args.status]}")

        if args.pr:
            print(f"  PR: #{args.pr}")

        if args.status == "blocked" and args.reason:
            print(f"  Reason: {args.reason}")

        if args.status == "deployed":
            print(f"  Deployed: {datetime.now().strftime('%Y-%m-%d')}")

        print(f"\n✅ Status updated successfully")
    else:
        print(f"\n❌ Failed to update status")


if __name__ == "__main__":
    main()
