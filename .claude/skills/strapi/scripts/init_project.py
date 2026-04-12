#!/usr/bin/env python3
"""
Initialize Strapi project with HRM structure.

Usage:
  python3 init_project.py <project-name> --new    # Create new project
  python3 init_project.py <project-name>          # Add structure to existing
"""

import argparse
import subprocess
import sys
from pathlib import Path

STRUCTURE = {
    "api": {},
    "bootstrap": {
        "hooks": {},
        "seeds": {},
    },
    "components": {},
    "extensions": {
        "users-permissions": {
            "content-types": {
                "user": {}
            }
        }
    },
    "middlewares": {},
    "policies": {},
    "utils": {}
}


def get_tenant_utils() -> str:
    """Generate tenant utility file."""
    return '''import type { Core } from '@strapi/strapi';

interface User {
  id: number;
  tenant?: {
    id?: number;
    documentId?: string;
  };
  isSuperAdmin?: boolean;
}

/**
 * Get user's tenant documentId
 */
export async function getTenantId(strapi: Core.Strapi, user: User): Promise<string | null> {
  if (user.isSuperAdmin) {
    return null;
  }

  if (user.tenant?.documentId) {
    return user.tenant.documentId;
  }

  const userWithTenant = await strapi.db
    .query('plugin::users-permissions.user')
    .findOne({
      where: { id: user.id },
      populate: ['tenant'],
    });

  return userWithTenant?.tenant?.documentId || null;
}

/**
 * Apply tenant filter to query
 */
export function applyTenantFilter(
  filters: Record<string, unknown>,
  tenantId: string
): Record<string, unknown> {
  return {
    ...filters,
    tenant: { documentId: { $eq: tenantId } },
  };
}
'''


def get_tenant_query_utils() -> str:
    """Generate tenant query utility file."""
    return '''import type { Core } from '@strapi/strapi';

/**
 * Build populate array that always includes tenant
 */
export function buildPopulateWithTenant(
  userPopulate?: string | string[],
  additionalPopulate: string[] = []
): string[] {
  let populateArray: string[] = [];

  if (typeof userPopulate === 'string') {
    populateArray = userPopulate.split(',').map(s => s.trim());
  } else if (Array.isArray(userPopulate)) {
    populateArray = userPopulate;
  }

  const combined = new Set(['tenant', ...populateArray, ...additionalPopulate]);
  return Array.from(combined);
}

interface TenantEntity {
  tenant?: { documentId?: string };
  [key: string]: unknown;
}

/**
 * Find one entity with tenant ownership check
 */
export async function findOneWithTenantCheck(
  strapi: Core.Strapi,
  uid: string,
  documentId: string,
  tenantId: string,
  options: { populate?: string | string[]; additionalPopulate?: string[] } = {}
): Promise<{ entity: TenantEntity | null; error?: 'not_found' | 'forbidden' }> {
  const populate = buildPopulateWithTenant(options.populate, options.additionalPopulate);

  const entity = await strapi.documents(uid as any).findOne({
    documentId,
    populate,
  }) as TenantEntity | null;

  if (!entity) {
    return { entity: null, error: 'not_found' };
  }

  if (entity.tenant?.documentId !== tenantId) {
    return { entity: null, error: 'forbidden' };
  }

  return { entity };
}
'''


def get_tenant_context_middleware() -> str:
    """Generate tenant context middleware."""
    return '''/**
 * Tenant context middleware
 * Ensures user.tenant is populated for downstream handlers
 */
export default () => async (ctx, next) => {
  const user = ctx.state.user;

  if (user && !user.tenant?.documentId) {
    const userWithTenant = await strapi.db
      .query('plugin::users-permissions.user')
      .findOne({
        where: { id: user.id },
        populate: ['tenant'],
      });

    if (userWithTenant?.tenant) {
      ctx.state.user.tenant = userWithTenant.tenant;
    }
  }

  await next();
};
'''


def get_is_tenant_owner_policy() -> str:
    """Generate is-tenant-owner policy."""
    return '''/**
 * Policy to check if user belongs to a tenant
 */
export default async (policyContext, config, { strapi }) => {
  const user = policyContext.state.user;

  if (!user) {
    return false;
  }

  // Super admins bypass tenant check
  if (user.isSuperAdmin) {
    return true;
  }

  const tenantId = user.tenant?.documentId;
  if (!tenantId) {
    return false;
  }

  return true;
};
'''


def get_bootstrap_index() -> str:
    """Generate bootstrap index file."""
    return '''export default async ({ strapi }) => {
  // Run bootstrap hooks
  // Example: await runSeeds(strapi);
};
'''


def get_src_index() -> str:
    """Generate src/index.ts file."""
    return '''export default {
  register(/* { strapi } */) {},

  async bootstrap({ strapi }) {
    const bootstrap = (await import('./bootstrap')).default;
    await bootstrap({ strapi });
  },
};
'''


BOILERPLATE_FILES = {
    "utils/tenant.ts": get_tenant_utils,
    "utils/tenant-query.ts": get_tenant_query_utils,
    "middlewares/tenant-context.ts": get_tenant_context_middleware,
    "policies/is-tenant-owner.ts": get_is_tenant_owner_policy,
    "bootstrap/index.ts": get_bootstrap_index,
    "index.ts": get_src_index,
}


def run_command(cmd: list, cwd: str = None, stdin_input: str = None):
    """Run a shell command."""
    print(f"  Running: {' '.join(cmd)}")
    result = subprocess.run(
        cmd,
        cwd=cwd,
        capture_output=False,
        input=stdin_input.encode() if stdin_input else None
    )
    return result.returncode == 0


def create_strapi_project(project_name: str):
    """Create new Strapi project using npx."""
    print("\n📦 Creating Strapi project with npx create-strapi...")

    cmd = [
        "npx", "create-strapi@latest", project_name,
        "--typescript",
        "--use-npm",
        "--no-example",
        "--skip-cloud",
        "--no-git-init",
        "--dbclient", "sqlite",
        "--install"
    ]

    if not run_command(cmd, stdin_input="n\n"):
        print("❌ Failed to create Strapi project")
        print("💡 Try running manually: npx create-strapi@latest", project_name)
        sys.exit(1)

    if not Path(project_name).exists():
        print("❌ Project directory was not created")
        sys.exit(1)

    print("✅ Strapi project created")


def create_structure(base_path: Path, structure: dict):
    """Recursively create directory structure."""
    for name, children in structure.items():
        path = base_path / name
        path.mkdir(parents=True, exist_ok=True)
        print(f"   {path}")
        if children:
            create_structure(path, children)


def create_boilerplate(base_path: Path):
    """Create boilerplate files."""
    for file_path, content_fn in BOILERPLATE_FILES.items():
        full_path = base_path / file_path
        if not full_path.exists():
            full_path.parent.mkdir(parents=True, exist_ok=True)
            full_path.write_text(content_fn())
            print(f"   {full_path}")


def main():
    parser = argparse.ArgumentParser(description='Initialize Strapi project')
    parser.add_argument('name', help='Project name / directory')
    parser.add_argument('--new', action='store_true', help='Create new project with npx')
    args = parser.parse_args()

    project_path = Path(args.name)
    src_path = project_path / "src"

    print(f"\n🚀 Initializing Strapi project: {args.name}\n")

    # Step 1: Create new project if --new flag
    if args.new:
        if project_path.exists():
            print(f"❌ Directory '{args.name}' already exists")
            sys.exit(1)
        create_strapi_project(args.name)

    # Verify project exists
    if not project_path.exists():
        print(f"❌ Directory '{args.name}' not found. Use --new to create.")
        sys.exit(1)

    # Step 2: Create directory structure
    print("📁 Creating directory structure...")
    create_structure(src_path, STRUCTURE)

    # Step 3: Create boilerplate files
    print("\n📄 Creating boilerplate files...")
    create_boilerplate(src_path)

    print(f"\n✅ Strapi project initialized at {project_path}")
    print("\nNext steps:")
    print(f"  1. cd {args.name}")
    print("  2. npm run develop")
    print("  3. Create API modules with: python3 .claude/skills/strapi/scripts/init_api.py <name>")


if __name__ == '__main__':
    main()
