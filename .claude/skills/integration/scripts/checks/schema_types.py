"""Check frontend types match backend schemas."""

from pathlib import Path
from utils import CheckResult
from loaders import get_backend_schemas


def check_schema_type_match(backend_dir: Path, frontend_dir: Path, result: CheckResult):
    """Check frontend types match backend schemas."""
    schemas = get_backend_schemas(backend_dir)

    for resource, schema in schemas.items():
        if 'attributes' not in schema:
            continue

        backend_fields = set(schema['attributes'].keys())
        backend_enums = {}

        # Extract enum values
        for field_name, field_def in schema['attributes'].items():
            if field_def.get('type') == 'enumeration' and 'enum' in field_def:
                backend_enums[field_name] = set(field_def['enum'])

        # Find corresponding frontend types file
        feature_name = resource.replace('-', '')
        types_file = frontend_dir / 'src' / 'features' / feature_name / 'types.ts'

        if not types_file.exists():
            # Try with hyphens
            features_dir = frontend_dir / 'src' / 'features'
            if features_dir.exists():
                for feature_dir in features_dir.iterdir():
                    if feature_dir.is_dir() and resource.replace('-', '') in feature_dir.name.replace('-', ''):
                        types_file = feature_dir / 'types.ts'
                        break

        if not types_file.exists():
            continue

        try:
            types_content = types_file.read_text()

            # Check for enum mismatches
            for enum_field, enum_values in backend_enums.items():
                # Look for type definition with these values
                for value in enum_values:
                    if f"'{value}'" not in types_content and f'"{value}"' not in types_content:
                        result.warning(
                            'missing-enum-value',
                            str(types_file.relative_to(frontend_dir.parent)),
                            None,
                            f"Backend enum '{enum_field}' has value '{value}' not found in frontend types",
                            f"Add '{value}' to the corresponding type union"
                        )
        except Exception:
            pass
