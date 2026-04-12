#!/usr/bin/env python3
"""
Generate bug report from E2E test failure.

Usage:
    python3 report_bug.py --test "test name" --file "file:line" --issue "description"
    python3 report_bug.py --from-trace test-results/folder/trace.zip

Creates a bug report in docs/bugs/FE-BUG-[DATE].md
"""

import os
import sys
import argparse
from datetime import datetime
from pathlib import Path

def get_project_root():
    """Find project root by looking for CLAUDE.md"""
    current = Path(__file__).resolve()
    for parent in current.parents:
        if (parent / "CLAUDE.md").exists():
            return parent
    return Path.cwd()

def create_bug_report(args):
    """Create a bug report file."""
    project_root = get_project_root()
    bugs_dir = project_root / "docs" / "bugs"
    bugs_dir.mkdir(parents=True, exist_ok=True)

    date = datetime.now().strftime("%Y-%m-%d")
    filename = f"FE-BUG-{date}.md"
    filepath = bugs_dir / filename

    # If file exists, append number
    counter = 1
    while filepath.exists():
        filename = f"FE-BUG-{date}-{counter}.md"
        filepath = bugs_dir / filename
        counter += 1

    content = f"""# Frontend Bug Report - {date}

## Bug: {args.title or 'E2E Test Failure'}

**Test**: `{args.file}` - {args.test}
**Location**: `frontend/src/[TODO: identify file]`
**Severity**: {args.severity or 'Medium'}

### Issue
{args.issue}

### Expected
[TODO: Describe expected behavior]

### Actual
[TODO: Describe actual behavior]

### Code Analysis
```typescript
// TODO: Add relevant code snippet
```

### Suggested Fix
```typescript
// TODO: Add suggested fix
```

---

## Next Steps

1. Review the bug details above
2. Use `/nextjs` skill to fix: `fix the bug in docs/bugs/{filename}`
"""

    with open(filepath, 'w') as f:
        f.write(content)

    print(f"Bug report created: {filepath}")
    print(f"\nTo fix, use: /nextjs fix the bug in docs/bugs/{filename}")
    return filepath

def main():
    parser = argparse.ArgumentParser(description='Generate bug report from E2E test failure')
    parser.add_argument('--test', required=True, help='Test name')
    parser.add_argument('--file', required=True, help='Test file:line')
    parser.add_argument('--issue', required=True, help='Issue description')
    parser.add_argument('--title', help='Bug title')
    parser.add_argument('--severity', choices=['Critical', 'High', 'Medium', 'Low'], default='Medium')

    args = parser.parse_args()
    create_bug_report(args)

if __name__ == '__main__':
    main()
