#!/usr/bin/env python3
"""
Unified Test Runner for HRM Platform

Runs backend (Strapi/Jest) and frontend (Playwright E2E) tests.
Auto-detects target or use explicit --target flag.
"""

import argparse
import json
import os
import re
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Literal


# Configuration
BACKEND_DIR = 'backend'
FRONTEND_DIR = 'frontend'
E2E_DIR = 'e2e'
BUG_LOG_FILE = '.tmp/test-bugs.log'
TEST_RESULTS_FILE = '.tmp/test-results.json'
FAILURES_JSON_FILE = '.tmp/test-failures.json'
FAILURES_SUMMARY_FILE = '.tmp/test-failures-summary.txt'

Target = Literal['be', 'fe', 'all']


def get_project_root() -> Path:
    """Find the project root."""
    current = Path.cwd()

    if (current / BACKEND_DIR).exists() or (current / FRONTEND_DIR).exists():
        return current

    for parent in current.parents:
        if (parent / BACKEND_DIR).exists() or (parent / FRONTEND_DIR).exists():
            return parent

    return current


def get_backend_path() -> Path:
    """Get backend directory path."""
    return get_project_root() / BACKEND_DIR


def get_frontend_path() -> Path:
    """Get frontend directory path."""
    return get_project_root() / FRONTEND_DIR


def get_e2e_path() -> Path:
    """Get e2e directory path."""
    return get_project_root() / E2E_DIR


def detect_target() -> Target:
    """Auto-detect target based on current directory."""
    cwd = Path.cwd()
    cwd_str = str(cwd)

    if '/backend' in cwd_str or cwd_str.endswith('backend'):
        return 'be'
    elif '/frontend' in cwd_str or cwd_str.endswith('frontend'):
        return 'fe'
    elif '/e2e' in cwd_str or cwd_str.endswith('e2e'):
        return 'fe'

    # Default to backend if both exist
    root = get_project_root()
    if (root / BACKEND_DIR).exists():
        return 'be'
    elif (root / FRONTEND_DIR).exists():
        return 'fe'

    return 'be'


def ensure_tmp_dir(base_path: Path):
    """Ensure .tmp directory exists."""
    tmp_dir = base_path / '.tmp'
    tmp_dir.mkdir(parents=True, exist_ok=True)


def get_strapi_server_script() -> Path:
    """Get path to strapi-server skill script."""
    return get_project_root() / '.claude' / 'skills' / 'strapi-server' / 'scripts' / 'server.py'


def check_backend_server_running() -> bool:
    """Check if Strapi server is running."""
    script = get_strapi_server_script()

    if not script.exists():
        import urllib.request
        try:
            with urllib.request.urlopen('http://localhost:1337/_health', timeout=5) as r:
                return r.status in (200, 204)
        except Exception:
            return False

    result = subprocess.run(
        ['python3', str(script), 'status'],
        capture_output=True,
        text=True
    )

    return 'Health:  OK' in result.stdout


def start_backend_server() -> bool:
    """Start Strapi server."""
    script = get_strapi_server_script()

    if not script.exists():
        print("Error: strapi-server skill not found.")
        return False

    print("Starting Strapi server...")
    result = subprocess.run(
        ['python3', str(script), 'start'],
        capture_output=True,
        text=True
    )
    print(result.stdout)

    if result.returncode != 0:
        print(result.stderr)
        return False

    print("Waiting for server to be ready...")
    result = subprocess.run(
        ['python3', str(script), 'wait', '--timeout', '120'],
        text=True
    )

    return result.returncode == 0


def check_dependencies(path: Path) -> bool:
    """Check if test dependencies are installed."""
    node_modules = path / 'node_modules'
    return node_modules.exists()


def install_dependencies(path: Path) -> bool:
    """Install dependencies."""
    print(f"Installing dependencies in {path.name}...")

    result = subprocess.run(
        ['npm', 'install'],
        cwd=path,
        capture_output=True,
        text=True
    )

    if result.returncode == 0:
        print("Dependencies installed successfully.")
        return True
    else:
        print(f"Failed to install dependencies: {result.stderr}")
        return False


def setup_playwright(e2e_path: Path) -> bool:
    """Setup Playwright for E2E testing if not configured."""
    package_json = e2e_path / 'package.json'
    playwright_config = e2e_path / 'playwright.config.ts'

    # Create e2e directory if not exists
    if not e2e_path.exists():
        print("Creating e2e directory...")
        e2e_path.mkdir(parents=True, exist_ok=True)

    # Check if already configured
    if package_json.exists() and playwright_config.exists():
        with open(package_json) as f:
            pkg = json.load(f)
        if '@playwright/test' in pkg.get('devDependencies', {}):
            return True

    print("Setting up Playwright for E2E testing...")

    # Create package.json if not exists
    if not package_json.exists():
        pkg_content = {
            "name": "e2e-tests",
            "version": "1.0.0",
            "scripts": {
                "test": "playwright test",
                "test:ui": "playwright test --ui",
                "test:headed": "playwright test --headed",
                "test:debug": "playwright test --debug",
                "report": "playwright show-report"
            },
            "devDependencies": {}
        }
        with open(package_json, 'w') as f:
            json.dump(pkg_content, f, indent=2)

    # Install Playwright
    result = subprocess.run(
        ['npm', 'install', '-D', '@playwright/test'],
        cwd=e2e_path,
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"Failed to install Playwright: {result.stderr}")
        return False

    # Install browsers
    print("Installing Playwright browsers...")
    result = subprocess.run(
        ['npx', 'playwright', 'install', 'chromium'],
        cwd=e2e_path,
        capture_output=True,
        text=True
    )

    if result.returncode != 0:
        print(f"Warning: Could not install browsers: {result.stderr}")

    # Create playwright.config.ts
    if not playwright_config.exists():
        playwright_config.write_text('''import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for HRM E2E tests
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html'],
    ['json', { outputFile: '.tmp/test-results.json' }],
  ],
  use: {
    baseURL: process.env.BASE_URL || 'https://hrm.musetools.com',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  outputDir: './test-results',
});
''')

    # Create tests directory
    tests_dir = e2e_path / 'tests'
    tests_dir.mkdir(exist_ok=True)

    # Create screenshots directory
    screenshots_dir = e2e_path / 'tests' / 'screenshots'
    screenshots_dir.mkdir(exist_ok=True)

    # Create example test if no tests exist
    test_files = list(tests_dir.glob('*.test.ts')) + list(tests_dir.glob('*.spec.ts'))
    if not test_files:
        example_test = tests_dir / 'example.spec.ts'
        example_test.write_text('''import { test, expect } from '@playwright/test';

test.describe('Home Page', () => {
  test('should load successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/HRM|Sign In/i);
  });
});
''')

    print("Playwright setup complete.")
    return True


def run_backend_tests(
    file_pattern: str | None = None,
    coverage: bool = False,
    verbose: bool = False,
    watch: bool = False
) -> tuple[bool, str, list[dict]]:
    """Run backend Jest tests."""
    backend = get_backend_path()
    ensure_tmp_dir(backend)

    cmd = ['npm', 'test', '--']

    if file_pattern:
        cmd.append(f'--testPathPattern={file_pattern}')

    if coverage:
        cmd.append('--coverage')

    if verbose:
        cmd.append('--verbose')

    if watch:
        cmd.append('--watch')

    cmd.extend([
        '--json',
        f'--outputFile={TEST_RESULTS_FILE}',
        '--testLocationInResults'
    ])

    print(f"[Backend] Running: {' '.join(cmd)}")
    print("=" * 60)

    result = subprocess.run(
        cmd,
        cwd=backend,
        capture_output=True,
        text=True
    )

    output = result.stdout + result.stderr
    failures = parse_jest_results(backend / TEST_RESULTS_FILE)

    return result.returncode == 0, output, failures


def run_frontend_tests(
    file_pattern: str | None = None,
    headed: bool = False,
    debug: bool = False,
    ui: bool = False
) -> tuple[bool, str, list[dict]]:
    """Run frontend Playwright E2E tests."""
    e2e_path = get_e2e_path()
    ensure_tmp_dir(e2e_path)

    # Setup Playwright if needed
    if not setup_playwright(e2e_path):
        return False, "Failed to setup Playwright", []

    cmd = ['npx', 'playwright', 'test']

    if file_pattern:
        cmd.append(file_pattern)

    if headed:
        cmd.append('--headed')

    if debug:
        cmd.append('--debug')

    if ui:
        cmd.append('--ui')

    print(f"[Frontend E2E] Running: {' '.join(cmd)}")
    print("=" * 60)

    result = subprocess.run(
        cmd,
        cwd=e2e_path,
        capture_output=True,
        text=True
    )

    output = result.stdout + result.stderr
    failures = parse_playwright_results(e2e_path / '.tmp' / 'test-results.json')

    return result.returncode == 0, output, failures


def parse_jest_results(results_file: Path) -> list[dict]:
    """Parse Jest JSON results."""
    failures = []

    if not results_file.exists():
        return failures

    try:
        with open(results_file) as f:
            data = json.load(f)

        for test_result in data.get('testResults', []):
            file_path = test_result.get('name', 'unknown')

            for assertion in test_result.get('assertionResults', []):
                if assertion.get('status') == 'failed':
                    failures.append({
                        'file': file_path,
                        'test': assertion.get('fullName', 'unknown'),
                        'title': assertion.get('title', 'unknown'),
                        'messages': assertion.get('failureMessages', []),
                        'location': assertion.get('location', {}),
                        'source': 'backend',
                    })
    except Exception as e:
        print(f"Warning: Could not parse Jest results: {e}")

    return failures


def parse_playwright_results(results_file: Path) -> list[dict]:
    """Parse Playwright JSON results."""
    failures = []

    if not results_file.exists():
        return failures

    try:
        with open(results_file) as f:
            data = json.load(f)

        for suite in data.get('suites', []):
            parse_playwright_suite(suite, failures)

    except Exception as e:
        print(f"Warning: Could not parse Playwright results: {e}")

    return failures


def parse_playwright_suite(suite: dict, failures: list[dict], parent_title: str = ''):
    """Recursively parse Playwright test suites."""
    suite_title = suite.get('title', '')
    full_title = f"{parent_title} > {suite_title}" if parent_title else suite_title

    for spec in suite.get('specs', []):
        for test in spec.get('tests', []):
            for result in test.get('results', []):
                if result.get('status') == 'failed':
                    failures.append({
                        'file': suite.get('file', 'unknown'),
                        'test': f"{full_title} > {spec.get('title', 'unknown')}",
                        'title': spec.get('title', 'unknown'),
                        'messages': [result.get('error', {}).get('message', 'Unknown error')],
                        'location': {},
                        'source': 'e2e',
                    })

    # Recurse into nested suites
    for child_suite in suite.get('suites', []):
        parse_playwright_suite(child_suite, failures, full_title)


def log_bugs(failures: list[dict], target: str):
    """Log failures to bug log file and save detailed JSON."""
    if not failures:
        return

    if target == 'be':
        base_path = get_backend_path()
    else:
        base_path = get_e2e_path()

    ensure_tmp_dir(base_path)
    log_path = base_path / BUG_LOG_FILE
    json_path = base_path / FAILURES_JSON_FILE
    summary_path = base_path / FAILURES_SUMMARY_FILE

    timestamp = datetime.now().isoformat()

    # Save detailed JSON for programmatic access
    failures_data = {
        'timestamp': timestamp,
        'target': target,
        'total_failures': len(failures),
        'failures': failures
    }
    with open(json_path, 'w') as f:
        json.dump(failures_data, f, indent=2, default=str)

    # Save quick summary file (easy to read)
    with open(summary_path, 'w') as f:
        f.write(f"Test Failures Summary\n")
        f.write(f"{'=' * 60}\n")
        f.write(f"Time: {timestamp}\n")
        f.write(f"Target: {target}\n")
        f.write(f"Total Failures: {len(failures)}\n")
        f.write(f"{'=' * 60}\n\n")

        for i, failure in enumerate(failures, 1):
            file_name = Path(failure['file']).name
            f.write(f"{i}. {file_name}\n")
            f.write(f"   Test: {failure['title']}\n")

            # Extract key error info
            if failure.get('messages'):
                first_msg = failure['messages'][0]
                clean = re.sub(r'\x1b\[[0-9;]*m', '', first_msg)
                lines = [l.strip() for l in clean.split('\n') if l.strip()]

                # Find Expected/Received lines
                for line in lines:
                    if 'Expected' in line or 'Received' in line:
                        f.write(f"   {line[:100]}\n")
                        break
                else:
                    if lines:
                        f.write(f"   Error: {lines[0][:100]}\n")

            f.write("\n")

    # Append to full bug log (preserves history)
    with open(log_path, 'a') as f:
        f.write(f"\n{'=' * 60}\n")
        f.write(f"Test Run: {timestamp}\n")
        f.write(f"Target: {target}\n")
        f.write(f"Failures: {len(failures)}\n")
        f.write(f"{'=' * 60}\n\n")

        for i, failure in enumerate(failures, 1):
            f.write(f"--- Bug #{i} ---\n")
            f.write(f"Source: {failure.get('source', 'unknown')}\n")
            f.write(f"File: {failure['file']}\n")
            f.write(f"Test: {failure['test']}\n")

            if failure.get('location'):
                loc = failure['location']
                f.write(f"Location: line {loc.get('line', '?')}, col {loc.get('column', '?')}\n")

            f.write(f"\nError:\n")
            for msg in failure.get('messages', []):
                clean_msg = re.sub(r'\x1b\[[0-9;]*m', '', msg)
                f.write(f"{clean_msg}\n")

            f.write("\n")

    # Print paths to all output files
    print(f"\n{'=' * 60}")
    print("FAILURE LOGS SAVED:")
    print(f"  Quick summary: {summary_path}")
    print(f"  Detailed JSON: {json_path}")
    print(f"  Full log:      {log_path}")
    print(f"{'=' * 60}")


def print_failures(failures: list[dict]):
    """Print failures to console."""
    if not failures:
        print("\n✓ All tests passed!")
        return

    print(f"\n✗ {len(failures)} test(s) failed:\n")

    for i, failure in enumerate(failures, 1):
        source = failure.get('source', 'unknown')
        print(f"  {i}. [{source.upper()}] {failure['title']}")
        print(f"     File: {Path(failure['file']).name}")

        if failure.get('messages'):
            first_msg = failure['messages'][0]
            clean = re.sub(r'\x1b\[[0-9;]*m', '', first_msg)
            lines = [l for l in clean.split('\n') if l.strip()]
            if lines:
                for line in lines[:5]:
                    if 'Expected' in line or 'Received' in line or 'expect(' in line:
                        print(f"     {line.strip()[:80]}")
                        break
                else:
                    print(f"     {lines[0][:80]}")
        print()


def view_bugs(target: Target, summary_only: bool = False):
    """View the bug log file or quick summary."""
    base_paths = []
    if target in ('be', 'all'):
        base_paths.append(get_backend_path())
    if target in ('fe', 'all'):
        base_paths.append(get_e2e_path())

    found = False
    for base_path in base_paths:
        # Choose which file to show
        if summary_only:
            log_path = base_path / FAILURES_SUMMARY_FILE
            file_type = "summary"
        else:
            log_path = base_path / BUG_LOG_FILE
            file_type = "full log"

        if log_path.exists():
            found = True
            print(f"\n--- {base_path.name} {file_type} ---\n")
            print(log_path.read_text())

            # Show paths to other files
            json_path = base_path / FAILURES_JSON_FILE
            summary_path = base_path / FAILURES_SUMMARY_FILE
            full_path = base_path / BUG_LOG_FILE

            print(f"\n--- Available files ---")
            if summary_path.exists():
                print(f"  Summary: {summary_path}")
            if json_path.exists():
                print(f"  JSON:    {json_path}")
            if full_path.exists():
                print(f"  Full:    {full_path}")

    if not found:
        print("No bugs logged yet.")


def clear_bugs(target: Target):
    """Clear all bug/failure log files."""
    base_paths = []
    if target in ('be', 'all'):
        base_paths.append(get_backend_path())
    if target in ('fe', 'all'):
        base_paths.append(get_e2e_path())

    cleared = False
    for base_path in base_paths:
        files_to_clear = [
            base_path / BUG_LOG_FILE,
            base_path / FAILURES_JSON_FILE,
            base_path / FAILURES_SUMMARY_FILE,
        ]

        for file_path in files_to_clear:
            if file_path.exists():
                file_path.unlink()
                print(f"Cleared: {file_path}")
                cleared = True

    if not cleared:
        print("No bug logs to clear.")


def cmd_run(args):
    """Run tests command."""
    target = args.target or detect_target()

    all_failures = []
    all_success = True

    # Backend tests
    if target in ('be', 'all'):
        # Clear bug log
        log_path = get_backend_path() / BUG_LOG_FILE
        if log_path.exists():
            log_path.unlink()

        # Check server
        if not check_backend_server_running():
            print("Strapi server is not running.")
            if not start_backend_server():
                print("Error: Could not start server.")
                if target == 'be':
                    return 1

        # Check dependencies
        backend = get_backend_path()
        if not check_dependencies(backend):
            if not install_dependencies(backend):
                if target == 'be':
                    return 1

        success, output, failures = run_backend_tests(
            file_pattern=args.file,
            coverage=args.coverage,
            verbose=args.verbose,
            watch=args.watch
        )

        # Print output
        for line in output.split('\n'):
            if not line.strip().startswith(('{', '"')):
                print(line)

        all_failures.extend(failures)
        all_success = all_success and success

        if failures:
            log_bugs(failures, 'be')

    # Frontend E2E tests
    if target in ('fe', 'all'):
        # Clear bug log
        e2e_path = get_e2e_path()
        log_path = e2e_path / BUG_LOG_FILE
        if log_path.exists():
            log_path.unlink()

        success, output, failures = run_frontend_tests(
            file_pattern=args.file,
            headed=args.headed,
            debug=args.debug,
            ui=args.ui
        )

        # Print output
        print(output)

        all_failures.extend(failures)
        all_success = all_success and success

        if failures:
            log_bugs(failures, 'fe')

    # Summary
    print_failures(all_failures)

    return 0 if all_success else 1


def cmd_bugs(args):
    """View bugs command."""
    target = args.target or 'all'
    summary_only = getattr(args, 'summary', False)
    view_bugs(target, summary_only=summary_only)
    return 0


def cmd_failures_json(args):
    """Print path to failures JSON for programmatic access."""
    target = args.target or detect_target()

    if target == 'be':
        json_path = get_backend_path() / FAILURES_JSON_FILE
    else:
        json_path = get_e2e_path() / FAILURES_JSON_FILE

    if json_path.exists():
        print(json_path)
        return 0
    else:
        print(f"No failures JSON found at: {json_path}", file=sys.stderr)
        return 1


def cmd_clear(args):
    """Clear bugs command."""
    target = args.target or 'all'
    clear_bugs(target)
    return 0


def cmd_report(args):
    """Open Playwright HTML report."""
    e2e_path = get_e2e_path()
    subprocess.run(['npx', 'playwright', 'show-report'], cwd=e2e_path)
    return 0


def main():
    parser = argparse.ArgumentParser(
        description='Unified Test Runner for HRM Platform',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s run                      Auto-detect and run tests
  %(prog)s run --target be          Run backend tests (Jest)
  %(prog)s run --target fe          Run frontend E2E tests (Playwright)
  %(prog)s run --target all         Run all tests
  %(prog)s run --file auth          Run tests matching 'auth'
  %(prog)s run --headed             Run E2E with visible browser
  %(prog)s run --ui                 Run E2E with Playwright UI
  %(prog)s bugs                     View full bug log
  %(prog)s bugs --summary           View quick failure summary
  %(prog)s failures-json            Print path to failures JSON file
  %(prog)s clear                    Clear bug log
  %(prog)s report                   Open Playwright HTML report

Output files (in .tmp/):
  test-failures-summary.txt   Quick summary of failures
  test-failures.json          Detailed JSON for programmatic access
  test-bugs.log               Full log with error traces
        """
    )

    # Global options
    parser.add_argument('--target', '-t', choices=['be', 'fe', 'all'],
                       help='Target platform (default: auto-detect)')

    subparsers = parser.add_subparsers(dest='command', required=True)

    # Run command
    run_parser = subparsers.add_parser('run', help='Run tests')
    run_parser.add_argument('--target', '-t', choices=['be', 'fe', 'all'],
                           help='Target platform (default: auto-detect)')
    run_parser.add_argument('--file', '-f', help='Test file pattern')
    run_parser.add_argument('--coverage', '-c', action='store_true', help='Generate coverage (BE only)')
    run_parser.add_argument('--verbose', '-v', action='store_true', help='Verbose output (BE only)')
    run_parser.add_argument('--watch', '-w', action='store_true', help='Watch mode (BE only)')
    run_parser.add_argument('--headed', action='store_true', help='Run with visible browser (FE only)')
    run_parser.add_argument('--debug', action='store_true', help='Debug mode (FE only)')
    run_parser.add_argument('--ui', action='store_true', help='Playwright UI mode (FE only)')
    run_parser.set_defaults(func=cmd_run)

    # Bugs command
    bugs_parser = subparsers.add_parser('bugs', help='View bug log')
    bugs_parser.add_argument('--target', '-t', choices=['be', 'fe', 'all'],
                            help='Target platform (default: all)')
    bugs_parser.add_argument('--summary', '-s', action='store_true',
                            help='Show quick summary instead of full log')
    bugs_parser.set_defaults(func=cmd_bugs)

    # Failures JSON path command (for programmatic access)
    json_parser = subparsers.add_parser('failures-json', help='Print path to failures JSON')
    json_parser.add_argument('--target', '-t', choices=['be', 'fe'],
                            help='Target platform (default: auto-detect)')
    json_parser.set_defaults(func=cmd_failures_json)

    # Clear command
    clear_parser = subparsers.add_parser('clear', help='Clear bug log')
    clear_parser.add_argument('--target', '-t', choices=['be', 'fe', 'all'],
                             help='Target platform (default: all)')
    clear_parser.set_defaults(func=cmd_clear)

    # Report command
    report_parser = subparsers.add_parser('report', help='Open Playwright HTML report')
    report_parser.set_defaults(func=cmd_report)

    args = parser.parse_args()
    sys.exit(args.func(args))


if __name__ == '__main__':
    main()
