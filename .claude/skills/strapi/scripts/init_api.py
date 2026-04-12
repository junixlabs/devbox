#!/usr/bin/env python3
"""
Initialize a Strapi API module following HRM coding standards.

Usage: python3 init_api.py <api-name> [--path backend/src/api]

Example:
    python3 init_api.py leave-type
    python3 init_api.py shift-template --path backend/src/api
"""

import argparse
from pathlib import Path

API_STRUCTURE = ["content-types/{name}", "controllers", "routes", "services"]


def to_pascal(name: str) -> str:
    """Convert kebab-case to PascalCase."""
    return ''.join(word.capitalize() for word in name.replace('-', '_').split('_'))


def to_camel(name: str) -> str:
    """Convert kebab-case to camelCase."""
    parts = name.replace('-', '_').split('_')
    return parts[0] + ''.join(word.capitalize() for word in parts[1:])


def get_schema_template(name: str, pascal_name: str) -> str:
    """Generate schema.json content."""
    return f'''{{
  "kind": "collectionType",
  "collectionName": "{name.replace('-', '_')}s",
  "info": {{
    "singularName": "{name}",
    "pluralName": "{name}s",
    "displayName": "{pascal_name}"
  }},
  "options": {{
    "draftAndPublish": false
  }},
  "attributes": {{
    "name": {{
      "type": "string",
      "required": true
    }},
    "description": {{
      "type": "text"
    }},
    "isActive": {{
      "type": "boolean",
      "default": true
    }},
    "tenant": {{
      "type": "relation",
      "relation": "manyToOne",
      "target": "api::tenant.tenant"
    }}
  }}
}}
'''


def get_controller_template(name: str) -> str:
    """Generate controller content."""
    return f'''import type {{ Core }} from '@strapi/strapi';
import {{ findOneWithTenantCheck }} from '../../../utils/tenant-query';

const UID = 'api::{name}.{name}';

const controller = ({{ strapi }}: {{ strapi: Core.Strapi }}) => ({{
  /**
   * Find all - scoped by tenant
   */
  async find(ctx) {{
    const user = ctx.state.user;
    if (!user) return ctx.unauthorized();

    const tenantId = user.tenant?.documentId;
    if (!tenantId) return ctx.forbidden('No tenant assigned');

    const filters = {{
      ...(ctx.query.filters || {{}}),
      tenant: {{ documentId: {{ $eq: tenantId }} }},
    }};

    const [data, total] = await Promise.all([
      strapi.documents(UID).findMany({{
        ...ctx.query,
        filters,
        populate: ctx.query.populate || ['tenant'],
      }}),
      strapi.documents(UID).count({{ filters }}),
    ]);

    return {{ data, meta: {{ pagination: {{ total }} }} }};
  }},

  /**
   * Find one - verify tenant ownership
   */
  async findOne(ctx) {{
    const user = ctx.state.user;
    if (!user) return ctx.unauthorized();

    const tenantId = user.tenant?.documentId;
    if (!tenantId) return ctx.forbidden('No tenant assigned');

    const {{ id }} = ctx.params;

    const {{ entity, error }} = await findOneWithTenantCheck(
      strapi,
      UID,
      id,
      tenantId
    );

    if (error === 'not_found') return ctx.notFound();
    if (error === 'forbidden') return ctx.forbidden();

    return {{ data: entity }};
  }},

  /**
   * Create - assign to tenant
   */
  async create(ctx) {{
    const user = ctx.state.user;
    if (!user) return ctx.unauthorized();

    const tenantId = user.tenant?.documentId;
    if (!tenantId) return ctx.forbidden('No tenant assigned');

    const {{ data }} = ctx.request.body;

    const entity = await strapi.documents(UID).create({{
      data: {{
        ...data,
        tenant: tenantId,
      }},
      populate: ['tenant'],
    }});

    return {{ data: entity }};
  }},

  /**
   * Update - verify tenant ownership
   */
  async update(ctx) {{
    const user = ctx.state.user;
    if (!user) return ctx.unauthorized();

    const tenantId = user.tenant?.documentId;
    if (!tenantId) return ctx.forbidden('No tenant assigned');

    const {{ id }} = ctx.params;
    const {{ data }} = ctx.request.body;

    const {{ entity, error }} = await findOneWithTenantCheck(strapi, UID, id, tenantId);

    if (error === 'not_found') return ctx.notFound();
    if (error === 'forbidden') return ctx.forbidden();

    const updated = await strapi.documents(UID).update({{
      documentId: id,
      data,
      populate: ['tenant'],
    }});

    return {{ data: updated }};
  }},

  /**
   * Delete - verify tenant ownership
   */
  async delete(ctx) {{
    const user = ctx.state.user;
    if (!user) return ctx.unauthorized();

    const tenantId = user.tenant?.documentId;
    if (!tenantId) return ctx.forbidden('No tenant assigned');

    const {{ id }} = ctx.params;

    const {{ error }} = await findOneWithTenantCheck(strapi, UID, id, tenantId);

    if (error === 'not_found') return ctx.notFound();
    if (error === 'forbidden') return ctx.forbidden();

    await strapi.documents(UID).delete({{ documentId: id }});

    return {{ data: {{ documentId: id }} }};
  }},
}});

export default controller;
'''


def get_routes_template(name: str) -> str:
    """Generate routes content."""
    return f'''export default {{
  routes: [
    {{
      method: 'GET',
      path: '/{name}s',
      handler: '{name}.find',
      config: {{
        policies: ['global::is-tenant-owner'],
        middlewares: ['global::tenant-context'],
      }},
    }},
    {{
      method: 'GET',
      path: '/{name}s/:id',
      handler: '{name}.findOne',
      config: {{
        policies: ['global::is-tenant-owner'],
        middlewares: ['global::tenant-context'],
      }},
    }},
    {{
      method: 'POST',
      path: '/{name}s',
      handler: '{name}.create',
      config: {{
        policies: ['global::is-tenant-owner'],
        middlewares: ['global::tenant-context'],
      }},
    }},
    {{
      method: 'PUT',
      path: '/{name}s/:id',
      handler: '{name}.update',
      config: {{
        policies: ['global::is-tenant-owner'],
        middlewares: ['global::tenant-context'],
      }},
    }},
    {{
      method: 'DELETE',
      path: '/{name}s/:id',
      handler: '{name}.delete',
      config: {{
        policies: ['global::is-tenant-owner'],
        middlewares: ['global::tenant-context'],
      }},
    }},
  ],
}};
'''


def get_service_template(name: str) -> str:
    """Generate service content."""
    return f'''import {{ factories }} from '@strapi/strapi';

const UID = 'api::{name}.{name}';

export default factories.createCoreService(UID, ({{ strapi }}) => ({{
  // Add custom service methods here
}}));
'''


def main():
    parser = argparse.ArgumentParser(description='Initialize a Strapi API module')
    parser.add_argument('name', help='API name (kebab-case, e.g., leave-type)')
    parser.add_argument('--path', default='backend/src/api', help='API directory')
    args = parser.parse_args()

    name = args.name.lower().replace('_', '-')
    pascal_name = to_pascal(name)

    api_path = Path(args.path) / name

    print(f"\n🚀 Creating API module: {name}\n")

    # Create directories
    print("📁 Creating directories...")
    for folder_template in API_STRUCTURE:
        folder = folder_template.format(name=name)
        (api_path / folder).mkdir(parents=True, exist_ok=True)
        print(f"   {api_path / folder}")

    # Create files
    templates = {
        f"content-types/{name}/schema.json": get_schema_template(name, pascal_name),
        f"controllers/{name}.ts": get_controller_template(name),
        f"routes/{name}.ts": get_routes_template(name),
        f"services/{name}.ts": get_service_template(name),
    }

    print("\n📄 Creating files...")
    for file_name, content in templates.items():
        file_path = api_path / file_name
        file_path.parent.mkdir(parents=True, exist_ok=True)
        file_path.write_text(content)
        print(f"   {file_path}")

    print(f"\n✅ API '{name}' created at {api_path}")
    print(f"\nUID: api::{name}.{name}")
    print(f"\nNext steps:")
    print(f"  1. Edit schema.json to add your attributes")
    print(f"  2. Add custom endpoints to routes/{name}.ts")
    print(f"  3. Implement business logic in controller")


if __name__ == '__main__':
    main()
