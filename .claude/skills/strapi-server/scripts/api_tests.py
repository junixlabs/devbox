#!/usr/bin/env python3
"""
Comprehensive API Tests for Strapi Backend

Tests all developed APIs using the proper auth flow:
1. Register company via /api/auth/register-company
2. Use JWT for authenticated requests
"""

import argparse
import json
import os
import sys
import urllib.request
import urllib.error
from typing import Any
from dataclasses import dataclass


# Configuration
DEFAULT_PORT = 1337


def get_default_host() -> str:
    """Get default host - use Windows IP if Strapi is running there."""
    env_host = os.environ.get('STRAPI_HOST')
    if env_host:
        return env_host

    # Check if Windows host has Strapi running (for WSL)
    if os.path.exists('/etc/resolv.conf'):
        try:
            with open('/etc/resolv.conf', 'r') as f:
                for line in f:
                    if line.startswith('nameserver'):
                        windows_ip = line.split()[1]
                        # Test if Windows host is reachable on Strapi port
                        import socket
                        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                        sock.settimeout(1)
                        result = sock.connect_ex((windows_ip, DEFAULT_PORT))
                        sock.close()
                        if result == 0:
                            return windows_ip
        except Exception:
            pass

    return 'localhost'


BASE_URL = os.environ.get('STRAPI_URL', f'http://{get_default_host()}:{DEFAULT_PORT}')


@dataclass
class TestResult:
    name: str
    passed: bool
    message: str
    data: Any = None


class APITester:
    def __init__(self, base_url: str, verbose: bool = False):
        self.base_url = base_url
        self.verbose = verbose
        self.jwt_token: str | None = None
        self.tenant_id: str | None = None
        self.user_data: dict | None = None
        self.test_data: dict = {}
        self.results: list[TestResult] = []

    def log(self, message: str) -> None:
        if self.verbose:
            print(f"  [DEBUG] {message}")

    def request(
        self,
        method: str,
        path: str,
        data: dict | None = None,
        headers: dict | None = None,
        use_auth: bool = True
    ) -> tuple[int, Any]:
        """Make HTTP request to API."""
        url = f"{self.base_url}{path}"
        req_headers = {'Content-Type': 'application/json'}

        if headers:
            req_headers.update(headers)

        if use_auth and self.jwt_token:
            req_headers['Authorization'] = f'Bearer {self.jwt_token}'

        body = json.dumps(data).encode('utf-8') if data else None

        self.log(f"{method} {url}")
        if data:
            self.log(f"Body: {json.dumps(data)[:200]}")

        try:
            req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
            with urllib.request.urlopen(req, timeout=30) as response:
                status = response.status
                try:
                    response_data = json.loads(response.read().decode('utf-8'))
                except Exception:
                    response_data = None
                self.log(f"Response: {status}")
                return status, response_data
        except urllib.error.HTTPError as e:
            try:
                error_data = json.loads(e.read().decode('utf-8'))
            except Exception:
                error_data = {'error': str(e)}
            self.log(f"Error: {e.code} - {error_data}")
            return e.code, error_data
        except Exception as e:
            self.log(f"Exception: {e}")
            return 0, {'error': str(e)}

    def add_result(self, name: str, passed: bool, message: str, data: Any = None) -> None:
        result = TestResult(name, passed, message, data)
        self.results.append(result)
        status = "[PASS]" if passed else "[FAIL]"
        print(f"{status} {name}: {message}")
        if self.verbose and data:
            print(f"       Data: {json.dumps(data, indent=2)[:500]}")

    # ==================== Health Tests ====================

    def test_health(self) -> None:
        """Test health endpoint."""
        status, _ = self.request('GET', '/_health', use_auth=False)
        self.add_result(
            "Health Check",
            status in [200, 204],
            f"Status {status}"
        )

    # ==================== Company Registration ====================

    def test_register_company(self) -> None:
        """Register a new company with admin user."""
        status, data = self.request(
            'POST',
            '/api/auth/register-company',
            data={
                'companyName': 'Test Company',
                'subdomain': 'testcompany',
                'plan': 'starter',
                'adminEmail': 'admin@testcompany.com',
                'adminPassword': 'TestPass123',
                'adminFirstName': 'Admin',
                'adminLastName': 'User'
            },
            use_auth=False
        )

        if status == 201 and data and data.get('data', {}).get('jwt'):
            self.jwt_token = data['data']['jwt']
            self.tenant_id = data['data']['tenant']['documentId']
            self.user_data = data['data']['user']
            self.add_result("Register Company", True, f"Created tenant {self.tenant_id}")
        else:
            error_msg = data.get('error', {}).get('message', '') if data else ''
            self.add_result("Register Company", False, f"Status {status}: {error_msg}")

    def test_login(self) -> None:
        """Login if registration failed (company already exists)."""
        if self.jwt_token:
            self.add_result("User Login", True, "Already authenticated via registration")
            return

        status, data = self.request(
            'POST',
            '/api/auth/local',
            data={
                'identifier': 'admin@testcompany.com',
                'password': 'TestPass123'
            },
            use_auth=False
        )

        if status == 200 and data and data.get('jwt'):
            self.jwt_token = data['jwt']
            self.user_data = data.get('user')
            self.add_result("User Login", True, "Login successful")
        else:
            error_msg = data.get('error', {}).get('message', '') if data else ''
            self.add_result("User Login", False, f"Status {status}: {error_msg}")

    def test_get_me(self) -> None:
        """Get current user profile with tenant info."""
        status, data = self.request('GET', '/api/auth/me')

        if status == 200 and data and data.get('data'):
            user = data['data']
            self.tenant_id = user.get('tenant', {}).get('documentId') if user.get('tenant') else self.tenant_id
            self.add_result("Get Current User", True, f"User: {user.get('email')}")
        else:
            self.add_result("Get Current User", False, f"Status {status}")

    # ==================== Organization Tests ====================

    def test_create_department(self) -> None:
        """Create a department."""
        status, data = self.request(
            'POST',
            '/api/departments',
            data={
                'data': {
                    'name': 'Engineering',
                    'code': 'ENG',
                    'isActive': True
                }
            }
        )

        if status in [200, 201] and data and data.get('data'):
            self.test_data['department_id'] = data['data'].get('documentId')
            self.add_result("Create Department", True, "Created department")
        else:
            error_msg = data.get('error', {}).get('message', '') if data else str(data)
            self.add_result("Create Department", False, f"Status {status}: {error_msg}")

    def test_list_departments(self) -> None:
        """List departments."""
        status, data = self.request('GET', '/api/departments')
        if status == 200 and data:
            departments = data.get('data', []) or []
            self.add_result("List Departments", True, f"Found {len(departments)} departments")
        else:
            self.add_result("List Departments", False, f"Status {status}")

    def test_create_employee(self) -> None:
        """Create an employee."""
        payload = {
            'data': {
                'employeeId': 'EMP001',
                'firstName': 'John',
                'lastName': 'Doe',
                'email': 'john.doe@testcompany.com',
                'status': 'active',
                'employmentType': 'full_time'
            }
        }

        if self.test_data.get('department_id'):
            payload['data']['department'] = self.test_data['department_id']

        status, data = self.request('POST', '/api/employees', data=payload)

        if status in [200, 201] and data and data.get('data'):
            self.test_data['employee_id'] = data['data'].get('documentId')
            self.add_result("Create Employee", True, "Created employee")
        else:
            error_msg = data.get('error', {}).get('message', '') if data else str(data)
            self.add_result("Create Employee", False, f"Status {status}: {error_msg}")

    def test_list_employees(self) -> None:
        """List employees."""
        status, data = self.request('GET', '/api/employees')
        if status == 200 and data:
            employees = data.get('data', []) or []
            self.add_result("List Employees", True, f"Found {len(employees)} employees")
        else:
            self.add_result("List Employees", False, f"Status {status}")

    def run_all_tests(self) -> bool:
        """Run all tests in sequence."""
        print("\n" + "=" * 60)
        print("STRAPI API COMPREHENSIVE TESTS")
        print("=" * 60)

        # Health
        print("\n--- Health Check ---")
        self.test_health()

        # Auth
        print("\n--- Authentication ---")
        self.test_register_company()
        self.test_login()

        if not self.jwt_token:
            print("\n[ERROR] Cannot proceed without authentication")
            return False

        self.test_get_me()

        # Organization
        print("\n--- Organization ---")
        self.test_create_department()
        self.test_list_departments()

        # Employees
        print("\n--- Employee Management ---")
        self.test_create_employee()
        self.test_list_employees()

        # Summary
        print("\n" + "=" * 60)
        print("TEST SUMMARY")
        print("=" * 60)

        passed = sum(1 for r in self.results if r.passed)
        failed = sum(1 for r in self.results if not r.passed)
        total = len(self.results)

        print(f"Total:  {total}")
        print(f"Passed: {passed}")
        print(f"Failed: {failed}")

        if failed > 0:
            print("\nFailed tests:")
            for r in self.results:
                if not r.passed:
                    print(f"  - {r.name}: {r.message}")

        print("=" * 60)

        return failed == 0


def main():
    parser = argparse.ArgumentParser(description='Comprehensive API Tests for Strapi')
    parser.add_argument(
        '--url',
        default=BASE_URL,
        help=f'Strapi base URL (default: {BASE_URL})'
    )
    parser.add_argument(
        '-v', '--verbose',
        action='store_true',
        help='Verbose output'
    )

    args = parser.parse_args()

    tester = APITester(base_url=args.url, verbose=args.verbose)
    success = tester.run_all_tests()

    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()
