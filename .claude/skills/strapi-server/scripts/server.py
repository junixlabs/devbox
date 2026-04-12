#!/usr/bin/env python3
"""
Strapi Development Server Manager for WSL
"""

import argparse
import os
import signal
import subprocess
import sys
import time
import urllib.request
import urllib.error
import json
from pathlib import Path


# Configuration
DEFAULT_HOST = os.environ.get('STRAPI_HOST', 'localhost')
DEFAULT_PORT = int(os.environ.get('STRAPI_PORT', 1337))
DEFAULT_TIMEOUT = 120  # 2 minutes
POLL_INTERVAL = 5  # seconds
BACKEND_DIR = os.environ.get('BACKEND_DIR', 'backend')


def get_backend_path() -> Path:
    """Get the backend directory path."""
    current = Path.cwd()

    if (current / BACKEND_DIR).exists():
        return current / BACKEND_DIR

    for parent in current.parents:
        if (parent / BACKEND_DIR).exists():
            return parent / BACKEND_DIR

    return current / BACKEND_DIR


def get_server_pids() -> list[int]:
    """Get all PIDs of running Strapi processes."""
    try:
        result = subprocess.run(
            ['pgrep', '-f', 'strapi develop'],
            capture_output=True,
            text=True
        )
        if result.returncode == 0 and result.stdout.strip():
            return [int(pid) for pid in result.stdout.strip().split('\n')]
    except Exception:
        pass
    return []


def get_server_pid() -> int | None:
    """Get the first PID of the running Strapi process."""
    pids = get_server_pids()
    return pids[0] if pids else None


def is_server_healthy(host: str = DEFAULT_HOST, port: int = DEFAULT_PORT) -> bool:
    """Check if Strapi is responding."""
    try:
        url = f'http://{host}:{port}/_health'
        with urllib.request.urlopen(url, timeout=5) as response:
            return response.status in (200, 204)
    except Exception:
        return False


def start_server() -> bool:
    """Start the Strapi development server."""
    backend_path = get_backend_path()

    if not backend_path.exists():
        print(f"Error: Backend directory not found at {backend_path}")
        return False

    if is_server_healthy():
        print("Server already running.")
        return True

    if get_server_pid():
        print("Server starting up...")
        return True

    print(f"Starting Strapi from {backend_path}...")

    log_file = backend_path / '.tmp' / 'strapi-server.log'
    log_file.parent.mkdir(parents=True, exist_ok=True)

    with open(log_file, 'w') as log:
        process = subprocess.Popen(
            ['npm', 'run', 'develop'],
            cwd=backend_path,
            stdout=log,
            stderr=subprocess.STDOUT,
            start_new_session=True
        )

    print(f"Started (PID {process.pid})")
    print(f"Logs: {log_file}")
    return True


def stop_server() -> bool:
    """Stop the Strapi server."""
    pids = get_server_pids()

    if not pids:
        print("Server not running.")
        return True

    print(f"Stopping server (PIDs: {pids})...")

    # Kill all processes
    for pid in pids:
        try:
            os.kill(pid, signal.SIGTERM)
        except ProcessLookupError:
            pass

    # Wait for processes to stop
    for _ in range(10):
        time.sleep(1)
        if not get_server_pids():
            print("Stopped.")
            return True

    # Force kill remaining
    for pid in get_server_pids():
        try:
            os.kill(pid, signal.SIGKILL)
        except ProcessLookupError:
            pass

    time.sleep(1)
    if not get_server_pids():
        print("Force stopped.")
        return True

    print("Warning: Some processes may still be running.")
    return False


def get_new_log_lines(log_file: Path, last_pos: int) -> tuple[list[str], int]:
    """Read new lines from log file since last position."""
    if not log_file.exists():
        return [], 0

    try:
        with open(log_file, 'r') as f:
            f.seek(last_pos)
            new_content = f.read()
            new_pos = f.tell()

        if new_content:
            return new_content.strip().split('\n'), new_pos
        return [], new_pos
    except Exception:
        return [], last_pos


def check_for_errors(lines: list[str]) -> str | None:
    """Check log lines for fatal errors. Returns error message if found."""
    error_patterns = [
        'error:',
        'Error:',
        'ERROR',
        'FATAL',
        'failed',
        'Cannot find module',
        'SyntaxError',
        'TypeError',
        'ReferenceError',
        'EADDRINUSE',
        'EACCES',
    ]

    for line in lines:
        for pattern in error_patterns:
            if pattern in line:
                return line
    return None


def wait_for_server(timeout: int = DEFAULT_TIMEOUT, host: str = DEFAULT_HOST, port: int = DEFAULT_PORT) -> bool:
    """Wait for the server to be ready, showing log output."""
    print(f"Waiting for {host}:{port} (timeout: {timeout}s)...")

    log_file = get_backend_path() / '.tmp' / 'strapi-server.log'
    last_log_pos = 0
    start_time = time.time()
    no_process_count = 0  # Count consecutive checks with no process

    while time.time() - start_time < timeout:
        # Check if healthy
        if is_server_healthy(host, port):
            elapsed = int(time.time() - start_time)
            print(f"\nReady! ({elapsed}s)")
            return True

        # Check if process died (allow a few retries since npm spawns child processes)
        if not get_server_pids():
            no_process_count += 1
            if no_process_count >= 3:  # 3 consecutive checks = ~15s
                print("\nServer process died!")
                # Show last log lines
                if log_file.exists():
                    with open(log_file) as f:
                        lines = f.readlines()[-30:]
                    print("Last log lines:")
                    for line in lines:
                        print(f"  {line.rstrip()}")
                return False
        else:
            no_process_count = 0

        # Read new log lines
        new_lines, last_log_pos = get_new_log_lines(log_file, last_log_pos)

        if new_lines:
            for line in new_lines:
                # Skip empty lines
                if not line.strip():
                    continue
                print(f"  {line}")

            # Check for fatal errors that mean we should stop waiting
            error = check_for_errors(new_lines)
            if error:
                if any(fatal in error for fatal in ['FATAL', 'Cannot find module', 'SyntaxError', 'EADDRINUSE', 'EACCES']):
                    print(f"\nFatal error detected!")
                    return False

        time.sleep(POLL_INTERVAL)

    print(f"\nTimeout after {timeout}s")
    return False


def check_status(host: str = DEFAULT_HOST, port: int = DEFAULT_PORT, verbose: bool = False) -> None:
    """Check server status."""
    pid = get_server_pid()
    healthy = is_server_healthy(host, port)

    print(f"Process: {'Running (PID {})'.format(pid) if pid else 'Not running'}")
    print(f"Health:  {'OK' if healthy else 'Not responding'}")

    if verbose and pid:
        log_file = get_backend_path() / '.tmp' / 'strapi-server.log'
        if log_file.exists():
            print(f"\nLast 10 log lines:")
            with open(log_file) as f:
                lines = f.readlines()[-10:]
                for line in lines:
                    print(f"  {line.rstrip()}")


def test_api(host: str = DEFAULT_HOST, port: int = DEFAULT_PORT) -> bool:
    """Test API endpoints."""
    base_url = f'http://{host}:{port}'
    print(f"Testing {base_url}")

    tests = [
        ('Health', '/_health', [200, 204]),
        ('Admin', '/admin/init', [200]),
        ('API', '/api', [200, 404]),
    ]

    all_passed = True
    for name, path, expected in tests:
        try:
            with urllib.request.urlopen(f'{base_url}{path}', timeout=5) as r:
                ok = r.status in expected
                print(f"  {'OK' if ok else 'FAIL'} {name}: {r.status}")
                if not ok:
                    all_passed = False
        except urllib.error.HTTPError as e:
            ok = e.code in expected
            print(f"  {'OK' if ok else 'FAIL'} {name}: {e.code}")
            if not ok:
                all_passed = False
        except Exception as e:
            print(f"  FAIL {name}: {e}")
            all_passed = False

    return all_passed


def main():
    parser = argparse.ArgumentParser(description='Strapi Server Manager')
    parser.add_argument('command', choices=['start', 'stop', 'status', 'wait', 'test'])
    parser.add_argument('--host', default=DEFAULT_HOST)
    parser.add_argument('--port', type=int, default=DEFAULT_PORT)
    parser.add_argument('--timeout', type=int, default=DEFAULT_TIMEOUT)
    parser.add_argument('-v', '--verbose', action='store_true')

    args = parser.parse_args()

    if args.command == 'start':
        sys.exit(0 if start_server() else 1)
    elif args.command == 'stop':
        sys.exit(0 if stop_server() else 1)
    elif args.command == 'status':
        check_status(args.host, args.port, args.verbose)
    elif args.command == 'wait':
        sys.exit(0 if wait_for_server(args.timeout, args.host, args.port) else 1)
    elif args.command == 'test':
        sys.exit(0 if test_api(args.host, args.port) else 1)


if __name__ == '__main__':
    main()
