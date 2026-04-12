#!/usr/bin/env python3
"""
Check 403 Test Message Consistency

Validates that 403 tests follow tester skill rules:
- apiRequest() (no JWT) → expect "Forbidden" (Strapi permission layer)
- authRequest(jwt) (with JWT) → expect policy/controller message:
  - "Super admin access required" (is-super-admin policy)
  - "Access denied to this resource" (controller authorization)
  - "Feature not available" (feature gate)

Usage:
    python3 .claude/skills/tester/scripts/check_403.py
    python3 .claude/skills/tester/scripts/check_403.py --fix  # Show suggested fixes
"""

import os
import re
import sys
import argparse
from pathlib import Path
from typing import List, Tuple, Optional

# Valid 403 messages
VALID_MESSAGES = {
    'no_jwt': ['Forbidden'],  # Strapi permission layer (no JWT)
    'with_jwt': [
        'Super admin access required',      # is-super-admin policy
        'Access denied to this resource',   # controller authorization
        'Feature not available',            # feature gate (partial match)
        'Forbidden',                        # Some controllers call ctx.forbidden() without message
        'System roles cannot be modified',  # Custom business logic
        'System roles cannot be deleted',   # Custom business logic
    ]
}

class Issue:
    def __init__(self, file: str, line: int, request_type: str,
                 expected_msg: Optional[str], actual_check: str, context: str):
        self.file = file
        self.line = line
        self.request_type = request_type  # 'apiRequest' or 'authRequest'
        self.expected_msg = expected_msg
        self.actual_check = actual_check
        self.context = context

def find_test_files(base_path: str) -> List[Path]:
    """Find all test files."""
    test_dir = Path(base_path) / 'backend' / 'tests'
    if not test_dir.exists():
        test_dir = Path(base_path) / 'tests'
    return list(test_dir.rglob('*.test.ts'))

def extract_403_blocks(content: str, file_path: str) -> List[Issue]:
    """Extract test blocks that check for 403 status."""
    issues = []
    lines = content.split('\n')

    i = 0
    while i < len(lines):
        line = lines[i]

        # Look for toBe(403) or toContain(403)
        if 'toBe(403)' in line or 'toContain(403)' in line:
            # Search backward for request type (apiRequest or authRequest)
            request_type = None
            context_lines = []

            for j in range(max(0, i - 15), i + 1):
                context_lines.append(lines[j])
                if 'apiRequest()' in lines[j] or 'apiRequest(' in lines[j]:
                    if 'authRequest' not in lines[j]:  # Make sure it's not authRequest
                        request_type = 'apiRequest'
                elif 'authRequest(' in lines[j]:
                    request_type = 'authRequest'

            # Search forward for message check (within next 5 lines)
            message_check = None
            # Collect all lines in the search window
            search_window = ''
            for j in range(i + 1, min(len(lines), i + 6)):
                line_content = lines[j].strip()
                # Skip empty lines and comments for extraction but include in window
                if line_content and not line_content.startswith('//'):
                    search_window += ' ' + line_content

            # Check for array-based checks first (multi-line patterns)
            if 'toContain' in search_window and ('Forbidden' in search_window or
                'Super admin' in search_window or 'Access denied' in search_window or
                'not available' in search_window):
                message_check = 'ARRAY_CHECK'
            # Check for .toBe patterns
            elif 'message' in search_window or 'error?' in search_window:
                match = re.search(r"\.toBe\(['\"](.+?)['\"]\)", search_window)
                if match:
                    message_check = match.group(1)
                else:
                    match = re.search(r"\.toContain\(['\"](.+?)['\"]\)", search_window)
                    if match:
                        message_check = match.group(1)

            # Check for issues
            if request_type:
                has_issue = False

                if request_type == 'apiRequest' and message_check is None:
                    # No JWT request without message check
                    has_issue = True
                elif request_type == 'authRequest' and message_check is None:
                    # JWT request without message check
                    has_issue = True
                elif request_type == 'apiRequest' and message_check and message_check != 'Forbidden':
                    # No JWT request should expect "Forbidden"
                    has_issue = True

                if has_issue:
                    issues.append(Issue(
                        file=str(file_path),
                        line=i + 1,
                        request_type=request_type,
                        expected_msg=message_check,
                        actual_check=line.strip(),
                        context='\n'.join(context_lines[-5:])
                    ))

        i += 1

    return issues

def check_all_tests(base_path: str) -> List[Issue]:
    """Check all test files for 403 issues."""
    all_issues = []

    test_files = find_test_files(base_path)

    for test_file in test_files:
        try:
            content = test_file.read_text()
            issues = extract_403_blocks(content, test_file)
            all_issues.extend(issues)
        except Exception as e:
            print(f"Error reading {test_file}: {e}", file=sys.stderr)

    return all_issues

def print_report(issues: List[Issue], show_fix: bool = False):
    """Print the report of issues found."""
    if not issues:
        print("✓ All 403 tests have proper message verification")
        return

    print(f"\n{'='*60}")
    print(f"403 Message Check Report")
    print(f"Found {len(issues)} potential issues")
    print(f"{'='*60}\n")

    # Group by file
    by_file = {}
    for issue in issues:
        if issue.file not in by_file:
            by_file[issue.file] = []
        by_file[issue.file].append(issue)

    for file, file_issues in by_file.items():
        rel_path = file.split('tests/')[-1] if 'tests/' in file else file
        print(f"\n📁 tests/{rel_path}")
        print("-" * 50)

        for issue in file_issues:
            print(f"\n  Line {issue.line}: {issue.request_type}")
            print(f"  Check: {issue.actual_check}")

            if issue.expected_msg:
                print(f"  Current message: \"{issue.expected_msg}\"")
            else:
                print(f"  ⚠️  Missing message verification!")

            if show_fix:
                if issue.request_type == 'apiRequest':
                    print(f"  💡 Suggested: expect(response.body.error?.message).toBe('Forbidden');")
                else:
                    print(f"  💡 Suggested: Add message check based on route policy:")
                    print(f"      - is-super-admin policy → 'Super admin access required'")
                    print(f"      - controller auth → 'Access denied to this resource'")
                    print(f"      - feature gate → toContain('Feature not available')")

    print(f"\n{'='*60}")
    print("Summary:")
    print(f"  - Total issues: {len(issues)}")
    print(f"  - Files affected: {len(by_file)}")
    print(f"\nRules (from tester skill):")
    print("  1. apiRequest() (no JWT) → expect 'Forbidden'")
    print("  2. authRequest(jwt) → expect policy/controller message")
    print(f"{'='*60}\n")

def save_report(issues: List[Issue], output_path: str):
    """Save report to file."""
    with open(output_path, 'w') as f:
        f.write("403 Message Check Report\n")
        f.write("=" * 60 + "\n\n")

        for issue in issues:
            f.write(f"File: {issue.file}\n")
            f.write(f"Line: {issue.line}\n")
            f.write(f"Request: {issue.request_type}\n")
            f.write(f"Check: {issue.actual_check}\n")
            if issue.expected_msg:
                f.write(f"Message: {issue.expected_msg}\n")
            else:
                f.write("Message: MISSING\n")
            f.write("-" * 40 + "\n\n")

def main():
    parser = argparse.ArgumentParser(description='Check 403 test message consistency')
    parser.add_argument('--fix', action='store_true', help='Show suggested fixes')
    parser.add_argument('--output', '-o', help='Save report to file')
    args = parser.parse_args()

    # Find project root
    script_dir = Path(__file__).parent
    project_root = script_dir.parent.parent.parent.parent

    print(f"Scanning tests in: {project_root}")

    issues = check_all_tests(str(project_root))

    print_report(issues, show_fix=args.fix)

    if args.output:
        save_report(issues, args.output)
        print(f"Report saved to: {args.output}")

    # Exit with error code if issues found
    sys.exit(1 if issues else 0)

if __name__ == '__main__':
    main()
