#!/usr/bin/env python3
"""
Pull all Forge skills to .claude/skills/ directory.
Usage: python3 scripts/pull-skills.py
"""

import json
import os
import urllib.request
import urllib.error
import sys

FORGE_API = "https://forge-api.sidcorp.co/api"
SKILLS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), ".claude", "skills")

# All skill names from forge
SKILLS = [
    "forge-issue", "forge-code", "forge-fix", "forge-plan", "forge-review",
    "pr-review", "forge-triage", "forge-clarify", "forge-release", "forge-staging",
    "forge-test", "forge-skill", "lessons-learned", "issue-creation", "nextjs",
    "strapi", "brand-assets", "tester", "chat-eval", "frontend-design",
    "e2e-playwright", "strapi-server", "skill-creator", "integration",
    "vercel-react-best-practices", "agent-optimize", "research-content-writer",
    "project-setup", "preview-deploy", "project-management",
    "web-design-guidelines", "antigravity-guide",
]


def fetch_skill(name):
    """Fetch a skill from Forge API."""
    url = f"{FORGE_API}/skills/{name}"
    req = urllib.request.Request(url, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        print(f"  ERROR {e.code}: {e.reason}")
        return None
    except Exception as e:
        print(f"  ERROR: {e}")
        return None


def write_skill(skill_dir, skill_md, files):
    """Write SKILL.md and associated files."""
    os.makedirs(skill_dir, exist_ok=True)

    # Write SKILL.md
    if skill_md:
        with open(os.path.join(skill_dir, "SKILL.md"), "w", encoding="utf-8") as f:
            f.write(skill_md)

    # Write bundled files
    for file_info in (files or []):
        file_path = os.path.join(skill_dir, file_info["path"])
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        content = file_info.get("content", "")
        encoding = file_info.get("encoding", "utf8")
        with open(file_path, "w", encoding="utf-8") as f:
            f.write(content)


def main():
    os.makedirs(SKILLS_DIR, exist_ok=True)
    total = len(SKILLS)
    success = 0
    skipped = 0

    for i, name in enumerate(SKILLS, 1):
        print(f"[{i}/{total}] Pulling {name}...", end=" ")
        data = fetch_skill(name)
        if not data:
            print("SKIP (fetch failed)")
            skipped += 1
            continue

        skill_md = data.get("skillMd", "")
        files = data.get("files", [])

        if not skill_md and not files:
            print("SKIP (empty)")
            skipped += 1
            continue

        skill_dir = os.path.join(SKILLS_DIR, name)
        write_skill(skill_dir, skill_md, files)

        file_count = len(files) if files else 0
        print(f"OK (SKILL.md + {file_count} files)")
        success += 1

    print(f"\nDone: {success} pulled, {skipped} skipped, {total} total")


if __name__ == "__main__":
    main()
