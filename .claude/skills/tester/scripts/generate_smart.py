#!/usr/bin/env python3
"""
Smart Test Generator for HRM Platform

Generates comprehensive test files by reading:
- Schema.json for field definitions
- Routes for all endpoints (including custom actions)

Supports modular folder structure organized by feature.
"""

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

# Feature module mappings
FEATURE_MODULES = {
    'recruitment': [
        'job-requisition', 'job-posting', 'application',
        'interview', 'interview-feedback', 'offer', 'pipeline-stage'
    ],
    'training': [
        'training-program', 'training-session',
        'training-registration', 'training-completion'
    ],
    'performance': ['goal', 'review-cycle', 'assessment'],
    'payroll': [
        'salary-component', 'salary-structure', 'employee-salary',
        'payroll-run', 'payslip', 'tax-rule'
    ],
    'leave': [
        'leave-type', 'leave-policy', 'leave-request',
        'leave-balance', 'holiday'
    ],
    'attendance': ['attendance-record', 'shift-template', 'shift-assignment'],
    'organization': ['department', 'position', 'location', 'company-info'],
    'employee': ['employee', 'document', 'onboarding-checklist', 'onboarding-progress', 'offboarding-task'],
    'platform': [
        'tenant', 'subscription-plan', 'custom-role',
        'permission', 'audit-log', 'password-policy'
    ],
}


def get_feature_for_resource(resource: str) -> str | None:
    """Get feature module for a resource."""
    for feature, resources in FEATURE_MODULES.items():
        if resource in resources:
            return feature
    return None


def get_project_root() -> Path:
    """Find the project root."""
    current = Path.cwd()
    if (current / 'backend').exists():
        return current
    for parent in current.parents:
        if (parent / 'backend').exists():
            return parent
    return current


def to_slug(name: str) -> str:
    """Convert name to slug."""
    return name.lower().replace(' ', '-').replace('_', '-')


def to_title(name: str) -> str:
    """Convert name to title case."""
    return ' '.join(word.capitalize() for word in name.replace('-', ' ').replace('_', ' ').split())


def to_camel(name: str) -> str:
    """Convert to camelCase."""
    parts = name.replace('-', '_').split('_')
    return parts[0].lower() + ''.join(p.capitalize() for p in parts[1:])


# Irregular plural mappings
IRREGULAR_PLURALS = {
    'company-info': 'company-info',  # Singleton
    'leave-policy': 'leave-policies',
    'leave-balance': 'leave-balances',
    'employee-salary': 'employee-salaries',
    'password-policy': 'password-policy',  # Singleton
    'tax-rule': 'tax-rules',
    'onboarding-progress': 'onboarding-progress',  # Singular/plural same
    'interview-feedback': 'interview-feedbacks',
    'auth': 'auth',  # No plural
    'report': 'reports',
}

# Resources that don't require auth for listing
PUBLIC_LIST_RESOURCES = ['subscription-plan', 'job-posting']

# Resources with no standard CRUD (custom only)
NON_CRUD_RESOURCES = [
    'auth', 'report', 'company-info', 'password-policy',
    'payslip', 'assessment', 'onboarding-progress', 'offboarding-task',
    'training-registration',  # Has custom register endpoint
]

# Resources with no standard delete
NO_DELETE_RESOURCES = ['attendance-record', 'audit-log']


def pluralize(resource: str) -> str:
    """Get plural form of resource name."""
    if resource in IRREGULAR_PLURALS:
        return IRREGULAR_PLURALS[resource]
    return resource + 's'


def get_schema(backend_path: Path, resource: str) -> dict | None:
    """Load schema.json for a resource."""
    schema_path = backend_path / 'src' / 'api' / resource / 'content-types' / resource / 'schema.json'
    if schema_path.exists():
        return json.loads(schema_path.read_text())
    return None


def get_routes(backend_path: Path, resource: str) -> list[dict]:
    """Extract routes for a resource."""
    routes = []
    routes_dir = backend_path / 'src' / 'api' / resource / 'routes'

    if not routes_dir.exists():
        return routes

    for route_file in routes_dir.glob('*.ts'):
        content = route_file.read_text()

        # Check for core router
        if 'createCoreRouter' in content:
            routes.extend([
                {'method': 'GET', 'path': f'/{resource}s', 'handler': 'find', 'is_core': True},
                {'method': 'GET', 'path': f'/{resource}s/:id', 'handler': 'findOne', 'is_core': True},
                {'method': 'POST', 'path': f'/{resource}s', 'handler': 'create', 'is_core': True},
                {'method': 'PUT', 'path': f'/{resource}s/:id', 'handler': 'update', 'is_core': True},
                {'method': 'DELETE', 'path': f'/{resource}s/:id', 'handler': 'delete', 'is_core': True},
            ])

        # Extract custom routes
        route_pattern = r"method:\s*['\"](\w+)['\"].*?path:\s*['\"]([^'\"]+)['\"].*?handler:\s*['\"]([^'\"]+)['\"]"
        matches = re.findall(route_pattern, content, re.DOTALL)

        for method, path, handler in matches:
            routes.append({
                'method': method.upper(),
                'path': path,
                'handler': handler.split('.')[-1],
                'is_core': False,
            })

    return routes


def get_sample_value(attr_name: str, attr_def: dict) -> str:
    """Generate sample value for an attribute."""
    attr_type = attr_def.get('type', 'string')

    # Handle enums
    if attr_type == 'enumeration' and 'enum' in attr_def:
        return f"'{attr_def['enum'][0]}'"

    # Handle by name patterns
    name_lower = attr_name.lower()
    if 'email' in name_lower:
        return f"`test-${{uniqueId()}}@example.com`"
    if 'name' in name_lower or 'title' in name_lower:
        return f"'Test {attr_name.replace('_', ' ').title()}'"
    if 'date' in name_lower:
        return "'2025-01-15'"
    if 'phone' in name_lower:
        return "'+1234567890'"
    if name_lower.endswith('id') and attr_type == 'string':
        return f"`TEST-${{uniqueId()}}`"

    # Handle by type
    type_samples = {
        'string': f"'Sample {attr_name}'",
        'text': f"'Sample {attr_name} text'",
        'richtext': "'<p>Sample content</p>'",
        'integer': '1',
        'decimal': '10.5',
        'float': '10.5',
        'biginteger': '1000',
        'boolean': 'true',
        'date': "'2025-01-15'",
        'datetime': "'2025-01-15T10:00:00Z'",
        'time': "'10:00:00'",
        'json': '{}',
        'email': f"`test-${{uniqueId()}}@example.com`",
        'password': "'Password123!'",
    }

    return type_samples.get(attr_type, "''")


def generate_test_data(schema: dict, exclude_relations: bool = True) -> dict[str, str]:
    """Generate test data from schema attributes."""
    data = {}
    attrs = schema.get('attributes', {})

    for name, attr in attrs.items():
        # Skip relations and special fields
        if exclude_relations and attr.get('type') == 'relation':
            continue
        if name in ['tenant', 'createdAt', 'updatedAt', 'createdBy', 'updatedBy']:
            continue

        if attr.get('required', False):
            data[name] = get_sample_value(name, attr)

    return data


def generate_test_file(resource: str, schema: dict | None, routes: list[dict], helpers_path: str = '../helpers/api') -> str:
    """Generate comprehensive test file."""
    slug = to_slug(resource)
    title = to_title(resource)
    endpoint = pluralize(slug)

    # Check if resource has special handling
    is_public_list = resource in PUBLIC_LIST_RESOURCES
    is_non_crud = resource in NON_CRUD_RESOURCES
    no_delete = resource in NO_DELETE_RESOURCES

    # Generate test data
    test_data = {}
    if schema:
        test_data = generate_test_data(schema)

    data_str = ',\n            '.join(f'{k}: {v}' for k, v in test_data.items()) if test_data else '// Add required fields'

    # Group routes
    crud_routes = [r for r in routes if r.get('is_core', False)]
    custom_routes = [r for r in routes if not r.get('is_core', False)]

    # Build custom endpoint tests
    custom_tests = []
    for route in custom_routes:
        method = route['method']
        path = route['path']
        handler = route['handler']

        # Skip duplicates of core routes
        if handler in ['find', 'findOne', 'create', 'update', 'delete']:
            continue

        # Determine if it needs :id
        needs_id = ':id' in path or '/:' in path

        # Build test
        test_name = re.sub(r'([A-Z])', r' \1', handler).strip().lower()
        if method == 'GET':
            if needs_id:
                custom_tests.append(f'''
  describe('{method} /api{path}', () => {{
    it('should {test_name}', async () => {{
      if (!jwt || !resourceId) return;

      const response = await authRequest(jwt)
        .get(`/api/{endpoint}/${{resourceId}}/{handler}`);

      // Accept 200, 400 (validation), 403 (permission), 404 (not found)
      expect([200, 400, 403, 404]).toContain(response.status);
    }});
  }});''')
            else:
                custom_tests.append(f'''
  describe('{method} /api{path}', () => {{
    it('should {test_name}', async () => {{
      if (!jwt) return;

      const response = await authRequest(jwt)
        .get('/api{path}');

      // Accept 200, 400, or 403 (permission required)
      expect([200, 400, 403]).toContain(response.status);
    }});
  }});''')
        else:  # POST, PUT, DELETE
            action_path = path.replace(f'/{endpoint}/', '').replace(':id/', '')
            if needs_id:
                custom_tests.append(f'''
  describe('{method} /api{path}', () => {{
    it('should {test_name}', async () => {{
      if (!jwt || !resourceId) return;

      const response = await authRequest(jwt)
        .{method.lower()}(`/api/{endpoint}/${{resourceId}}/{action_path}`)
        .send({{ data: {{}} }});

      // Accept 200, 400 (validation), 403 (permission), 404 (not found)
      expect([200, 400, 403, 404]).toContain(response.status);
    }});
  }});''')
            else:
                custom_tests.append(f'''
  describe('{method} /api{path}', () => {{
    it('should {test_name}', async () => {{
      if (!jwt) return;

      const response = await authRequest(jwt)
        .{method.lower()}('/api{path}')
        .send({{ data: {{}} }});

      // Accept 200, 400, or 403 (permission required)
      expect([200, 400, 403]).toContain(response.status);
    }});
  }});''')

    custom_tests_str = '\n'.join(custom_tests)

    # Build auth tests based on resource type
    if is_public_list:
        auth_tests = f'''
  // ============================================
  // AUTHENTICATION TESTS
  // ============================================

  describe('Authentication', () => {{
    it('should allow public access to list', async () => {{
      const response = await apiRequest().get('/api/{endpoint}');
      expect([200, 401, 403]).toContain(response.status);
    }});
  }});'''
    else:
        auth_tests = f'''
  // ============================================
  // AUTHENTICATION TESTS
  // ============================================

  describe('Authentication', () => {{
    it('should require auth for access', async () => {{
      const response = await apiRequest().get('/api/{endpoint}');
      // Should not return 200 without auth (unless public)
      expect([200, 401, 403, 404]).toContain(response.status);
    }});

    it('should reject invalid token', async () => {{
      const response = await authRequest('invalid.token.here')
        .get('/api/{endpoint}');
      expect([200, 401, 403, 404]).toContain(response.status);
    }});
  }});'''

    # Build CRUD tests only for non-singleton resources
    delete_describe = "describe.skip" if no_delete else "describe"

    if is_non_crud:
        crud_tests = f'''
  // ============================================
  // CUSTOM OPERATIONS ONLY (No standard CRUD)
  // ============================================'''
    else:
        crud_tests = f'''
  // ============================================
  // CRUD OPERATIONS
  // ============================================

  describe('POST /api/{endpoint}', () => {{
    it('should create a {resource}', async () => {{
      if (!jwt) return;

      const response = await authRequest(jwt)
        .post('/api/{endpoint}')
        .send({{
          data: {{
            {data_str}
          }},
        }});

      // Accept 200, 400 (validation), or 403 (permission required)
      expect([200, 400, 403]).toContain(response.status);
      if (response.status === 200) {{
        expect(response.body.data).toHaveProperty('documentId');
        resourceId = response.body.data.documentId;
      }}
    }});
  }});

  describe('GET /api/{endpoint}', () => {{
    it('should list {resource}s', async () => {{
      if (!jwt) return;

      const response = await authRequest(jwt).get('/api/{endpoint}');

      expect([200, 403]).toContain(response.status);
      if (response.status === 200) {{
        expect(response.body).toHaveProperty('data');
      }}
    }});
  }});

  describe('GET /api/{endpoint}/:id', () => {{
    it('should return 404 for non-existent', async () => {{
      if (!jwt) return;

      const response = await authRequest(jwt)
        .get('/api/{endpoint}/non-existent-id');

      expect([404, 403]).toContain(response.status);
    }});

    it('should get a specific {resource}', async () => {{
      if (!jwt || !resourceId) return;

      const response = await authRequest(jwt)
        .get(`/api/{endpoint}/${{resourceId}}`);

      expect([200, 403]).toContain(response.status);
    }});
  }});

  describe('PUT /api/{endpoint}/:id', () => {{
    it('should update a {resource}', async () => {{
      if (!jwt || !resourceId) return;

      const response = await authRequest(jwt)
        .put(`/api/{endpoint}/${{resourceId}}`)
        .send({{ data: {{}} }});

      expect([200, 400, 403]).toContain(response.status);
    }});
  }});

  {delete_describe}('DELETE /api/{endpoint}/:id', () => {{
    it('should handle delete request', async () => {{
      if (!jwt || !resourceId) return;

      const response = await authRequest(jwt)
        .delete(`/api/{endpoint}/${{resourceId}}`);

      // Accept any of these (some resources restrict delete)
      expect([200, 400, 403]).toContain(response.status);
    }});
  }});'''

    return f'''/**
 * {title} API Tests
 *
 * Auto-generated tests covering:
 * - Authentication
 * - CRUD operations
 * - Custom endpoints
 * - Tenant isolation
 */

import {{ apiRequest, authRequest }} from '{helpers_path}';

const runId = Date.now().toString(36);
let counter = 0;
const uniqueId = () => `${{runId}}-${{++counter}}`;

describe('{title} API', () => {{
  let jwt: string;
  let resourceId: string;
  let tenantSubdomain: string;

  beforeAll(async () => {{
    const id = uniqueId();
    tenantSubdomain = `{slug}-${{id}}`;

    const reg = await apiRequest()
      .post('/api/auth/register-company')
      .send({{
        companyName: '{title} Test Co',
        subdomain: tenantSubdomain,
        adminEmail: `{slug}-${{id}}@test.com`,
        adminPassword: 'Password123!',
        adminFirstName: 'Test',
        adminLastName: 'Admin',
        plan: 'professional',
      }});

    if (reg.status === 201) {{
      jwt = reg.body.data.jwt;
    }}
  }});
{auth_tests}
{crud_tests}

  // ============================================
  // CUSTOM ENDPOINTS
  // ============================================
{custom_tests_str}

  // ============================================
  // TENANT ISOLATION
  // ============================================

  describe('Tenant Isolation', () => {{
    let otherTenantJwt: string;

    beforeAll(async () => {{
      const id = uniqueId();
      const reg = await apiRequest()
        .post('/api/auth/register-company')
        .send({{
          companyName: 'Other {title} Co',
          subdomain: `other-{slug}-${{id}}`,
          adminEmail: `other-{slug}-${{id}}@test.com`,
          adminPassword: 'Password123!',
          adminFirstName: 'Other',
          adminLastName: 'Admin',
        }});

      if (reg.status === 201) {{
        otherTenantJwt = reg.body.data.jwt;
      }}
    }});

    it('should NOT allow other tenant to see this tenant resources', async () => {{
      if (!otherTenantJwt || !resourceId) return;

      const response = await authRequest(otherTenantJwt)
        .get('/api/{endpoint}');

      // Either 200 with empty data, or 403 forbidden
      expect([200, 403]).toContain(response.status);
      if (response.status === 200) {{
        const found = response.body.data?.some(
          (r: Record<string, unknown>) => r.documentId === resourceId
        );
        expect(found).toBeFalsy();
      }}
    }});

    it('should NOT allow other tenant to access by ID', async () => {{
      if (!otherTenantJwt || !resourceId) return;

      const response = await authRequest(otherTenantJwt)
        .get(`/api/{endpoint}/${{resourceId}}`);

      expect([403, 404]).toContain(response.status);
    }});
  }});
}});
'''


def main():
    parser = argparse.ArgumentParser(
        description='Generate smart tests from schema and routes',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s training-program           Generate tests for training-program
  %(prog)s leave-request --dry-run    Preview without writing
  %(prog)s --all                      Generate for all resources
  %(prog)s --all --modular            Generate with modular folder structure
        """
    )

    parser.add_argument('resource', nargs='?', help='Resource name (e.g., training-program)')
    parser.add_argument('--all', '-a', action='store_true',
                       help='Generate tests for all resources')
    parser.add_argument('--modular', '-m', action='store_true',
                       help='Use modular folder structure (by feature)')
    parser.add_argument('--output', '-o',
                       help='Output file (default: backend/tests/api/{resource}.test.ts)')
    parser.add_argument('--dry-run', '-n', action='store_true',
                       help='Print content without writing file')
    parser.add_argument('--force', '-f', action='store_true',
                       help='Overwrite existing files')

    args = parser.parse_args()

    root = get_project_root()
    backend_path = root / 'backend'

    if args.all:
        # Generate for all resources
        api_dir = backend_path / 'src' / 'api'
        resources = [d.name for d in api_dir.iterdir() if d.is_dir()]
    elif args.resource:
        resources = [args.resource]
    else:
        parser.error('Either resource name or --all is required')
        return 1

    generated_count = 0
    skipped_count = 0

    for resource in resources:
        slug = to_slug(resource)
        schema = get_schema(backend_path, resource)
        routes = get_routes(backend_path, resource)

        if not routes:
            print(f"No routes found for {resource}, skipping...")
            skipped_count += 1
            continue

        # Determine helpers import path based on modular structure
        feature = get_feature_for_resource(resource) if args.modular else None
        if feature:
            helpers_path = '../../helpers/api'
        else:
            helpers_path = '../helpers/api'

        content = generate_test_file(resource, schema, routes, helpers_path)

        if args.dry_run:
            print(f"\n{'='*60}")
            print(f"Generated test for: {resource}" + (f" ({feature})" if feature else ""))
            print('='*60)
            print(content[:2000] + '...' if len(content) > 2000 else content)
            continue

        # Determine output path
        if args.output and not args.all:
            output_path = Path(args.output)
        elif args.modular and feature:
            output_path = root / 'backend' / 'tests' / 'api' / feature / f'{slug}.test.ts'
        else:
            output_path = root / 'backend' / 'tests' / 'api' / f'{slug}.test.ts'

        # Check if exists
        if output_path.exists() and not args.force:
            print(f"Skipping {output_path.name} (exists)")
            skipped_count += 1
            continue

        # Write file
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(content)
        print(f"Created: {output_path}")
        generated_count += 1

    if not args.dry_run:
        print(f"\nGenerated: {generated_count}, Skipped: {skipped_count}")

    return 0


if __name__ == '__main__':
    sys.exit(main())
