#!/usr/bin/env python3
"""
Generate frontend API functions from Strapi routes.

Usage:
    python3 generate_api.py <resource-name>

Example:
    python3 generate_api.py leave-request

Reads from:  backend/src/api/{resource}/routes/{resource}.ts
Outputs to:  stdout (copy to frontend/src/features/{feature}/api/index.ts)
"""

import re
import sys
from pathlib import Path


def to_camel_case(name: str) -> str:
    """Convert kebab-case to camelCase."""
    parts = name.split('-')
    return parts[0] + ''.join(word.capitalize() for word in parts[1:])


def to_pascal_case(name: str) -> str:
    """Convert kebab-case to PascalCase."""
    return ''.join(word.capitalize() for word in name.split('-'))


def parse_routes(content: str, resource: str) -> list[dict]:
    """Extract route definitions from TypeScript file."""
    routes = []

    # Find all route objects
    route_pattern = r"\{\s*method:\s*['\"](\w+)['\"].*?path:\s*['\"]([^'\"]+)['\"].*?handler:\s*['\"]([^'\"]+)['\"]"
    matches = re.findall(route_pattern, content, re.DOTALL)

    for method, path, handler in matches:
        # Normalize path
        path = path.strip()
        handler_name = handler.split('.')[-1] if '.' in handler else handler

        routes.append({
            'method': method.upper(),
            'path': path,
            'handler': handler_name,
        })

    return routes


def generate_api_function(route: dict, resource: str, entity_name: str) -> str:
    """Generate API function for a route."""
    method = route['method']
    path = route['path']
    handler = route['handler']

    # Determine function name from handler
    func_name = handler
    if handler == 'find':
        func_name = f'get{entity_name}s'
    elif handler == 'findOne':
        func_name = f'get{entity_name}'
    elif handler == 'create':
        func_name = f'create{entity_name}'
    elif handler == 'update':
        func_name = f'update{entity_name}'
    elif handler == 'delete':
        func_name = f'delete{entity_name}'

    # Build function
    lines = []

    if method == 'GET' and ':id' not in path:
        # List endpoint
        lines.append(f'export async function {func_name}(params?: ListParams) {{')
        lines.append('  const query = buildQueryString(params);')
        lines.append(f"  return api.get<ListResponse<{entity_name}>>('{path}', query);")
        lines.append('}')

    elif method == 'GET' and ':id' in path:
        # Single item endpoint
        lines.append(f'export async function {func_name}(id: string) {{')
        api_path = path.replace(':id', '${id}')
        lines.append(f"  return api.get<SingleResponse<{entity_name}>>(`{api_path}`);")
        lines.append('}')

    elif method == 'POST' and ':id' not in path:
        # Create endpoint
        lines.append(f'export async function {func_name}(data: {entity_name}FormData) {{')
        lines.append(f"  return api.post<SingleResponse<{entity_name}>>('{path}', {{ data }});")
        lines.append('}')

    elif method == 'PUT' and ':id' in path:
        # Update endpoint
        lines.append(f'export async function {func_name}(id: string, data: Partial<{entity_name}FormData>) {{')
        api_path = path.replace(':id', '${id}')
        lines.append(f"  return api.put<SingleResponse<{entity_name}>>(`{api_path}`, {{ data }});")
        lines.append('}')

    elif method == 'DELETE' and ':id' in path:
        # Delete endpoint
        lines.append(f'export async function {func_name}(id: string) {{')
        api_path = path.replace(':id', '${id}')
        lines.append(f"  return api.delete(`{api_path}`);")
        lines.append('}')

    else:
        # Custom endpoint - generate stub
        lines.append(f'// TODO: Implement {func_name} for {method} {path}')
        lines.append(f'export async function {func_name}() {{')
        lines.append("  throw new Error('Not implemented');")
        lines.append('}')

    return '\n'.join(lines)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    resource_name = sys.argv[1]
    entity_name = to_pascal_case(resource_name)

    # Find project root
    cwd = Path.cwd()
    while cwd != cwd.parent:
        if (cwd / 'backend').is_dir() and (cwd / 'frontend').is_dir():
            break
        cwd = cwd.parent
    else:
        print('Error: Could not find project root')
        sys.exit(1)

    routes_path = cwd / 'backend' / 'src' / 'api' / resource_name / 'routes' / f'{resource_name}.ts'

    if not routes_path.exists():
        print(f'Error: Routes not found at {routes_path}')
        sys.exit(1)

    with open(routes_path) as f:
        content = f.read()

    routes = parse_routes(content, resource_name)

    # Generate output
    feature_name = resource_name.replace('-', '')
    print(f'// Generated from {resource_name} routes')
    print(f'// Copy to: frontend/src/features/{feature_name}/api/index.ts')
    print()
    print("import { api } from '@/lib/api/client';")
    print(f"import type {{ {entity_name}, {entity_name}FormData }} from '../types';")
    print()
    print('interface ListParams {')
    print('  page?: number;')
    print('  pageSize?: number;')
    print('}')
    print()
    print('interface Pagination {')
    print('  page: number;')
    print('  pageSize: number;')
    print('  pageCount: number;')
    print('  total: number;')
    print('}')
    print()
    print('interface ListResponse<T> { data: T[]; meta: { pagination: Pagination }; }')
    print('interface SingleResponse<T> { data: T; }')
    print()
    print('function buildQueryString(params?: ListParams): Record<string, string> {')
    print('  const query: Record<string, string> = {};')
    print("  if (params?.page) query['pagination[page]'] = String(params.page);")
    print("  if (params?.pageSize) query['pagination[pageSize]'] = String(params.pageSize);")
    print('  return query;')
    print('}')
    print()

    for route in routes:
        print(generate_api_function(route, resource_name, entity_name))
        print()


if __name__ == '__main__':
    main()
