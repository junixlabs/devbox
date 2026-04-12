#!/usr/bin/env python3
"""
Check frontend-backend integration for mismatches and anti-patterns.

Usage:
    python3 check_integration.py [--fix] [--verbose]

Checks for:
1. Direct fetch() calls instead of API client usage
2. Calls to non-existent Next.js API routes (/api/...)
3. Frontend API endpoints not matching backend routes
4. Type field mismatches between frontend types and backend schemas
5. Enum value mismatches
6. Strapi response unwrapping issues (api.get<T[]> vs {data: T[]})
7. Form field coverage - detects when backend validation requires a field
   but frontend form doesn't include it (e.g., percentageOf required when
   calculationType='percentage' but form missing the Select)

Example:
    python3 check_integration.py
    python3 check_integration.py --verbose
"""

import argparse
import sys
from pathlib import Path

# Add scripts directory to path for imports
SCRIPTS_DIR = Path(__file__).parent
sys.path.insert(0, str(SCRIPTS_DIR))

from utils import find_project_root, CheckResult, Issue
from loaders import get_backend_routes, get_frontend_api_calls
from checks import (
    check_direct_fetch_calls,
    check_nextjs_api_routes,
    check_api_route_coverage,
    check_schema_type_match,
    check_response_unwrapping,
    check_form_field_coverage,
)


def print_results(result: CheckResult, verbose: bool = False):
    """Print check results in a readable format."""
    if not result.issues:
        print("✅ No integration issues found!")
        return

    errors = [i for i in result.issues if i.severity == 'error']
    warnings = [i for i in result.issues if i.severity == 'warning']
    infos = [i for i in result.issues if i.severity == 'info']

    print(f"\n{'='*60}")
    print("INTEGRATION CHECK RESULTS")
    print(f"{'='*60}")
    print(f"  Errors:   {len(errors)}")
    print(f"  Warnings: {len(warnings)}")
    print(f"  Info:     {len(infos)}")
    print(f"{'='*60}\n")

    # Group by category
    by_category: dict[str, list[Issue]] = {}
    for issue in result.issues:
        if issue.category not in by_category:
            by_category[issue.category] = []
        by_category[issue.category].append(issue)

    severity_icons = {'error': '❌', 'warning': '⚠️', 'info': 'ℹ️'}

    for category, issues in sorted(by_category.items()):
        print(f"\n## {category.upper().replace('-', ' ')}")
        print("-" * 40)

        for issue in issues:
            icon = severity_icons.get(issue.severity, '•')
            line_info = f":{issue.line}" if issue.line else ""
            print(f"\n{icon} [{issue.severity.upper()}] {issue.file}{line_info}")
            print(f"   {issue.message}")
            if issue.suggestion and (verbose or issue.severity == 'error'):
                print(f"   💡 {issue.suggestion}")

    print(f"\n{'='*60}")
    if errors:
        print(f"⚠️  Found {len(errors)} error(s) that should be fixed")
        sys.exit(1)


def main():
    parser = argparse.ArgumentParser(description='Check frontend-backend integration')
    parser.add_argument('--verbose', '-v', action='store_true', help='Show suggestions for all issues')
    parser.add_argument('--fix', action='store_true', help='Attempt to auto-fix issues (not implemented)')
    args = parser.parse_args()

    try:
        root = find_project_root()
    except RuntimeError as e:
        print(f"Error: {e}")
        sys.exit(1)

    backend_dir = root / 'backend'
    frontend_dir = root / 'frontend'

    print("🔍 Checking frontend-backend integration...")
    print(f"   Backend:  {backend_dir}")
    print(f"   Frontend: {frontend_dir}")

    result = CheckResult()

    # Run all checks
    print("\n📋 Running checks...")

    print("   • Checking for direct fetch() calls...")
    check_direct_fetch_calls(frontend_dir, result)

    print("   • Checking Next.js API routes...")
    check_nextjs_api_routes(frontend_dir, result)

    print("   • Loading backend routes...")
    backend_routes = get_backend_routes(backend_dir)

    print("   • Loading frontend API calls...")
    frontend_apis = get_frontend_api_calls(frontend_dir)

    print("   • Checking API route coverage...")
    check_api_route_coverage(backend_routes, frontend_apis, result)

    print("   • Checking schema/type alignment...")
    check_schema_type_match(backend_dir, frontend_dir, result)

    print("   • Checking Strapi response unwrapping...")
    check_response_unwrapping(frontend_dir, result)

    print("   • Checking form field coverage...")
    check_form_field_coverage(backend_dir, frontend_dir, result)

    print_results(result, args.verbose)


if __name__ == '__main__':
    main()
