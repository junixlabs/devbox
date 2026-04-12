#!/usr/bin/env python3
"""
Add a coding standard to skill rules files.

With the modular structure, each rule is its own file (30-80 lines).

Usage:
    # Create a new rule file
    python3 add_standard.py --file "strapi/new-rule.md" \
        --title "New Rule" --wrong "bad code" --correct "good code" --why "explanation"

    # Add to existing rule file
    python3 add_standard.py --file "strapi/svc-modular.md" \
        --title "Additional Pattern" --wrong "..." --correct "..." --why "..."
"""

import argparse
from pathlib import Path


def find_project_root() -> Path:
    """Find project root by looking for .claude directory."""
    current = Path.cwd()
    while current != current.parent:
        if (current / '.claude' / 'skills').is_dir():
            return current
        current = current.parent
    return Path.cwd()


def get_skill_rules_path(file: str) -> Path:
    """Get path to skill rules file."""
    root = find_project_root()
    parts = file.split('/')
    if len(parts) != 2:
        return None
    stack, filename = parts
    return root / '.claude' / 'skills' / stack / 'rules' / filename


def detect_lang(file: str) -> str:
    """Detect language from file path."""
    if 'nextjs' in file or 'component' in file or 'hook' in file:
        return 'tsx'
    if 'strapi' in file:
        return 'typescript'
    return 'typescript'


def format_entry(title: str, wrong: str, correct: str, why: str, lang: str) -> str:
    """Format a standard entry with Wrong/Correct."""
    return f"""
**Incorrect:**
```{lang}
{wrong}
```

**Correct:**
```{lang}
{correct}
```

Why: {why}
"""


def format_new_file(title: str, wrong: str, correct: str, why: str, lang: str) -> str:
    """Format a new rule file."""
    return f"""# {title}

**Incorrect:**
```{lang}
{wrong}
```

**Correct:**
```{lang}
{correct}
```

Why: {why}
"""


def add_standard(file: str, title: str, wrong: str, correct: str, why: str) -> None:
    """Add a standard to a rule file."""
    filepath = get_skill_rules_path(file)

    if not filepath:
        print(f"Error: Invalid file path: {file}")
        print("Expected format: stack/rule-name.md (e.g., strapi/svc-modular.md)")
        return

    if not filepath.parent.exists():
        print(f"Error: Directory not found: {filepath.parent}")
        return

    lang = detect_lang(file)

    if filepath.exists():
        # Append to existing file
        text = filepath.read_text()
        entry = format_entry(title, wrong, correct, why, lang)
        new_text = text.rstrip() + f"\n\n## {title}\n{entry}"
        filepath.write_text(new_text)
        print(f"Appended to: {filepath}")
    else:
        # Create new file
        content = format_new_file(title, wrong, correct, why, lang)
        filepath.write_text(content)
        print(f"Created: {filepath}")

    print(f"Rule: {title}")


def main():
    parser = argparse.ArgumentParser(description='Add a Wrong/Correct coding standard')
    parser.add_argument('--file', required=True,
                        help='Target file (e.g., strapi/new-rule.md)')
    parser.add_argument('--title', required=True,
                        help='Rule title')
    parser.add_argument('--wrong', required=True,
                        help='Wrong code example')
    parser.add_argument('--correct', required=True,
                        help='Correct code example')
    parser.add_argument('--why', required=True,
                        help='Why the correct version is better')

    args = parser.parse_args()
    add_standard(args.file, args.title, args.wrong, args.correct, args.why)


if __name__ == '__main__':
    main()
