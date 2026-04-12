#!/usr/bin/env python3
"""
Test Generator for HRM Platform

Creates modular test files from templates.
"""

import argparse
import sys
from pathlib import Path

TEMPLATES = {
    'crud': '''/**
 * {title} API Tests
 *
 * Tests for {resource} CRUD operations
 */

import {{ apiRequest, authRequest }} from '../helpers/api';

const runId = Date.now().toString(36);
let counter = 0;
const uniqueId = () => `${{runId}}-${{++counter}}`;

describe('{title} API', () => {{
  let jwt: string;

  beforeAll(async () => {{
    const id = uniqueId();
    const reg = await apiRequest()
      .post('/api/auth/register-company')
      .send({{
        companyName: '{title} Test Co',
        subdomain: `{slug}-test-${{id}}`,
        adminEmail: `{slug}-${{id}}@test.com`,
        adminPassword: 'Password123!',
        adminFirstName: 'Test',
        adminLastName: 'Admin',
      }});
    jwt = reg.body.data.jwt;
  }});

  describe('POST /api/{endpoint}', () => {{
    it('should create a {resource}', async () => {{
      const response = await authRequest(jwt)
        .post('/api/{endpoint}')
        .send({{
          data: {{
            // Add required fields
          }},
        }});

      expect(response.status).toBe(200);
      expect(response.body.data).toHaveProperty('documentId');
    }});

    it('should reject without auth', async () => {{
      const response = await apiRequest()
        .post('/api/{endpoint}')
        .send({{ data: {{}} }});

      expect([401, 403]).toContain(response.status);
    }});
  }});

  describe('GET /api/{endpoint}', () => {{
    it('should list all {resource}s', async () => {{
      const response = await authRequest(jwt).get('/api/{endpoint}');

      expect(response.status).toBe(200);
      expect(response.body.data).toBeInstanceOf(Array);
    }});
  }});

  describe('GET /api/{endpoint}/:id', () => {{
    it('should return 404 for non-existent', async () => {{
      const response = await authRequest(jwt).get('/api/{endpoint}/non-existent');
      expect(response.status).toBe(404);
    }});
  }});

  describe('PUT /api/{endpoint}/:id', () => {{
    it('should update a {resource}', async () => {{
      // Create first
      const create = await authRequest(jwt)
        .post('/api/{endpoint}')
        .send({{ data: {{}} }});

      const id = create.body.data.documentId;

      const response = await authRequest(jwt)
        .put(`/api/{endpoint}/${{id}}`)
        .send({{ data: {{}} }});

      expect(response.status).toBe(200);
    }});
  }});

  describe('DELETE /api/{endpoint}/:id', () => {{
    it('should delete a {resource}', async () => {{
      // Create first
      const create = await authRequest(jwt)
        .post('/api/{endpoint}')
        .send({{ data: {{}} }});

      const id = create.body.data.documentId;

      const response = await authRequest(jwt).delete(`/api/{endpoint}/${{id}}`);
      expect(response.status).toBe(200);

      // Verify deleted
      const get = await authRequest(jwt).get(`/api/{endpoint}/${{id}}`);
      expect(get.status).toBe(404);
    }});
  }});
}});
''',

    'auth': '''/**
 * {title} Auth Tests
 *
 * Tests for {resource} authentication and authorization
 */

import {{ apiRequest, authRequest }} from '../helpers/api';

const runId = Date.now().toString(36);
let counter = 0;
const uniqueId = () => `${{runId}}-${{++counter}}`;

describe('{title} Auth', () => {{
  describe('Unauthenticated Access', () => {{
    it('should deny access to {endpoint} without auth', async () => {{
      const response = await apiRequest().get('/api/{endpoint}');
      expect([401, 403]).toContain(response.status);
    }});
  }});

  describe('Invalid Token', () => {{
    it('should deny access with invalid token', async () => {{
      const response = await authRequest('invalid-token').get('/api/{endpoint}');
      expect([401, 403]).toContain(response.status);
    }});
  }});
}});
''',

    'tenant': '''/**
 * {title} Tenant Isolation Tests
 *
 * Tests for {resource} tenant data isolation
 */

import {{ apiRequest, authRequest }} from '../helpers/api';

const runId = Date.now().toString(36);
let counter = 0;
const uniqueId = () => `${{runId}}-${{++counter}}`;

describe('{title} Tenant Isolation', () => {{
  let tenant1Jwt: string;
  let tenant2Jwt: string;
  let tenant1ResourceId: string;

  beforeAll(async () => {{
    // Create tenant 1
    const id1 = uniqueId();
    const reg1 = await apiRequest()
      .post('/api/auth/register-company')
      .send({{
        companyName: 'Tenant 1',
        subdomain: `t1-{slug}-${{id1}}`,
        adminEmail: `t1-{slug}-${{id1}}@test.com`,
        adminPassword: 'Password123!',
        adminFirstName: 'Admin',
        adminLastName: 'One',
      }});
    tenant1Jwt = reg1.body.data.jwt;

    // Create tenant 2
    const id2 = uniqueId();
    const reg2 = await apiRequest()
      .post('/api/auth/register-company')
      .send({{
        companyName: 'Tenant 2',
        subdomain: `t2-{slug}-${{id2}}`,
        adminEmail: `t2-{slug}-${{id2}}@test.com`,
        adminPassword: 'Password123!',
        adminFirstName: 'Admin',
        adminLastName: 'Two',
      }});
    tenant2Jwt = reg2.body.data.jwt;

    // Create resource in tenant 1
    const create = await authRequest(tenant1Jwt)
      .post('/api/{endpoint}')
      .send({{ data: {{}} }});
    tenant1ResourceId = create.body.data.documentId;
  }});

  it('should NOT allow tenant 2 to see tenant 1 resources', async () => {{
    const response = await authRequest(tenant2Jwt).get('/api/{endpoint}');

    expect(response.status).toBe(200);
    const found = response.body.data.some(
      (r: any) => r.documentId === tenant1ResourceId
    );
    expect(found).toBe(false);
  }});

  it('should NOT allow tenant 2 to access tenant 1 resource by ID', async () => {{
    const response = await authRequest(tenant2Jwt)
      .get(`/api/{endpoint}/${{tenant1ResourceId}}`);

    expect([403, 404]).toContain(response.status);
  }});

  it('should NOT allow tenant 2 to update tenant 1 resource', async () => {{
    const response = await authRequest(tenant2Jwt)
      .put(`/api/{endpoint}/${{tenant1ResourceId}}`)
      .send({{ data: {{}} }});

    expect([403, 404]).toContain(response.status);
  }});

  it('should NOT allow tenant 2 to delete tenant 1 resource', async () => {{
    const response = await authRequest(tenant2Jwt)
      .delete(`/api/{endpoint}/${{tenant1ResourceId}}`);

    expect([403, 404]).toContain(response.status);
  }});
}});
''',

    'workflow': '''/**
 * {title} Workflow Tests
 *
 * Tests for {resource} state transitions and workflows
 */

import {{ apiRequest, authRequest }} from '../helpers/api';

const runId = Date.now().toString(36);
let counter = 0;
const uniqueId = () => `${{runId}}-${{++counter}}`;

describe('{title} Workflow', () => {{
  let jwt: string;
  let resourceId: string;

  beforeAll(async () => {{
    const id = uniqueId();
    const reg = await apiRequest()
      .post('/api/auth/register-company')
      .send({{
        companyName: '{title} Workflow Co',
        subdomain: `{slug}-wf-${{id}}`,
        adminEmail: `{slug}-wf-${{id}}@test.com`,
        adminPassword: 'Password123!',
        adminFirstName: 'Test',
        adminLastName: 'Admin',
      }});
    jwt = reg.body.data.jwt;

    // Create resource for workflow tests
    const create = await authRequest(jwt)
      .post('/api/{endpoint}')
      .send({{ data: {{}} }});
    resourceId = create.body.data.documentId;
  }});

  describe('State Transitions', () => {{
    it('should transition from initial to next state', async () => {{
      // Example: submit, approve, etc.
      const response = await authRequest(jwt)
        .post(`/api/{endpoint}/${{resourceId}}/submit`)
        .send({{}});

      expect(response.status).toBe(200);
      // expect(response.body.data.status).toBe('submitted');
    }});
  }});
}});
''',
}


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


def generate_test(name: str, template: str, endpoint: str | None = None) -> str:
    """Generate test file content."""
    slug = to_slug(name)
    title = to_title(name)
    ep = endpoint or slug.replace('-', '-')

    if template not in TEMPLATES:
        raise ValueError(f"Unknown template: {template}. Available: {', '.join(TEMPLATES.keys())}")

    return TEMPLATES[template].format(
        title=title,
        resource=name.lower(),
        slug=slug,
        endpoint=ep,
    )


def main():
    parser = argparse.ArgumentParser(
        description='Generate test files for HRM Platform',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=f"""
Templates available:
  crud      - Full CRUD test suite
  auth      - Authentication/authorization tests
  tenant    - Tenant isolation tests
  workflow  - State transition tests

Examples:
  %(prog)s training-program --template crud
  %(prog)s leave-request --template workflow --endpoint leave-requests
  %(prog)s custom-role --template tenant
        """
    )

    parser.add_argument('name', help='Resource name (e.g., training-program)')
    parser.add_argument('--template', '-t', default='crud',
                       choices=list(TEMPLATES.keys()),
                       help='Test template to use')
    parser.add_argument('--endpoint', '-e',
                       help='API endpoint (default: derived from name)')
    parser.add_argument('--output', '-o',
                       help='Output file (default: backend/tests/api/{name}.test.ts)')
    parser.add_argument('--dry-run', '-n', action='store_true',
                       help='Print content without writing file')

    args = parser.parse_args()

    # Generate content
    content = generate_test(args.name, args.template, args.endpoint)

    if args.dry_run:
        print(content)
        return 0

    # Determine output path
    if args.output:
        output_path = Path(args.output)
    else:
        root = get_project_root()
        slug = to_slug(args.name)
        suffix = f'-{args.template}' if args.template != 'crud' else ''
        output_path = root / 'backend' / 'tests' / 'api' / f'{slug}{suffix}.test.ts'

    # Check if exists
    if output_path.exists():
        print(f"File already exists: {output_path}")
        response = input("Overwrite? [y/N] ").strip().lower()
        if response != 'y':
            print("Aborted.")
            return 1

    # Write file
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(content)
    print(f"Created: {output_path}")

    return 0


if __name__ == '__main__':
    sys.exit(main())
