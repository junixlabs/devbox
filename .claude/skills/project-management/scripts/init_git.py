#!/usr/bin/env python3
"""
Initialize git repository with standard configuration.

Usage:
  python3 init_git.py <project-path>
"""

import argparse
import subprocess
import sys
from pathlib import Path


GITIGNORE_TEMPLATE = """# Dependencies
node_modules/
.pnp/
.pnp.js

# Build outputs
.next/
out/
build/
dist/
.strapi/

# Environment
.env
.env.local
.env.*.local

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Logs
*.log
npm-debug.log*

# Testing
coverage/
.nyc_output/

# Cache
.cache/
.eslintcache
*.tsbuildinfo
__pycache__/
*.py[cod]

# Database
*.sqlite
*.db

# Uploads
public/uploads/
uploads/

# Temporary
tmp/
temp/
"""


def run_command(cmd: list, cwd: str = None) -> tuple[bool, str]:
    """Run a shell command and return success status and output."""
    try:
        result = subprocess.run(
            cmd,
            cwd=cwd,
            capture_output=True,
            text=True
        )
        return result.returncode == 0, result.stdout + result.stderr
    except Exception as e:
        return False, str(e)


def is_git_repo(path: Path) -> bool:
    """Check if directory is a git repository."""
    return (path / ".git").exists()


def init_git(project_path: Path):
    """Initialize git repository."""
    print("  Initializing git repository...")
    success, output = run_command(["git", "init"], cwd=str(project_path))
    if not success:
        print(f"  ❌ Failed to init git: {output}")
        return False
    print("  ✅ Git repository initialized")
    return True


def create_gitignore(project_path: Path):
    """Create .gitignore file."""
    gitignore_path = project_path / ".gitignore"

    if gitignore_path.exists():
        print("  ⚠️  .gitignore already exists, skipping")
        return True

    gitignore_path.write_text(GITIGNORE_TEMPLATE)
    print("  ✅ Created .gitignore")
    return True


def create_initial_commit(project_path: Path):
    """Create initial commit."""
    # Stage all files
    success, _ = run_command(["git", "add", "."], cwd=str(project_path))
    if not success:
        print("  ⚠️  Failed to stage files")
        return False

    # Check if there are files to commit
    success, output = run_command(
        ["git", "status", "--porcelain"],
        cwd=str(project_path)
    )

    if not output.strip():
        print("  ⚠️  No files to commit")
        return True

    # Create commit
    success, output = run_command(
        ["git", "commit", "-m", "Initial commit\n\nProject initialized with documentation structure."],
        cwd=str(project_path)
    )

    if success:
        print("  ✅ Created initial commit")
    else:
        print(f"  ⚠️  Commit skipped: {output.strip()}")

    return True


def main():
    parser = argparse.ArgumentParser(description="Initialize git repository")
    parser.add_argument("path", help="Project path")
    parser.add_argument("--no-commit", action="store_true", help="Skip initial commit")
    args = parser.parse_args()

    project_path = Path(args.path).resolve()

    if not project_path.exists():
        print(f"❌ Directory not found: {project_path}")
        sys.exit(1)

    print(f"\n🔧 Initializing git for: {project_path}\n")

    # Check if already a git repo
    if is_git_repo(project_path):
        print("  ℹ️  Already a git repository")

        # Still create gitignore if missing
        create_gitignore(project_path)

        print("\n✅ Git configuration complete")
        return

    # Initialize git
    if not init_git(project_path):
        sys.exit(1)

    # Create .gitignore
    create_gitignore(project_path)

    # Create initial commit
    if not args.no_commit:
        create_initial_commit(project_path)

    print("\n✅ Git initialization complete")
    print("\nNext steps:")
    print("  1. Add remote: git remote add origin <url>")
    print("  2. Push: git push -u origin main")


if __name__ == "__main__":
    main()
