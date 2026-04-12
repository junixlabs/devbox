"""Data loaders for backend schemas, routes, and frontend API calls."""

import json
import re
from pathlib import Path


def get_backend_schemas(backend_dir: Path) -> dict[str, dict]:
    """Load all backend schemas."""
    schemas = {}
    api_dir = backend_dir / 'src' / 'api'

    if not api_dir.exists():
        return schemas

    for schema_file in api_dir.rglob('schema.json'):
        try:
            resource = schema_file.parent.parent.parent.name
            with open(schema_file) as f:
                schemas[resource] = json.load(f)
        except Exception:
            pass

    return schemas


def get_plural_name_from_schema(api_dir: Path, resource: str) -> str:
    """Get plural name from schema.json, falling back to resource + 's'."""
    schema_path = api_dir / resource / 'content-types' / resource / 'schema.json'
    if schema_path.exists():
        try:
            with open(schema_path) as f:
                schema = json.load(f)
                return schema.get('info', {}).get('pluralName', resource + 's')
        except Exception:
            pass
    # Default: add 's' for pluralization
    return resource + 's'


def get_backend_routes(backend_dir: Path) -> dict[str, list[dict]]:
    """Load all backend route definitions."""
    routes = {}
    api_dir = backend_dir / 'src' / 'api'

    if not api_dir.exists():
        return routes

    # Pattern for explicit route definitions
    route_pattern = r"\{\s*method:\s*['\"](\w+)['\"].*?path:\s*['\"]([^'\"]+)['\"].*?handler:\s*['\"]([^'\"]+)['\"]"

    # Pattern for createCoreRouter (implies default CRUD routes)
    core_router_pattern = r"factories\.createCoreRouter\s*\(\s*['\"]api::([^'\"]+)\.[^'\"]+['\"]"

    for routes_file in api_dir.rglob('*.ts'):
        if 'routes' not in str(routes_file):
            continue

        try:
            resource = routes_file.parent.parent.name
            content = routes_file.read_text()

            if resource not in routes:
                routes[resource] = []

            # Check for createCoreRouter - adds default CRUD routes
            core_match = re.search(core_router_pattern, content)
            if core_match:
                # Get the pluralized name from schema.json (Strapi uses pluralName for routes)
                plural_resource = get_plural_name_from_schema(api_dir, resource)
                routes[resource].extend([
                    {'method': 'GET', 'path': f'/{plural_resource}', 'handler': 'find', 'file': str(routes_file), 'implicit': True},
                    {'method': 'GET', 'path': f'/{plural_resource}/:id', 'handler': 'findOne', 'file': str(routes_file), 'implicit': True},
                    {'method': 'POST', 'path': f'/{plural_resource}', 'handler': 'create', 'file': str(routes_file), 'implicit': True},
                    {'method': 'PUT', 'path': f'/{plural_resource}/:id', 'handler': 'update', 'file': str(routes_file), 'implicit': True},
                    {'method': 'DELETE', 'path': f'/{plural_resource}/:id', 'handler': 'delete', 'file': str(routes_file), 'implicit': True},
                ])

            # Also get explicit route definitions
            matches = re.findall(route_pattern, content, re.DOTALL)

            for method, path, handler in matches:
                routes[resource].append({
                    'method': method.upper(),
                    'path': path,
                    'handler': handler,
                    'file': str(routes_file),
                })
        except Exception:
            pass

    return routes


def get_frontend_api_calls(frontend_dir: Path) -> dict[str, list[dict]]:
    """Extract API calls from frontend API modules."""
    api_calls = {}

    features_dir = frontend_dir / 'src' / 'features'
    if not features_dir.exists():
        return api_calls

    # Pattern for api.get/post/put/delete calls with single/double quotes
    api_string_pattern = r"api\.(get|post|put|delete|patch)\s*(?:<[^>]+>)?\s*\(\s*'(/[^']+)'"
    api_dquote_pattern = r'api\.(get|post|put|delete|patch)\s*(?:<[^>]+>)?\s*\(\s*"(/[^"]+)"'
    # Template literal paths - capture everything between backticks
    # Note: This may include nested backticks in ternary expressions like ${query ? `?${query}` : ''}
    api_template_pattern = r"api\.(get|post|put|delete|patch)\s*(?:<[^>]+>)?\s*\(\s*`(.*?)`\s*[,)]"

    for api_file in features_dir.rglob('api/index.ts'):
        try:
            feature = api_file.parent.parent.name
            content = api_file.read_text()

            if feature not in api_calls:
                api_calls[feature] = []

            seen_paths = set()  # Avoid duplicates

            # Find single-quoted paths
            for method, path in re.findall(api_string_pattern, content):
                base_path = path.split('?')[0].rstrip('/')
                if not base_path:
                    continue
                key = f"{method.upper()} {base_path}"
                if key not in seen_paths:
                    seen_paths.add(key)
                    api_calls[feature].append({
                        'method': method.upper(),
                        'path': base_path,
                        'file': str(api_file),
                    })

            # Find double-quoted paths
            for method, path in re.findall(api_dquote_pattern, content):
                base_path = path.split('?')[0].rstrip('/')
                if not base_path:
                    continue
                key = f"{method.upper()} {base_path}"
                if key not in seen_paths:
                    seen_paths.add(key)
                    api_calls[feature].append({
                        'method': method.upper(),
                        'path': base_path,
                        'file': str(api_file),
                    })

            # Find template literal paths
            for method, path in re.findall(api_template_pattern, content, re.DOTALL):
                # Check if path starts with valid API segment
                if not path.startswith('/'):
                    continue

                # Strategy: Process the path character by character to distinguish
                # path params (/${id}) from query injections (${query...})
                #
                # Path params: always preceded by /
                # Query injections: preceded by ? or nothing (at end of static path)

                result_path = []
                i = 0
                while i < len(path):
                    if path[i:i+2] == '${':
                        # Found a template variable
                        # Find the closing }
                        close_idx = path.find('}', i + 2)
                        if close_idx == -1:
                            break  # Malformed, stop

                        var_content = path[i+2:close_idx]

                        # Check what precedes this ${
                        if i > 0 and path[i-1] == '/':
                            # This is a path param - replace with :id
                            result_path.append(':id')
                            i = close_idx + 1
                        elif '?' in var_content or 'query' in var_content.lower() or 'params' in var_content.lower() or 'buildQuery' in var_content:
                            # This is query string injection - stop here
                            break
                        else:
                            # Some other injection (like at end for query), stop
                            break
                    else:
                        result_path.append(path[i])
                        i += 1

                base_path = ''.join(result_path)

                # Remove trailing ? and query string remnants
                base_path = base_path.split('?')[0]

                # Clean up any doubled slashes or trailing slashes
                base_path = re.sub(r'//+', '/', base_path).rstrip('/')

                if base_path and base_path != '/':
                    key = f"{method.upper()} {base_path}"
                    if key not in seen_paths:
                        seen_paths.add(key)
                        api_calls[feature].append({
                            'method': method.upper(),
                            'path': base_path,
                            'file': str(api_file),
                        })
        except Exception:
            pass

    return api_calls


def get_controller_validations(backend_dir: Path) -> dict[str, list[dict]]:
    """Extract validation rules from backend controllers.

    Looks for patterns like:
    - if (!data.field) return ctx.badRequest('...')
    - if (data.type === 'x' && !data.field) return ctx.badRequest('...')
    """
    validations = {}
    api_dir = backend_dir / 'src' / 'api'

    if not api_dir.exists():
        return validations

    # Patterns for validation checks
    patterns = [
        # Simple required: if (!data.field) ctx.badRequest
        (r"if\s*\(\s*!data\.(\w+)\s*\).*?(?:ctx\.)?badRequest\s*\(\s*['\"]([^'\"]+)['\"]",
         'required', None),
        # Conditional required: if (data.x === 'y' && !data.field) ctx.badRequest
        (r"if\s*\(\s*data\.(\w+)\s*===?\s*['\"](\w+)['\"]\s*&&\s*!data\.(\w+)\s*\).*?(?:ctx\.)?badRequest\s*\(\s*['\"]([^'\"]+)['\"]",
         'conditional', None),
        # Reverse conditional: if (!data.field && data.x === 'y') ctx.badRequest
        (r"if\s*\(\s*!data\.(\w+)\s*&&\s*data\.(\w+)\s*===?\s*['\"](\w+)['\"]\s*\).*?(?:ctx\.)?badRequest\s*\(\s*['\"]([^'\"]+)['\"]",
         'conditional_reverse', None),
    ]

    for controller_file in api_dir.rglob('controllers/*.ts'):
        try:
            resource = controller_file.parent.parent.name
            content = controller_file.read_text()

            if resource not in validations:
                validations[resource] = []

            # Check for simple required fields
            for match in re.finditer(patterns[0][0], content, re.DOTALL):
                field = match.group(1)
                message = match.group(2)
                validations[resource].append({
                    'type': 'required',
                    'field': field,
                    'message': message,
                    'file': str(controller_file),
                })

            # Check for conditional required fields (data.x === 'y' && !data.field)
            for match in re.finditer(patterns[1][0], content, re.DOTALL):
                condition_field = match.group(1)
                condition_value = match.group(2)
                required_field = match.group(3)
                message = match.group(4)
                validations[resource].append({
                    'type': 'conditional',
                    'field': required_field,
                    'condition': f"{condition_field} === '{condition_value}'",
                    'condition_field': condition_field,
                    'condition_value': condition_value,
                    'message': message,
                    'file': str(controller_file),
                })

            # Check for reverse conditional (!data.field && data.x === 'y')
            for match in re.finditer(patterns[2][0], content, re.DOTALL):
                required_field = match.group(1)
                condition_field = match.group(2)
                condition_value = match.group(3)
                message = match.group(4)
                validations[resource].append({
                    'type': 'conditional',
                    'field': required_field,
                    'condition': f"{condition_field} === '{condition_value}'",
                    'condition_field': condition_field,
                    'condition_value': condition_value,
                    'message': message,
                    'file': str(controller_file),
                })

        except Exception:
            pass

    return validations


def get_frontend_form_fields(frontend_dir: Path) -> dict[str, dict]:
    """Extract form fields from frontend components.

    Looks for:
    - Input/Select components with value={formData.field}
    - Form state with field names
    - Conditional rendering based on form state
    """
    form_fields = {}

    # Check pages and feature components
    search_dirs = [
        frontend_dir / 'src' / 'app',
        frontend_dir / 'src' / 'features',
    ]

    # Patterns to find form fields
    field_patterns = [
        # value={formData.field} or value={formData.field ?? ''}
        r"value=\{(?:formData|form|data)\.(\w+)",
        # onChange with formData.field
        r"setFormData\(\s*\{[^}]*(\w+):\s*",
        # formData: { field: ... } in default state
        r"(?:defaultFormData|initialValues?|defaultValues?)\s*[=:]\s*\{([^}]+)\}",
        # Input/Select with name="field"
        r"(?:Input|Select|Switch|Checkbox|DatePicker|TimePicker)[^>]*name=['\"](\w+)['\"]",
    ]

    # Pattern to find conditional rendering
    conditional_pattern = r"formData\.(\w+)\s*===?\s*['\"](\w+)['\"]\s*&&"

    # Detect resource from file content - component-specific patterns
    # Maps unique field/type names to their backend resources
    content_resource_indicators = [
        # Salary component specific fields
        (r'calculationType|percentageOf|isDeduction|isTaxable', 'salary-component'),
        # Insurance rate specific
        (r'employeeRate|employerRate|minSalary|maxSalary', 'insurance-rate'),
        # Tax deduction specific
        (r'deductionType|dependentCount|threshold', 'tax-deduction-rule'),
        # Leave type specific
        (r'accrualRate|carryOver|maxBalance', 'leave-type'),
    ]

    for search_dir in search_dirs:
        if not search_dir.exists():
            continue

        for tsx_file in search_dir.rglob('*.tsx'):
            if 'node_modules' in str(tsx_file) or '.next' in str(tsx_file):
                continue

            try:
                content = tsx_file.read_text()

                # Skip files without forms
                if 'formData' not in content and 'useForm' not in content:
                    continue

                # Try to identify what resource this form is for
                resource = None
                file_path = str(tsx_file)

                # PRIORITY 1: Check content for create/update mutations (most reliable)
                mutation_match = re.search(r'use(?:Create|Update)(\w+)', content)
                if mutation_match:
                    # Convert CamelCase to kebab-case
                    name = mutation_match.group(1)
                    resource = re.sub(r'(?<!^)(?=[A-Z])', '-', name).lower()

                # PRIORITY 2: Check for component-specific field names in content
                if not resource:
                    for pattern, res in content_resource_indicators:
                        if re.search(pattern, content):
                            resource = res
                            break

                # PRIORITY 3: Check for resource indicators in path (least reliable)
                if not resource:
                    # Check for resource indicators
                    resource_indicators = [
                        (r'/salary-component', 'salary-component'),
                        (r'/insurance-rate', 'insurance-rate'),
                        (r'/tax-deduction', 'tax-deduction-rule'),
                        (r'/payroll/', 'payroll'),
                        (r'/employee', 'employee'),
                        (r'/leave', 'leave'),
                        (r'/attendance', 'attendance'),
                    ]

                    for pattern, res in resource_indicators:
                        if re.search(pattern, file_path):
                            resource = res
                            break

                if not resource:
                    continue

                if resource not in form_fields:
                    form_fields[resource] = {
                        'fields': set(),
                        'conditionals': {},  # field -> list of (condition_field, condition_value) when shown
                        'files': [],
                    }

                form_fields[resource]['files'].append(str(tsx_file))

                # Extract field names
                for pattern in field_patterns[:2]:  # Direct field references
                    for match in re.finditer(pattern, content):
                        field = match.group(1)
                        form_fields[resource]['fields'].add(field)

                # Extract fields from default state objects
                default_match = re.search(field_patterns[2], content, re.DOTALL)
                if default_match:
                    default_content = default_match.group(1)
                    for field_match in re.finditer(r"(\w+)\s*:", default_content):
                        form_fields[resource]['fields'].add(field_match.group(1))

                # Extract named inputs
                for match in re.finditer(field_patterns[3], content):
                    form_fields[resource]['fields'].add(match.group(1))

                # Find conditional field rendering
                # Look for patterns like: {formData.type === 'percentage' && (<Input for percentageOf>)}
                lines = content.split('\n')
                for i, line in enumerate(lines):
                    cond_match = re.search(conditional_pattern, line)
                    if cond_match:
                        condition_field = cond_match.group(1)
                        condition_value = cond_match.group(2)

                        # Look in next 20 lines for field references
                        context_lines = '\n'.join(lines[i:i+20])
                        for field_match in re.finditer(r"formData\.(\w+)", context_lines):
                            shown_field = field_match.group(1)
                            if shown_field != condition_field:
                                if shown_field not in form_fields[resource]['conditionals']:
                                    form_fields[resource]['conditionals'][shown_field] = []
                                form_fields[resource]['conditionals'][shown_field].append(
                                    (condition_field, condition_value)
                                )

            except Exception:
                pass

    return form_fields
