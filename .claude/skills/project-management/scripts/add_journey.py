#!/usr/bin/env python3
"""Generate journey test spec (.feature) for a user role."""

import argparse
import sys
from pathlib import Path

TEMPLATE = '''Feature: {role} User Journeys
  Complete user flows for {description}

  # {section1} Journey

  Scenario: {scenario1}
    Given I am logged in as "{role}"
    When [action]
    Then [expected result]

  Scenario: {scenario2}
    Given [precondition]
    When [action]
    Then [expected result]

  # {section2} Journey

  Scenario: {scenario3}
    Given [precondition]
    When [action]
    Then [expected result]

  # TODO: Add more user journey scenarios
  # Reference role permissions: prd/features/permissions.md
'''

ROLE_INFO = {
    'admin': {
        'description': 'tenant administrators',
        'section1': 'Tenant Setup',
        'section2': 'User Management',
        'scenario1': 'New tenant onboarding',
        'scenario2': 'Create organization structure',
        'scenario3': 'Onboard new team member',
    },
    'manager': {
        'description': 'department managers',
        'section1': 'Team Management',
        'section2': 'Document Oversight',
        'scenario1': 'View team dashboard',
        'scenario2': 'Manage department users',
        'scenario3': 'Review team documents',
    },
    'approver': {
        'description': 'document approvers',
        'section1': 'Approval Queue',
        'section2': 'Document Review',
        'scenario1': 'Check pending approvals',
        'scenario2': 'Review and approve document',
        'scenario3': 'Reject with feedback',
    },
    'editor': {
        'description': 'document editors',
        'section1': 'Document Creation',
        'section2': 'Version Management',
        'scenario1': 'Upload new document',
        'scenario2': 'Organize documents in folders',
        'scenario3': 'Update document with new version',
    },
    'viewer': {
        'description': 'read-only users',
        'section1': 'Document Discovery',
        'section2': 'Document Viewing',
        'scenario1': 'Search for documents',
        'scenario2': 'Browse by department',
        'scenario3': 'View and download document',
    },
}

def find_project_root() -> Path:
    """Find project root by looking for docs/ directory."""
    current = Path.cwd()
    for parent in [current] + list(current.parents):
        if (parent / "docs" / "requirements").exists():
            return parent
    return current

def create_journey(role_name: str, project_root: Path) -> None:
    """Create journey test spec for a role."""
    role = role_name.lower()

    if role not in ROLE_INFO:
        print(f"Error: Unknown role '{role}'")
        print(f"Available roles: {', '.join(ROLE_INFO.keys())}")
        sys.exit(1)

    journey_dir = project_root / "docs/requirements/test-specs/journeys"
    journey_path = journey_dir / f"{role}.feature"

    if journey_path.exists():
        print(f"Journey already exists: {journey_path}")
        response = input("Overwrite? (y/N): ")
        if response.lower() != 'y':
            print("Aborted.")
            sys.exit(0)

    # Get role info
    info = ROLE_INFO[role]

    # Create journey file
    journey_dir.mkdir(parents=True, exist_ok=True)
    content = TEMPLATE.format(
        role=role.title(),
        **info
    )

    journey_path.write_text(content)
    print(f"Created: {journey_path}")
    print(f"\nNext steps:")
    print(f"1. Add scenarios for key user flows")
    print(f"2. Cover happy path and error cases")
    print(f"3. Include role-specific permissions tests")

def main():
    parser = argparse.ArgumentParser(
        description="Generate journey test spec for a user role"
    )
    parser.add_argument(
        "role",
        help="Role name (admin, manager, approver, editor, viewer)"
    )
    args = parser.parse_args()

    project_root = find_project_root()
    create_journey(args.role, project_root)

if __name__ == "__main__":
    main()
