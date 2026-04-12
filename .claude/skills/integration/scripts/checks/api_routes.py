"""Check API route coverage and Next.js API routes."""

import re
from pathlib import Path
from utils import CheckResult


def check_nextjs_api_routes(frontend_dir: Path, result: CheckResult):
    """Find calls to /api/ that don't have corresponding Next.js routes."""
    api_routes_dir = frontend_dir / 'src' / 'app' / 'api'
    pages_api_dir = frontend_dir / 'src' / 'pages' / 'api'

    # Collect existing API routes
    existing_routes = set()

    for api_dir in [api_routes_dir, pages_api_dir]:
        if api_dir.exists():
            for route_file in api_dir.rglob('route.ts'):
                # Convert file path to API route
                rel_path = route_file.relative_to(api_dir).parent
                route = '/api/' + str(rel_path).replace('\\', '/')
                existing_routes.add(route.rstrip('/'))

            for route_file in api_dir.rglob('*.ts'):
                if route_file.name not in ['route.ts', 'route.js']:
                    rel_path = route_file.relative_to(api_dir)
                    route = '/api/' + str(rel_path.with_suffix('')).replace('\\', '/')
                    existing_routes.add(route.rstrip('/'))

    # Pattern to find /api/ calls
    api_call_pattern = r"['\"`](/api/[^'\"`\s]+)['\"`]"

    for ts_file in frontend_dir.rglob('*.tsx'):
        if 'node_modules' in str(ts_file) or '.next' in str(ts_file):
            continue

        try:
            content = ts_file.read_text()
            lines = content.split('\n')

            for line_num, line in enumerate(lines, 1):
                matches = re.findall(api_call_pattern, line)
                for api_path in matches:
                    # Normalize path (remove query params, dynamic segments)
                    normalized = re.sub(r'\?.*$', '', api_path)
                    normalized = re.sub(r'/\$\{[^}]+\}', '/:id', normalized)
                    base_path = '/'.join(normalized.split('/')[:4])  # /api/resource/action

                    # Check if route exists
                    if not existing_routes and api_path.startswith('/api/'):
                        result.error(
                            'missing-api-route',
                            str(ts_file.relative_to(frontend_dir.parent)),
                            line_num,
                            f"No Next.js API routes exist, but calling: {api_path}",
                            "This likely should call Strapi backend via API client, not Next.js API route"
                        )
        except Exception:
            pass


def normalize_route_path(path: str) -> str:
    """Normalize a route path for comparison."""
    # Replace any path parameter pattern with :id
    normalized = re.sub(r'/:[\w]+', '/:id', path)
    # Remove trailing slashes
    normalized = normalized.rstrip('/')
    return normalized


def check_api_route_coverage(backend_routes: dict, frontend_apis: dict, result: CheckResult):
    """Check that frontend API calls match backend routes."""
    # Build set of all backend endpoints (normalized)
    backend_endpoints: dict[str, set[str]] = {}  # method -> set of paths
    backend_endpoints_details: dict[str, dict[str, dict]] = {}  # method -> path -> route info

    for resource, routes in backend_routes.items():
        for route in routes:
            method = route['method']
            path = normalize_route_path(route['path'])
            if method not in backend_endpoints:
                backend_endpoints[method] = set()
                backend_endpoints_details[method] = {}
            backend_endpoints[method].add(path)
            backend_endpoints_details[method][path] = route

    # Check frontend calls against backend
    for feature, calls in frontend_apis.items():
        for call in calls:
            method = call['method']
            path = normalize_route_path(call['path'])

            # Skip Next.js API routes
            if path.startswith('/api/'):
                continue

            # Check if this endpoint exists in backend
            found = False
            if method in backend_endpoints:
                if path in backend_endpoints[method]:
                    found = True
                else:
                    # Check with :id normalization
                    for be_path in backend_endpoints[method]:
                        # Compare normalized paths
                        if path == be_path:
                            found = True
                            break
                        # Also check if frontend path matches backend with different param names
                        fe_parts = path.split('/')
                        be_parts = be_path.split('/')
                        if len(fe_parts) == len(be_parts):
                            match = True
                            for fe_p, be_p in zip(fe_parts, be_parts):
                                if fe_p != be_p and not (fe_p.startswith(':') and be_p.startswith(':')):
                                    match = False
                                    break
                            if match:
                                found = True
                                break

            if not found:
                # Determine severity - missing endpoints are errors, not just warnings
                result.error(
                    'missing-backend-route',
                    call['file'],
                    None,
                    f"Frontend calls {method} {call['path']} but backend route NOT IMPLEMENTED",
                    f"Add route to backend/src/api/{path.split('/')[1]}/routes/ or remove frontend call"
                )
