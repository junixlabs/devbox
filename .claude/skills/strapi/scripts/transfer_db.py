#!/usr/bin/env python3
"""
Transfer Strapi database from remote (production) to local using transfer tokens.

Reads TRANSFER_REMOTE_URL and TRANSFER_TOKEN from backend/.env by default.

Usage:
    python3 transfer_db.py [--backup] [--force]
    python3 transfer_db.py --from <url> --token <token>
    python3 transfer_db.py --exclude files

Examples:
    python3 transfer_db.py --backup --force
    python3 transfer_db.py --from https://prod.example.com/admin --token abc123
    python3 transfer_db.py --exclude files

Notes:
    - Set TRANSFER_REMOTE_URL and TRANSFER_TOKEN in backend/.env
    - Or create a Pull transfer token on remote Strapi admin:
      Settings → Transfer Tokens → Create (type: Pull)
    - Local DB (SQLite) will be overwritten
    - Strapi handles PostgreSQL → SQLite conversion automatically
"""

import argparse
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

BACKEND_DIR = Path(__file__).resolve().parents[4] / "forge/strapi"
DB_PATH = BACKEND_DIR / "database" / "data.db"
ENV_PATH = BACKEND_DIR / ".env"


def load_env_var(name: str) -> str:
    """Read a variable from backend/.env file."""
    if not ENV_PATH.exists():
        return ""
    for line in ENV_PATH.read_text().splitlines():
        line = line.strip()
        if line.startswith(f"{name}=") and not line.startswith("#"):
            return line.split("=", 1)[1].strip().strip('"').strip("'")
    return ""


def backup_local_db():
    """Backup current local database."""
    if not DB_PATH.exists():
        print("No local database found, skipping backup.")
        return None

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_path = DB_PATH.with_name(f"data.db.bak_{timestamp}")
    shutil.copy2(DB_PATH, backup_path)
    size_mb = backup_path.stat().st_size / (1024 * 1024)
    print(f"Backed up to {backup_path.name} ({size_mb:.1f} MB)")
    return backup_path


def run_transfer(remote_url: str, token: str = None, force: bool = False, exclude: str = None):
    """Run strapi transfer command."""
    cmd = ["npx", "strapi", "transfer", "--from", remote_url]

    if force:
        cmd.append("--force")

    if token:
        cmd.extend(["--from-token", token])

    if exclude:
        cmd.extend(["--exclude", exclude])

    # Mask token in printed command
    display_cmd = [c if c != token else "***" for c in cmd]
    print(f"\nRunning: {' '.join(display_cmd)}")
    print("=" * 60)

    result = subprocess.run(cmd, cwd=BACKEND_DIR)

    if result.returncode != 0:
        print(f"\nTransfer failed with exit code {result.returncode}")
        sys.exit(1)

    print("\nTransfer complete!")


def post_transfer():
    """Clean cache and rebuild after transfer."""
    cache_dir = BACKEND_DIR / ".cache"
    if cache_dir.exists():
        shutil.rmtree(cache_dir)
        print("Cleared .cache directory")

    print("Running strapi build...")
    subprocess.run(["npx", "strapi", "build"], cwd=BACKEND_DIR)


def main():
    parser = argparse.ArgumentParser(description="Transfer remote Strapi DB to local")
    parser.add_argument("--from", dest="remote_url",
                        help="Remote Strapi admin URL (default: TRANSFER_REMOTE_URL from .env)")
    parser.add_argument("--token",
                        help="Transfer token (default: TRANSFER_TOKEN from .env)")
    parser.add_argument("--backup", action="store_true", help="Backup local DB before transfer")
    parser.add_argument("--force", action="store_true", help="Overwrite without confirmation")
    parser.add_argument("--exclude", choices=["files", "config", "content"],
                        help="Exclude a data type from transfer")
    parser.add_argument("--no-build", action="store_true", help="Skip post-transfer build")

    args = parser.parse_args()

    # Resolve from .env if not provided via CLI
    remote_url = args.remote_url or load_env_var("TRANSFER_REMOTE_URL")
    token = args.token or load_env_var("TRANSFER_TOKEN")

    if not remote_url:
        print("Error: No remote URL. Set TRANSFER_REMOTE_URL in backend/.env or use --from")
        sys.exit(1)

    if not token:
        print("Warning: No transfer token. Set TRANSFER_TOKEN in backend/.env or use --token")
        print("Strapi will prompt for the token interactively.\n")

    print(f"Source: {remote_url}")
    print(f"Target: {DB_PATH}")
    print(f"Token:  {'***' + token[-4:] if token and len(token) > 4 else '(interactive)'}")

    if args.backup:
        backup_local_db()

    if not args.force:
        confirm = input("\nThis will overwrite your local database. Continue? [y/N] ")
        if confirm.lower() != "y":
            print("Aborted.")
            sys.exit(0)

    run_transfer(remote_url, token, force=True, exclude=args.exclude)

    if not args.no_build:
        post_transfer()


if __name__ == "__main__":
    main()
