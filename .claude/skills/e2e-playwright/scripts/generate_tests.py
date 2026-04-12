#!/usr/bin/env python3
"""
Generate E2E test file from user story.

Usage:
    python3 generate_tests.py <story-id>
    python3 generate_tests.py EMP-001
    python3 generate_tests.py lve-003

The script:
1. Finds the user story file in docs/user-stories/
2. Extracts acceptance criteria (AC) and test cases (TC)
3. Generates a test file template
"""

import os
import re
import sys
from pathlib import Path

# Module mapping from story prefix to test folder and frontend module
MODULE_MAP = {
    'EMP': ('04-employees', 'employees'),
    'LVE': ('05-operations', 'leave'),
    'ATT': ('05-operations', 'attendance'),
    'ORG': ('03-configuration', 'organization'),
    'SET': ('03-configuration', 'settings'),
    'PAY': ('05-operations', 'payroll'),
    'PRF': ('05-operations', 'performance'),
    'REC': ('05-operations', 'recruitment'),
    'TRN': ('05-operations', 'training'),
    'PLT': ('01-platform', 'platform'),
    'RPT': ('05-operations', 'reports'),
}


def find_project_root() -> Path:
    """Find project root by looking for CLAUDE.md or .claude directory."""
    current = Path.cwd()
    while current != current.parent:
        if (current / 'CLAUDE.md').exists() or (current / '.claude').exists():
            return current
        current = current.parent
    return Path.cwd()


def find_user_story(story_id: str, root: Path) -> Path | None:
    """Find user story file by ID."""
    story_id_lower = story_id.lower()
    user_stories_dir = root / 'docs' / 'user-stories'

    if not user_stories_dir.exists():
        return None

    # Search all subdirectories
    for story_file in user_stories_dir.rglob('*.md'):
        if story_id_lower in story_file.name.lower():
            return story_file

    return None


def extract_acceptance_criteria(content: str) -> list[dict]:
    """Extract acceptance criteria from user story content."""
    criteria = []

    # Look for acceptance criteria section
    ac_section = re.search(r'##\s*Acceptance Criteria(.*?)(?=##|\Z)', content, re.DOTALL | re.IGNORECASE)

    if ac_section:
        section_content = ac_section.group(1)

        # Find numbered items or bullet points
        items = re.findall(r'(?:AC-(\d+)|[-*]\s*\[[ x]\])\s*(.+?)(?=\n(?:AC-|\s*[-*]\s*\[|\Z))', section_content, re.DOTALL)

        for i, match in enumerate(items):
            if isinstance(match, tuple):
                num, desc = match
                criteria.append({
                    'id': f'AC-{num}' if num else f'AC-{i+1}',
                    'description': desc.strip().split('\n')[0]
                })
            else:
                criteria.append({
                    'id': f'AC-{i+1}',
                    'description': match.strip().split('\n')[0]
                })

    return criteria


def extract_test_cases(content: str) -> list[dict]:
    """Extract test cases from user story content."""
    test_cases = []

    # Look for test scenarios section
    tc_section = re.search(r'##\s*Test (?:Scenarios|Cases)(.*?)(?=##|\Z)', content, re.DOTALL | re.IGNORECASE)

    if tc_section:
        section_content = tc_section.group(1)

        # Find table rows with TC-XXX
        rows = re.findall(r'\|\s*(TC-\d+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|', section_content)

        for tc_id, scenario, expected in rows:
            test_cases.append({
                'id': tc_id,
                'scenario': scenario.strip(),
                'expected': expected.strip()
            })

    return test_cases


def extract_story_title(content: str) -> str:
    """Extract story title from content."""
    title_match = re.search(r'^#\s+(.+?)$', content, re.MULTILINE)
    if title_match:
        return title_match.group(1).strip()
    return 'Unknown'


def generate_test_file(story_id: str, title: str, criteria: list, test_cases: list, module_info: tuple) -> str:
    """Generate test file content."""
    test_folder, feature_module = module_info

    # Determine page object to use
    page_class = f'{feature_module.title().replace("-", "")}Page'

    template = f'''import {{ test, expect }} from '../fixtures/test-users';
// TODO: Import appropriate page objects
// import {{ {page_class} }} from '../pages';

/**
 * {title} E2E Tests
 * Based on User Story: {story_id}
 *
 * IMPORTANT: Before completing these tests:
 * 1. Read frontend implementation in frontend/src/app/(tenant)/{feature_module}/
 * 2. Verify selectors match actual UI elements
 * 3. Update page objects if needed
 */

test.describe('{title}', () => {{
    test.beforeEach(async ({{ page, hrAdminUser }}) => {{
        await hrAdminUser.login(page);
        // await page.goto('/{feature_module}');
    }});

'''

    # Generate tests from acceptance criteria
    if criteria:
        template += '    // ============================================\n'
        template += '    // Acceptance Criteria Tests\n'
        template += '    // ============================================\n\n'

        for ac in criteria[:5]:  # Limit to first 5 for template
            test_name = f"{story_id}-{ac['id']}: {ac['description'][:50]}"
            template += f'''    test('{test_name}', async ({{ page }}) => {{
        // GIVEN: User is on the {feature_module} page
        // TODO: Verify preconditions

        // WHEN: User performs action
        // TODO: Implement action

        // THEN: Expected result
        // TODO: Add assertions based on: {ac['description']}

        await page.screenshot({{ path: 'tests/screenshots/{story_id.lower()}-{ac['id'].lower()}.png' }});
    }});

'''

    # Generate tests from test cases
    if test_cases:
        template += '    // ============================================\n'
        template += '    // Test Scenarios from User Story\n'
        template += '    // ============================================\n\n'

        for tc in test_cases[:5]:  # Limit to first 5 for template
            test_name = f"{tc['id']}: {tc['scenario'][:50]}"
            template += f'''    test('{test_name}', async ({{ page }}) => {{
        // Scenario: {tc['scenario']}
        // Expected: {tc['expected']}

        // TODO: Implement test

        await page.screenshot({{ path: 'tests/screenshots/{tc['id'].lower()}.png' }});
    }});

'''

    template += '});\n'

    return template


def main():
    if len(sys.argv) < 2:
        print('Usage: python3 generate_tests.py <story-id>')
        print('Example: python3 generate_tests.py EMP-001')
        sys.exit(1)

    story_id = sys.argv[1].upper()
    root = find_project_root()

    print(f'Looking for user story: {story_id}')

    # Find user story file
    story_file = find_user_story(story_id, root)

    if not story_file:
        print(f'User story not found: {story_id}')
        print(f'   Searched in: {root / "docs" / "user-stories"}')
        sys.exit(1)

    print(f'Found: {story_file}')

    # Read content
    content = story_file.read_text()

    # Extract information
    title = extract_story_title(content)
    criteria = extract_acceptance_criteria(content)
    test_cases = extract_test_cases(content)

    print(f'Title: {title}')
    print(f'Found {len(criteria)} acceptance criteria')
    print(f'Found {len(test_cases)} test cases')

    # Determine module
    prefix = story_id.split('-')[0]
    module_info = MODULE_MAP.get(prefix, ('05-operations', 'feature'))

    # Generate test file
    test_content = generate_test_file(story_id, title, criteria, test_cases, module_info)

    # Determine output path
    test_folder = module_info[0]
    test_filename = f'{story_id.lower().replace("-", "_")}.spec.ts'
    output_dir = root / 'e2e' / 'tests' / test_folder
    output_path = output_dir / test_filename

    # Check if file exists
    if output_path.exists():
        print(f'Test file already exists: {output_path}')
        response = input('Overwrite? (y/N): ')
        if response.lower() != 'y':
            print('Cancelled.')
            sys.exit(0)

    # Create directory if needed
    output_dir.mkdir(parents=True, exist_ok=True)

    # Write file
    output_path.write_text(test_content)

    print(f'\nGenerated test file: {output_path}')
    print('\nNext steps:')
    print('1. Read frontend implementation to verify selectors')
    print('2. Update page objects if needed')
    print('3. Complete TODO items in the test file')
    print('4. Run tests: npx playwright test ' + str(output_path.relative_to(root)))


if __name__ == '__main__':
    main()
