#!/usr/bin/env python3
"""
Unified Test Coverage Checker for HRM Platform

Analyzes backend routes and frontend pages to find untested code.
- Backend: API endpoint coverage (Jest)
- Frontend: Page/flow coverage (Playwright E2E)
"""

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Literal


Target = Literal['be', 'fe', 'all']


def get_project_root() -> Path:
    """Find the project root."""
    current = Path.cwd()
    if (current / 'backend').exists() or (current / 'frontend').exists():
        return current
    for parent in current.parents:
        if (parent / 'backend').exists() or (parent / 'frontend').exists():
            return parent
    return current


def detect_target() -> Target:
    """Auto-detect target based on current directory."""
    cwd = Path.cwd()
    cwd_str = str(cwd)

    if '/backend' in cwd_str or cwd_str.endswith('backend'):
        return 'be'
    elif '/frontend' in cwd_str or cwd_str.endswith('frontend'):
        return 'fe'
    elif '/e2e' in cwd_str or cwd_str.endswith('e2e'):
        return 'fe'

    return 'be'


# =============================================================================
# Backend Coverage
# =============================================================================

def get_backend_routes(backend_path: Path) -> list[dict]:
    """Extract all routes from backend API."""
    routes = []
    api_dir = backend_path / 'src' / 'api'

    if not api_dir.exists():
        return routes

    for api in api_dir.iterdir():
        if not api.is_dir():
            continue

        routes_dir = api / 'routes'
        if not routes_dir.exists():
            continue

        for route_file in routes_dir.glob('*.ts'):
            content = route_file.read_text()

            # Check for core router (auto-generates CRUD routes)
            if 'createCoreRouter' in content:
                resource = api.name
                endpoint = resource.replace('-', '-') + 's'
                routes.extend([
                    {'method': 'GET', 'path': f'/api/{endpoint}', 'resource': resource, 'handler': 'find'},
                    {'method': 'GET', 'path': f'/api/{endpoint}/:id', 'resource': resource, 'handler': 'findOne'},
                    {'method': 'POST', 'path': f'/api/{endpoint}', 'resource': resource, 'handler': 'create'},
                    {'method': 'PUT', 'path': f'/api/{endpoint}/:id', 'resource': resource, 'handler': 'update'},
                    {'method': 'DELETE', 'path': f'/api/{endpoint}/:id', 'resource': resource, 'handler': 'delete'},
                ])

            # Extract custom routes
            route_pattern = r"method:\s*['\"](\w+)['\"].*?path:\s*['\"]([^'\"]+)['\"].*?handler:\s*['\"]([^'\"]+)['\"]"
            matches = re.findall(route_pattern, content, re.DOTALL)

            for method, path, handler in matches:
                if not path.startswith('/api/'):
                    path = '/api' + path if path.startswith('/') else f'/api/{path}'

                routes.append({
                    'method': method.upper(),
                    'path': path,
                    'resource': api.name,
                    'handler': handler,
                })

    return routes


def get_tested_backend_endpoints(tests_path: Path) -> set[tuple[str, str]]:
    """Extract tested endpoints from backend test files."""
    tested = set()

    if not tests_path.exists():
        return tested

    for test_file in tests_path.glob('**/*.test.ts'):
        content = test_file.read_text()

        patterns = [
            r"\.(get|post|put|delete|patch)\(['\"]([^'\"]+)['\"]",
            r"\.(get|post|put|delete|patch)\(`([^`]+)`",
            r"(GET|POST|PUT|DELETE|PATCH)\s+['\"`]?(/api/[^'\"` ]+)",
        ]

        for pattern in patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            for method, path in matches:
                path = re.sub(r'\$\{[^}]+\}', ':id', path)
                path = re.sub(r'/[a-f0-9-]{20,}', '/:id', path)
                tested.add((method.upper(), path))

    return tested


def paths_match(route_path: str, test_path: str) -> bool:
    """Check if a test path matches a route path."""
    route_parts = route_path.split('/')
    test_parts = test_path.split('/')

    if len(route_parts) != len(test_parts):
        return False

    for route_seg, test_seg in zip(route_parts, test_parts):
        if route_seg.startswith(':'):
            continue
        if route_seg != test_seg:
            return False

    return True


def check_backend_coverage() -> dict:
    """Check test coverage for backend API endpoints."""
    root = get_project_root()
    backend_path = root / 'backend'
    tests_path = backend_path / 'tests' / 'api'

    routes = get_backend_routes(backend_path)
    tested = get_tested_backend_endpoints(tests_path)

    covered = []
    uncovered = []

    for route in routes:
        method = route['method']
        route_path = route['path']
        is_covered = False

        for test_method, test_path in tested:
            if method == test_method and paths_match(route_path, test_path):
                is_covered = True
                break

        if is_covered:
            covered.append(route)
        else:
            uncovered.append(route)

    return {
        'type': 'backend',
        'total': len(routes),
        'covered': len(covered),
        'uncovered': len(uncovered),
        'coverage_pct': round(len(covered) / len(routes) * 100, 1) if routes else 0,
        'covered_items': covered,
        'uncovered_items': uncovered,
    }


# =============================================================================
# Lifecycle Hooks Coverage
# =============================================================================

def get_lifecycle_hooks(backend_path: Path) -> list[dict]:
    """Extract all lifecycle hooks from backend."""
    hooks = []
    hooks_dir = backend_path / 'src' / 'bootstrap' / 'hooks'

    if not hooks_dir.exists():
        return hooks

    for hook_file in hooks_dir.glob('*.ts'):
        content = hook_file.read_text()
        hook_name = hook_file.stem

        models = []

        # Pattern 1: Inline array - models: ['api::...', ...]
        inline_pattern = r"models:\s*\[([^\]]+)\]"
        inline_matches = re.findall(inline_pattern, content)
        for match in inline_matches:
            models.extend(re.findall(r"['\"]([^'\"]+)['\"]", match))

        # Pattern 2: Constant reference - models: CONSTANT_NAME
        # First find the constant reference
        const_ref_pattern = r"models:\s*([A-Z_][A-Z0-9_]*)"
        const_refs = re.findall(const_ref_pattern, content)
        for const_name in const_refs:
            # Find the constant definition
            const_def_pattern = rf"(?:const|let)\s+{const_name}\s*=\s*\[([^\]]+)\]"
            const_matches = re.findall(const_def_pattern, content)
            for match in const_matches:
                models.extend(re.findall(r"['\"]([^'\"]+)['\"]", match))

        # Extract lifecycle events (afterCreate, beforeUpdate, etc.)
        event_pattern = r"async\s+(afterCreate|beforeCreate|afterUpdate|beforeUpdate|afterDelete|beforeDelete)\s*\("
        events = re.findall(event_pattern, content)

        for model in models:
            for event in events:
                hooks.append({
                    'file': hook_name,
                    'model': model,
                    'event': event,
                    'key': f"{hook_name}:{model}:{event}",
                })

    return hooks


def get_tested_hooks(tests_path: Path) -> set[str]:
    """Extract tested hooks from test files by analyzing test descriptions and assertions."""
    tested = set()

    if not tests_path.exists():
        return tested

    # Look for hook integration test files
    hook_test_files = list(tests_path.glob('**/hooks*.test.ts')) + \
                      list(tests_path.glob('**/lifecycle*.test.ts')) + \
                      list(tests_path.glob('**/*-hooks.test.ts'))

    for test_file in hook_test_files:
        content = test_file.read_text()

        # Extract hook names from describe/test blocks
        # e.g., describe('audit-logging hooks', ...)
        hook_name_pattern = r"describe\(['\"]([^'\"]+)['\"]"
        hook_names = re.findall(hook_name_pattern, content)

        for name in hook_names:
            # Normalize to hook file name
            normalized = name.lower().replace(' hooks', '').replace(' ', '-')
            tested.add(normalized)

        # Also look for specific assertions about hook behavior
        # Audit logging: audit-log entries
        if 'audit-log' in content.lower() or 'auditlog' in content.lower():
            if any(phrase in content for phrase in ['afterCreate', 'afterUpdate', 'afterDelete', 'action:']):
                tested.add('audit-logging')

        # Email triggers
        if 'email' in content.lower() and any(phrase in content for phrase in ['queueEmail', 'email-log', 'emailLog']):
            tested.add('email-triggers')

        # Security hooks
        if 'isSuperAdmin' in content or 'security' in content.lower():
            tested.add('security')

        # Onboarding linked tasks
        if 'linked' in content.lower() and 'onboarding' in content.lower():
            tested.add('onboarding-linked-tasks')

    # Also check regular test files for hook side-effect testing
    for test_file in tests_path.glob('**/*.test.ts'):
        content = test_file.read_text()

        # Audit logging verification
        if re.search(r"audit-log|auditLog", content, re.IGNORECASE):
            if re.search(r"action.*['\"]create['\"]|action.*['\"]update['\"]|action.*['\"]delete['\"]", content):
                tested.add('audit-logging')

        # Email queue verification
        if 'email-log' in content or 'emailLog' in content:
            tested.add('email-triggers')

    return tested


def check_hooks_coverage() -> dict:
    """Check test coverage for lifecycle hooks."""
    root = get_project_root()
    backend_path = root / 'backend'
    tests_path = backend_path / 'tests'

    hooks = get_lifecycle_hooks(backend_path)
    tested_hook_files = get_tested_hooks(tests_path)

    # Group by hook file
    hook_files = {}
    for hook in hooks:
        file_name = hook['file']
        if file_name not in hook_files:
            hook_files[file_name] = {
                'name': file_name,
                'hooks': [],
                'is_covered': file_name in tested_hook_files,
            }
        hook_files[file_name]['hooks'].append(hook)

    covered = [h for h in hook_files.values() if h['is_covered']]
    uncovered = [h for h in hook_files.values() if not h['is_covered']]

    return {
        'type': 'hooks',
        'total': len(hook_files),
        'covered': len(covered),
        'uncovered': len(uncovered),
        'coverage_pct': round(len(covered) / len(hook_files) * 100, 1) if hook_files else 0,
        'covered_items': covered,
        'uncovered_items': uncovered,
    }


def print_hooks_report(result: dict, show_covered: bool = False):
    """Print lifecycle hooks coverage report."""
    print("\n" + "=" * 60)
    print("LIFECYCLE HOOKS COVERAGE REPORT")
    print("=" * 60)

    print(f"\nTotal hook files:   {result['total']}")
    print(f"Covered by tests:   {result['covered']}")
    print(f"Not covered:        {result['uncovered']}")
    print(f"Coverage:           {result['coverage_pct']}%")

    if result['uncovered_items']:
        print("\n" + "-" * 60)
        print("UNCOVERED HOOKS")
        print("-" * 60)

        for hook_file in result['uncovered_items']:
            print(f"\n{hook_file['name']}:")
            for hook in hook_file['hooks']:
                print(f"  {hook['model']} -> {hook['event']}")

    if show_covered and result['covered_items']:
        print("\n" + "-" * 60)
        print("COVERED HOOKS")
        print("-" * 60)

        for hook_file in result['covered_items']:
            print(f"\n{hook_file['name']}:")
            for hook in hook_file['hooks']:
                print(f"  {hook['model']} -> {hook['event']}")


# =============================================================================
# Frontend Coverage (E2E Pages)
# =============================================================================

def get_frontend_pages(frontend_path: Path) -> list[dict]:
    """Extract all pages from frontend App Router."""
    pages = []
    app_dir = frontend_path / 'src' / 'app'

    if not app_dir.exists():
        return pages

    for page_file in app_dir.glob('**/page.tsx'):
        path_parts = page_file.parent.relative_to(app_dir).parts

        # Filter out route groups (parentheses)
        clean_parts = [p for p in path_parts if not p.startswith('(')]
        route = '/' + '/'.join(clean_parts) if clean_parts else '/'

        # Determine page type based on route group
        route_group = None
        for part in path_parts:
            if part.startswith('(') and part.endswith(')'):
                route_group = part[1:-1]
                break

        pages.append({
            'route': route,
            'path': str(page_file.relative_to(frontend_path)),
            'group': route_group or 'public',
        })

    return pages


def get_tested_pages(e2e_path: Path) -> set[str]:
    """Extract tested pages from E2E test files."""
    tested = set()

    if not e2e_path.exists():
        return tested

    # Check for test files
    test_patterns = ['**/*.test.ts', '**/*.spec.ts']

    for pattern in test_patterns:
        for test_file in e2e_path.glob(pattern):
            content = test_file.read_text()

            # Extract page.goto() calls
            goto_pattern = r"page\.goto\(['\"`]([^'\"`]+)['\"`]"
            matches = re.findall(goto_pattern, content)

            for url in matches:
                # Normalize URL
                if url.startswith('http'):
                    # Extract path from full URL
                    from urllib.parse import urlparse
                    parsed = urlparse(url)
                    url = parsed.path

                # Clean up dynamic segments
                url = re.sub(r'/[a-f0-9-]{20,}', '/:id', url)
                url = re.sub(r'/\d+', '/:id', url)

                tested.add(url)

    return tested


def check_frontend_coverage() -> dict:
    """Check E2E test coverage for frontend pages."""
    root = get_project_root()
    frontend_path = root / 'frontend'
    e2e_path = root / 'e2e'

    pages = get_frontend_pages(frontend_path)
    tested = get_tested_pages(e2e_path)

    covered = []
    uncovered = []

    for page in pages:
        route = page['route']
        is_covered = False

        # Check if route is tested
        for test_route in tested:
            # Normalize both routes
            norm_route = re.sub(r'/\[.*?\]', '/:id', route)
            norm_test = test_route

            if norm_route == norm_test or route == norm_test:
                is_covered = True
                break

        if is_covered:
            covered.append(page)
        else:
            uncovered.append(page)

    return {
        'type': 'frontend-e2e',
        'total': len(pages),
        'covered': len(covered),
        'uncovered': len(uncovered),
        'coverage_pct': round(len(covered) / len(pages) * 100, 1) if pages else 0,
        'covered_items': covered,
        'uncovered_items': uncovered,
    }


# =============================================================================
# Reporting
# =============================================================================

def print_backend_report(result: dict, show_covered: bool = False):
    """Print backend coverage report."""
    print("\n" + "=" * 60)
    print("BACKEND API COVERAGE REPORT")
    print("=" * 60)

    print(f"\nTotal endpoints:    {result['total']}")
    print(f"Covered by tests:   {result['covered']}")
    print(f"Not covered:        {result['uncovered']}")
    print(f"Coverage:           {result['coverage_pct']}%")

    if result['uncovered_items']:
        print("\n" + "-" * 60)
        print("UNCOVERED ENDPOINTS")
        print("-" * 60)

        by_resource = {}
        for route in result['uncovered_items']:
            resource = route['resource']
            if resource not in by_resource:
                by_resource[resource] = []
            by_resource[resource].append(route)

        for resource in sorted(by_resource.keys()):
            print(f"\n{resource}:")
            for route in by_resource[resource]:
                print(f"  {route['method']:7} {route['path']}")

    if show_covered and result['covered_items']:
        print("\n" + "-" * 60)
        print("COVERED ENDPOINTS")
        print("-" * 60)

        by_resource = {}
        for route in result['covered_items']:
            resource = route['resource']
            if resource not in by_resource:
                by_resource[resource] = []
            by_resource[resource].append(route)

        for resource in sorted(by_resource.keys()):
            print(f"\n{resource}:")
            for route in by_resource[resource]:
                print(f"  {route['method']:7} {route['path']}")


def print_frontend_report(result: dict, show_covered: bool = False):
    """Print frontend E2E coverage report."""
    print("\n" + "=" * 60)
    print("FRONTEND E2E PAGE COVERAGE REPORT")
    print("=" * 60)

    print(f"\nTotal pages:        {result['total']}")
    print(f"Covered by tests:   {result['covered']}")
    print(f"Not covered:        {result['uncovered']}")
    print(f"Coverage:           {result['coverage_pct']}%")

    if result['uncovered_items']:
        print("\n" + "-" * 60)
        print("UNCOVERED PAGES")
        print("-" * 60)

        by_group = {}
        for page in result['uncovered_items']:
            group = page['group']
            if group not in by_group:
                by_group[group] = []
            by_group[group].append(page)

        for group in sorted(by_group.keys()):
            print(f"\n[{group}]:")
            for page in by_group[group]:
                print(f"  {page['route']}")
                print(f"    {page['path']}")

    if show_covered and result['covered_items']:
        print("\n" + "-" * 60)
        print("COVERED PAGES")
        print("-" * 60)

        by_group = {}
        for page in result['covered_items']:
            group = page['group']
            if group not in by_group:
                by_group[group] = []
            by_group[group].append(page)

        for group in sorted(by_group.keys()):
            print(f"\n[{group}]:")
            for page in by_group[group]:
                print(f"  {page['route']}")


def main():
    parser = argparse.ArgumentParser(
        description='Check test coverage for API endpoints and E2E pages',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                     Auto-detect and show uncovered
  %(prog)s --target be         Backend API coverage
  %(prog)s --target fe         Frontend E2E page coverage
  %(prog)s --all               Show covered items too
  %(prog)s --json              Output as JSON
        """
    )

    parser.add_argument('--target', '-t', choices=['be', 'fe', 'hooks', 'all'],
                       help='Target platform (default: auto-detect)')
    parser.add_argument('--all', '-a', action='store_true',
                       help='Show covered items too')
    parser.add_argument('--json', '-j', action='store_true',
                       help='Output as JSON')
    parser.add_argument('--include-hooks', action='store_true',
                       help='Include lifecycle hooks in backend coverage')

    args = parser.parse_args()
    target = args.target or detect_target()

    results = []

    if target in ('be', 'all'):
        results.append(check_backend_coverage())
        if args.include_hooks or target == 'all':
            results.append(check_hooks_coverage())

    if target == 'hooks':
        results.append(check_hooks_coverage())

    if target in ('fe', 'all'):
        results.append(check_frontend_coverage())

    if args.json:
        output = []
        for result in results:
            output.append({
                'type': result['type'],
                'total': result['total'],
                'covered': result['covered'],
                'uncovered': result['uncovered'],
                'coverage_pct': result['coverage_pct'],
                'uncovered_items': [
                    {'name': item.get('route') or f"{item.get('method')} {item.get('path')}",
                     'path': item.get('path', '')}
                    for item in result['uncovered_items']
                ],
            })
        print(json.dumps(output, indent=2))
    else:
        for result in results:
            if result['type'] == 'backend':
                print_backend_report(result, show_covered=args.all)
            elif result['type'] == 'hooks':
                print_hooks_report(result, show_covered=args.all)
            else:
                print_frontend_report(result, show_covered=args.all)

    print("\n" + "=" * 60)

    # Overall coverage
    total = sum(r['total'] for r in results)
    covered = sum(r['covered'] for r in results)
    overall = round(covered / total * 100, 1) if total else 0

    if len(results) > 1:
        print(f"\nOVERALL COVERAGE: {overall}% ({covered}/{total})")

    return 0 if overall >= 80 else 1


if __name__ == '__main__':
    sys.exit(main())
