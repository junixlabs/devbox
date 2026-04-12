#!/usr/bin/env python3
"""
E2E Test Coverage Checker

Analyzes frontend pages/components and checks which have e2e test coverage.
Also maps user stories to tests to identify gaps.

Usage:
    python3 coverage_check.py              # Full report
    python3 coverage_check.py --summary    # Quick summary
    python3 coverage_check.py --json       # JSON output
    python3 coverage_check.py --missing    # Only show missing tests
"""

import os
import re
import sys
import json
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class PageInfo:
    path: str
    route: str
    module: str
    has_test: bool = False
    test_files: list[str] = field(default_factory=list)
    test_count: int = 0


@dataclass
class UserStoryInfo:
    id: str
    title: str
    module: str
    file_path: str
    has_test: bool = False
    test_references: list[str] = field(default_factory=list)


@dataclass
class CoverageReport:
    pages: list[PageInfo] = field(default_factory=list)
    user_stories: list[UserStoryInfo] = field(default_factory=list)
    total_pages: int = 0
    tested_pages: int = 0
    total_stories: int = 0
    tested_stories: int = 0
    total_tests: int = 0


def find_project_root() -> Path:
    """Find project root by looking for CLAUDE.md or .claude directory."""
    current = Path.cwd()
    while current != current.parent:
        if (current / 'CLAUDE.md').exists() or (current / '.claude').exists():
            return current
        current = current.parent
    return Path.cwd()


def extract_route_from_path(page_path: Path, app_dir: Path) -> str:
    """Convert file path to route."""
    relative = page_path.relative_to(app_dir)
    parts = list(relative.parts)

    # Remove page.tsx
    if parts and parts[-1] == 'page.tsx':
        parts = parts[:-1]

    # Handle route groups (tenant), (auth), etc.
    route_parts = []
    for part in parts:
        if part.startswith('(') and part.endswith(')'):
            continue  # Skip route groups
        if part.startswith('[') and part.endswith(']'):
            route_parts.append(f':{part[1:-1]}')  # Convert [id] to :id
        else:
            route_parts.append(part)

    return '/' + '/'.join(route_parts) if route_parts else '/'


def get_module_from_route(route: str) -> str:
    """Extract module name from route."""
    parts = route.strip('/').split('/')
    if parts and parts[0]:
        return parts[0]
    return 'root'


def scan_frontend_pages(root: Path) -> list[PageInfo]:
    """Scan frontend/src/app for page.tsx files."""
    pages = []
    app_dir = root / 'frontend' / 'src' / 'app'

    if not app_dir.exists():
        return pages

    for page_file in app_dir.rglob('page.tsx'):
        # Skip special pages
        if 'not-found' in str(page_file) or 'error' in str(page_file):
            continue

        route = extract_route_from_path(page_file, app_dir)
        module = get_module_from_route(route)

        pages.append(PageInfo(
            path=str(page_file.relative_to(root)),
            route=route,
            module=module,
        ))

    return sorted(pages, key=lambda p: p.route)


def scan_e2e_tests(root: Path) -> dict:
    """Scan e2e tests and extract test info."""
    tests_dir = root / 'e2e' / 'tests'
    test_info = {
        'files': [],
        'tests_by_route': {},
        'tests_by_story': {},
        'total_tests': 0,
    }

    if not tests_dir.exists():
        return test_info

    for spec_file in tests_dir.rglob('*.spec.ts'):
        # Skip pages directory
        if 'pages' in spec_file.parts:
            continue

        content = spec_file.read_text()
        test_info['files'].append(str(spec_file.relative_to(root)))

        # Extract routes being tested (goto calls)
        routes = re.findall(r"(?:goto|navigateTo)\s*\(\s*['\"]([^'\"]+)['\"]", content)
        for route in routes:
            if route not in test_info['tests_by_route']:
                test_info['tests_by_route'][route] = []
            test_info['tests_by_route'][route].append(str(spec_file.name))

        # Extract user story references
        story_refs = re.findall(r'([A-Z]{2,4}-\d{3})', content)
        for ref in set(story_refs):
            if ref not in test_info['tests_by_story']:
                test_info['tests_by_story'][ref] = []
            test_info['tests_by_story'][ref].append(str(spec_file.name))

        # Count test cases
        test_count = len(re.findall(r"test\s*\(", content))
        test_info['total_tests'] += test_count

    return test_info


def scan_user_stories(root: Path) -> list[UserStoryInfo]:
    """Scan user stories directory."""
    stories = []
    stories_dir = root / 'docs' / 'user-stories'

    if not stories_dir.exists():
        return stories

    for story_file in stories_dir.rglob('*.md'):
        if story_file.name == 'README.md':
            continue

        content = story_file.read_text()

        # Extract story ID from filename or content
        story_id_match = re.search(r'([A-Z]{2,4}-\d{3})', story_file.name.upper())
        if not story_id_match:
            story_id_match = re.search(r'Story ID[:\s|]+([A-Z]{2,4}-\d{3})', content)

        if story_id_match:
            story_id = story_id_match.group(1)

            # Extract title
            title_match = re.search(r'^#\s+(.+?)$', content, re.MULTILINE)
            title = title_match.group(1) if title_match else story_file.stem

            # Get module from parent directory
            module = story_file.parent.name

            stories.append(UserStoryInfo(
                id=story_id,
                title=title[:50],
                module=module,
                file_path=str(story_file.relative_to(root)),
            ))

    return sorted(stories, key=lambda s: s.id)


def match_pages_to_tests(pages: list[PageInfo], test_info: dict) -> None:
    """Match pages to their tests."""
    for page in pages:
        route = page.route
        # Check direct route match
        if route in test_info['tests_by_route']:
            page.has_test = True
            page.test_files = test_info['tests_by_route'][route]
            page.test_count = len(page.test_files)
            continue

        # Check partial route match
        for tested_route, files in test_info['tests_by_route'].items():
            if route.rstrip('/') in tested_route or tested_route in route:
                page.has_test = True
                page.test_files.extend(files)

        page.test_files = list(set(page.test_files))
        page.test_count = len(page.test_files)


def match_stories_to_tests(stories: list[UserStoryInfo], test_info: dict) -> None:
    """Match user stories to their tests."""
    for story in stories:
        if story.id in test_info['tests_by_story']:
            story.has_test = True
            story.test_references = test_info['tests_by_story'][story.id]


def generate_report(root: Path) -> CoverageReport:
    """Generate the coverage report."""
    report = CoverageReport()

    # Scan everything
    report.pages = scan_frontend_pages(root)
    test_info = scan_e2e_tests(root)
    report.user_stories = scan_user_stories(root)

    # Match
    match_pages_to_tests(report.pages, test_info)
    match_stories_to_tests(report.user_stories, test_info)

    # Calculate totals
    report.total_pages = len(report.pages)
    report.tested_pages = sum(1 for p in report.pages if p.has_test)
    report.total_stories = len(report.user_stories)
    report.tested_stories = sum(1 for s in report.user_stories if s.has_test)
    report.total_tests = test_info['total_tests']

    return report


def print_summary(report: CoverageReport) -> None:
    """Print summary statistics."""
    page_pct = (report.tested_pages / report.total_pages * 100) if report.total_pages else 0
    story_pct = (report.tested_stories / report.total_stories * 100) if report.total_stories else 0

    print('\n' + '=' * 60)
    print('E2E TEST COVERAGE SUMMARY')
    print('=' * 60)
    print(f'\nPages:        {report.tested_pages}/{report.total_pages} ({page_pct:.1f}%)')
    print(f'User Stories: {report.tested_stories}/{report.total_stories} ({story_pct:.1f}%)')
    print(f'Total Tests:  {report.total_tests}')
    print()


def print_full_report(report: CoverageReport, show_missing_only: bool = False) -> None:
    """Print detailed report."""
    print_summary(report)

    # Group pages by module
    modules = {}
    for page in report.pages:
        if page.module not in modules:
            modules[page.module] = []
        modules[page.module].append(page)

    print('=' * 60)
    print('PAGE COVERAGE BY MODULE')
    print('=' * 60)

    for module in sorted(modules.keys()):
        pages = modules[module]
        tested = sum(1 for p in pages if p.has_test)
        pct = (tested / len(pages) * 100) if pages else 0

        print(f'\n{module.upper()} ({tested}/{len(pages)} - {pct:.0f}%)')
        print('-' * 40)

        for page in pages:
            if show_missing_only and page.has_test:
                continue

            status = 'OK' if page.has_test else 'MISSING'
            tests = f' [{page.test_count} tests]' if page.has_test else ''
            print(f'  [{status}] {page.route}{tests}')

    # User stories
    print('\n' + '=' * 60)
    print('USER STORY COVERAGE')
    print('=' * 60)

    story_modules = {}
    for story in report.user_stories:
        if story.module not in story_modules:
            story_modules[story.module] = []
        story_modules[story.module].append(story)

    for module in sorted(story_modules.keys()):
        stories = story_modules[module]
        tested = sum(1 for s in stories if s.has_test)
        pct = (tested / len(stories) * 100) if stories else 0

        print(f'\n{module.upper()} ({tested}/{len(stories)} - {pct:.0f}%)')
        print('-' * 40)

        for story in stories:
            if show_missing_only and story.has_test:
                continue

            status = 'OK' if story.has_test else 'MISSING'
            refs = f' [{len(story.test_references)} refs]' if story.has_test else ''
            print(f'  [{status}] {story.id}: {story.title[:40]}{refs}')

    print()


def print_json(report: CoverageReport) -> None:
    """Print report as JSON."""
    data = {
        'summary': {
            'total_pages': report.total_pages,
            'tested_pages': report.tested_pages,
            'page_coverage': round(report.tested_pages / report.total_pages * 100, 1) if report.total_pages else 0,
            'total_stories': report.total_stories,
            'tested_stories': report.tested_stories,
            'story_coverage': round(report.tested_stories / report.total_stories * 100, 1) if report.total_stories else 0,
            'total_tests': report.total_tests,
        },
        'pages': [
            {
                'route': p.route,
                'module': p.module,
                'has_test': p.has_test,
                'test_count': p.test_count,
            }
            for p in report.pages
        ],
        'user_stories': [
            {
                'id': s.id,
                'title': s.title,
                'module': s.module,
                'has_test': s.has_test,
                'test_references': s.test_references,
            }
            for s in report.user_stories
        ],
    }
    print(json.dumps(data, indent=2))


def main():
    args = sys.argv[1:]
    root = find_project_root()

    print(f'Project root: {root}')
    print('Scanning frontend pages and e2e tests...')

    report = generate_report(root)

    if '--json' in args:
        print_json(report)
    elif '--summary' in args:
        print_summary(report)
    elif '--missing' in args:
        print_full_report(report, show_missing_only=True)
    else:
        print_full_report(report)


if __name__ == '__main__':
    main()
