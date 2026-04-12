"""Shared utilities for integration checks."""

from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional


@dataclass
class Issue:
    """Represents an integration issue found during checks."""
    severity: str  # 'error', 'warning', 'info'
    category: str
    file: str
    line: Optional[int]
    message: str
    suggestion: Optional[str] = None


@dataclass
class CheckResult:
    """Accumulates issues found during integration checks."""
    issues: list[Issue] = field(default_factory=list)

    def add(self, severity: str, category: str, file: str, line: Optional[int],
            message: str, suggestion: Optional[str] = None):
        self.issues.append(Issue(severity, category, file, line, message, suggestion))

    def error(self, category: str, file: str, line: Optional[int], message: str, suggestion: Optional[str] = None):
        self.add('error', category, file, line, message, suggestion)

    def warning(self, category: str, file: str, line: Optional[int], message: str, suggestion: Optional[str] = None):
        self.add('warning', category, file, line, message, suggestion)

    def info(self, category: str, file: str, line: Optional[int], message: str, suggestion: Optional[str] = None):
        self.add('info', category, file, line, message, suggestion)


def find_project_root() -> Path:
    """Find project root containing backend/ and frontend/ dirs."""
    cwd = Path.cwd()
    while cwd != cwd.parent:
        if (cwd / 'backend').is_dir() and (cwd / 'frontend').is_dir():
            return cwd
        cwd = cwd.parent
    raise RuntimeError('Could not find project root with backend/ and frontend/')
