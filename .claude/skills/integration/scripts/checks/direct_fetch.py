"""Check for direct fetch() calls that should use API client."""

import re
from pathlib import Path
from utils import CheckResult


def check_direct_fetch_calls(frontend_dir: Path, result: CheckResult):
    """Find direct fetch() calls that should use API client."""
    patterns = [
        # fetch('/api/...')
        (r"fetch\s*\(\s*['\"`]/api/", "Direct fetch to /api/ route"),
        # fetch(`/api/...`)
        (r"fetch\s*\(\s*`/api/", "Direct fetch to /api/ route (template literal)"),
        # fetch(url) where url contains /api
        (r"fetch\s*\([^)]*['\"`][^'\"]*strapi[^'\"]*['\"`]", "Direct fetch to Strapi"),
    ]

    # Files to check (exclude node_modules, .next, etc.)
    for ts_file in frontend_dir.rglob('*.tsx'):
        if 'node_modules' in str(ts_file) or '.next' in str(ts_file):
            continue

        try:
            content = ts_file.read_text()
            lines = content.split('\n')

            for line_num, line in enumerate(lines, 1):
                for pattern, description in patterns:
                    if re.search(pattern, line):
                        result.error(
                            'direct-fetch',
                            str(ts_file.relative_to(frontend_dir.parent)),
                            line_num,
                            f"{description}: {line.strip()[:80]}",
                            "Use API client from '@/lib/api/client' or feature API module instead"
                        )
        except Exception as e:
            result.warning('file-read', str(ts_file), None, f"Could not read file: {e}")

    # Also check .ts files in features
    features_dir = frontend_dir / 'src' / 'features'
    if features_dir.exists():
        for ts_file in features_dir.rglob('*.ts'):
            if ts_file.name.endswith('.d.ts'):
                continue
            try:
                content = ts_file.read_text()
                lines = content.split('\n')

                for line_num, line in enumerate(lines, 1):
                    # Check for fetch without using api client
                    if 'fetch(' in line and "from '@/lib/api" not in content[:content.find('fetch(')]:
                        if '/api/' in line or 'strapi' in line.lower():
                            result.warning(
                                'direct-fetch',
                                str(ts_file.relative_to(frontend_dir.parent)),
                                line_num,
                                f"Possible direct fetch instead of API client: {line.strip()[:80]}"
                            )
            except Exception:
                pass
