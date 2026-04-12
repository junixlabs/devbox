#!/usr/bin/env python3
"""
Extract skills from MCP tool result files and write to .claude/skills/.
Reads all JSON files in the tool-results directory.
"""

import json
import os
import glob

RESULTS_DIR = os.path.expanduser(
    "~/.claude/projects/-home-kieutrung-tools-devbox/74436197-76d9-41da-847b-d7ee8e7a6fa5/tool-results"
)
SKILLS_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), ".claude", "skills")


def extract_skill_from_text(text):
    """Try to extract skill data from a text block."""
    try:
        data = json.loads(text)
        if isinstance(data, dict) and "name" in data and "skillMd" in data:
            return data
    except:
        pass
    return None


def process_file(filepath):
    """Process a single tool result file and extract skill data."""
    skills = []
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()

    # Try parsing as JSON array first
    try:
        arr = json.loads(content)
        if isinstance(arr, list):
            for item in arr:
                if isinstance(item, dict) and "text" in item:
                    skill = extract_skill_from_text(item["text"])
                    if skill:
                        skills.append(skill)
            return skills
    except:
        pass

    # Try as direct JSON object
    skill = extract_skill_from_text(content)
    if skill:
        skills.append(skill)

    return skills


def write_skill(name, skill_md, files):
    """Write a skill to disk."""
    skill_dir = os.path.join(SKILLS_DIR, name)
    os.makedirs(skill_dir, exist_ok=True)

    if skill_md:
        with open(os.path.join(skill_dir, "SKILL.md"), "w", encoding="utf-8") as f:
            f.write(skill_md)

    for file_info in (files or []):
        file_path = os.path.join(skill_dir, file_info["path"])
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        with open(file_path, "w", encoding="utf-8") as f:
            f.write(file_info.get("content", ""))


def main():
    os.makedirs(SKILLS_DIR, exist_ok=True)

    all_files = glob.glob(os.path.join(RESULTS_DIR, "*.json")) + glob.glob(os.path.join(RESULTS_DIR, "*.txt"))
    print(f"Found {len(all_files)} result files")

    all_skills = {}
    for fpath in sorted(all_files):
        fname = os.path.basename(fpath)
        skills = process_file(fpath)
        for s in skills:
            name = s["name"]
            if name not in all_skills or (s.get("skillMd") and not all_skills[name].get("skillMd")):
                all_skills[name] = s
        if skills:
            print(f"  {fname}: extracted {len(skills)} skill(s): {[s['name'] for s in skills]}")

    print(f"\nTotal unique skills: {len(all_skills)}")
    print()

    for name, data in sorted(all_skills.items()):
        skill_md = data.get("skillMd", "")
        files = data.get("files", [])
        if not skill_md and not files:
            print(f"  SKIP {name} (empty)")
            continue
        write_skill(name, skill_md, files)
        fc = len(files) if files else 0
        print(f"  OK {name} (SKILL.md + {fc} files)")

    print(f"\nDone!")


if __name__ == "__main__":
    main()
