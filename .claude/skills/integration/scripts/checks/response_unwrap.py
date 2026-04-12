"""Check Strapi response unwrapping issues."""

import re
from pathlib import Path
from utils import CheckResult


def check_response_unwrapping(frontend_dir: Path, result: CheckResult):
    """Check that API calls properly unwrap Strapi responses.

    Strapi returns { data: T, meta?: {...} } but frontend code often expects T directly.
    APIs should either:
    - Use wrapped types like { data: T[] } or StrapiResponse<T>
    - Use .then(unwrap) to extract the data
    """
    features_dir = frontend_dir / 'src' / 'features'
    if not features_dir.exists():
        return

    # Pattern for api calls that return array types directly (likely wrong)
    # Matches: api.get<SomeType[]>('/path') without .then(unwrap)
    bad_array_pattern = r"api\.(get|post|put|delete|patch)\s*<\s*(\w+)\[\]\s*>\s*\([^)]+\)(?!\s*\.then\s*\(\s*unwrap)"

    # Pattern for api calls that return single types directly (likely wrong for CRUD)
    bad_single_pattern = r"api\.(get|post|put|delete|patch)\s*<\s*(\w+)\s*>\s*\([^)]+\)(?!\s*\.then\s*\(\s*unwrap)"

    # Good patterns (wrapped types or unwrap)
    good_patterns = [
        r"<\s*\{\s*data:",  # { data: T }
        r"<\s*StrapiResponse",  # StrapiResponse<T>
        r"<\s*ApiResponse",  # ApiResponse<T>
        r"<\s*ListResponse",  # ListResponse<T>
        r"<\s*SingleResponse",  # SingleResponse<T>
        r"\.then\s*\(\s*unwrap",  # .then(unwrap)
    ]

    for api_file in features_dir.rglob('api/index.ts'):
        try:
            feature = api_file.parent.parent.name
            content = api_file.read_text()
            lines = content.split('\n')

            # Check if file has good patterns (properly handling responses)
            has_good_pattern = any(re.search(p, content) for p in good_patterns)

            # If file already uses good patterns, it's likely fine
            if has_good_pattern:
                continue

            # Look for bad patterns - direct array types without unwrap
            for line_num, line in enumerate(lines, 1):
                # Skip imports, comments, type definitions
                stripped = line.strip()
                if stripped.startswith('import') or stripped.startswith('//') or stripped.startswith('*'):
                    continue
                if 'type ' in line or 'interface ' in line:
                    continue

                # Check for array return types without proper wrapping
                match = re.search(bad_array_pattern, line)
                if match:
                    method, type_name = match.groups()
                    result.error(
                        'response-unwrap',
                        str(api_file.relative_to(frontend_dir.parent)),
                        line_num,
                        f"api.{method}<{type_name}[]> returns Strapi wrapped response but expects array",
                        f"Add .then(unwrap) or use <StrapiResponse<{type_name}[]>> type"
                    )
                    continue

                # Check for single return types on CRUD-like endpoints
                match = re.search(bad_single_pattern, line)
                if match:
                    method, type_name = match.groups()
                    # Only flag if it looks like a CRUD endpoint (not custom responses)
                    if method in ['get', 'post', 'put'] and type_name[0].isupper():
                        # Skip if it's clearly a custom response type
                        if type_name.endswith('Response') or type_name.endswith('Result'):
                            continue
                        result.warning(
                            'response-unwrap',
                            str(api_file.relative_to(frontend_dir.parent)),
                            line_num,
                            f"api.{method}<{type_name}> may need response unwrapping",
                            f"Verify if Strapi returns {{ data: {type_name} }} and add unwrap if needed"
                        )
        except Exception:
            pass
