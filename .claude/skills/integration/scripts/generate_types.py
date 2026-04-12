#!/usr/bin/env python3
"""
Generate TypeScript types from Strapi content-type schema.

Usage:
    python3 generate_types.py <resource-name>

Example:
    python3 generate_types.py leave-request

Reads from:  backend/src/api/{resource}/content-types/{resource}/schema.json
Outputs to:  stdout (copy to frontend/src/features/{feature}/types.ts)
"""

import json
import sys
from pathlib import Path

# Type mapping from Strapi to TypeScript
STRAPI_TO_TS = {
    'string': 'string',
    'text': 'string',
    'richtext': 'string',
    'email': 'string',
    'password': 'string',
    'uid': 'string',
    'integer': 'number',
    'biginteger': 'number',
    'float': 'number',
    'decimal': 'number',
    'date': 'string',
    'datetime': 'string',
    'time': 'string',
    'boolean': 'boolean',
    'json': 'Record<string, unknown>',
    'media': 'MediaAsset',
}


def to_pascal_case(name: str) -> str:
    """Convert kebab-case to PascalCase."""
    return ''.join(word.capitalize() for word in name.split('-'))


def get_ts_type(attr: dict, attr_name: str) -> str:
    """Convert Strapi attribute to TypeScript type."""
    attr_type = attr.get('type', 'string')

    # Enumeration
    if attr_type == 'enumeration':
        values = attr.get('enum', [])
        return ' | '.join(f"'{v}'" for v in values)

    # Relation
    if attr_type == 'relation':
        target = attr.get('target', '')
        # Extract entity name from api::entity.entity
        entity_name = target.split('.')[-1] if '.' in target else target
        pascal_name = to_pascal_case(entity_name)

        relation_type = attr.get('relation', '')
        if 'Many' in relation_type:
            return f'{pascal_name}Ref[]'
        return f'{pascal_name}Ref'

    # Component
    if attr_type == 'component':
        component = attr.get('component', '')
        component_name = component.split('.')[-1] if '.' in component else component
        pascal_name = to_pascal_case(component_name)
        if attr.get('repeatable'):
            return f'{pascal_name}[]'
        return pascal_name

    # Dynamic zone
    if attr_type == 'dynamiczone':
        return 'unknown[]'

    return STRAPI_TO_TS.get(attr_type, 'unknown')


def generate_entity_type(name: str, schema: dict) -> str:
    """Generate TypeScript interface for entity."""
    pascal_name = to_pascal_case(name)
    attrs = schema.get('attributes', {})

    lines = [f'export interface {pascal_name} {{']
    lines.append('  documentId: string;')

    for attr_name, attr in attrs.items():
        if attr_name in ('createdAt', 'updatedAt', 'publishedAt', 'tenant'):
            continue

        ts_type = get_ts_type(attr, attr_name)
        required = attr.get('required', False)
        optional = '' if required else '?'

        lines.append(f'  {attr_name}{optional}: {ts_type};')

    lines.append('  createdAt: string;')
    lines.append('  updatedAt: string;')
    lines.append('}')

    return '\n'.join(lines)


def generate_form_type(name: str, schema: dict) -> str:
    """Generate TypeScript interface for form data."""
    pascal_name = to_pascal_case(name)
    attrs = schema.get('attributes', {})

    lines = [f'export interface {pascal_name}FormData {{']

    for attr_name, attr in attrs.items():
        if attr_name in ('createdAt', 'updatedAt', 'publishedAt', 'tenant'):
            continue

        attr_type = attr.get('type', 'string')
        required = attr.get('required', False)
        optional = '' if required else '?'

        # Relations become string IDs in forms
        if attr_type == 'relation':
            relation = attr.get('relation', '')
            if 'Many' in relation:
                lines.append(f'  {attr_name}{optional}: string[];')
            else:
                lines.append(f'  {attr_name}{optional}: string;')
        else:
            ts_type = get_ts_type(attr, attr_name)
            lines.append(f'  {attr_name}{optional}: {ts_type};')

    lines.append('}')

    return '\n'.join(lines)


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    resource_name = sys.argv[1]

    # Find project root (contains backend/ and frontend/)
    cwd = Path.cwd()
    while cwd != cwd.parent:
        if (cwd / 'backend').is_dir() and (cwd / 'frontend').is_dir():
            break
        cwd = cwd.parent
    else:
        print('Error: Could not find project root (with backend/ and frontend/)')
        sys.exit(1)

    schema_path = cwd / 'backend' / 'src' / 'api' / resource_name / 'content-types' / resource_name / 'schema.json'

    if not schema_path.exists():
        print(f'Error: Schema not found at {schema_path}')
        sys.exit(1)

    with open(schema_path) as f:
        schema = json.load(f)

    feature_name = resource_name.replace('-', '')
    print(f'// Generated from {resource_name} schema')
    print(f'// Copy to: frontend/src/features/{feature_name}/types.ts')
    print()
    print(generate_entity_type(resource_name, schema))
    print()
    print(generate_form_type(resource_name, schema))


if __name__ == '__main__':
    main()
