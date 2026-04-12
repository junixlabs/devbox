#!/usr/bin/env python3
"""
Generate documentation from existing codebase.

Analyzes project structure and creates documentation scaffolding.

Usage:
  python3 generate_docs.py <project-path>
"""

import argparse
import json
from pathlib import Path
from typing import Optional

SKILL_DIR = Path(__file__).parent.parent


def detect_project_type(project_path: Path) -> dict:
    """Detect project type from files."""
    result = {
        "has_frontend": False,
        "has_backend": False,
        "frontend_type": None,
        "backend_type": None,
        "monorepo": False,
    }

    # Check for monorepo structure
    if (project_path / "frontend").exists() or (project_path / "backend").exists():
        result["monorepo"] = True

    # Check for Next.js
    nextjs_indicators = ["next.config.js", "next.config.mjs", "next.config.ts"]
    for indicator in nextjs_indicators:
        paths_to_check = [project_path, project_path / "frontend"]
        for check_path in paths_to_check:
            if (check_path / indicator).exists():
                result["has_frontend"] = True
                result["frontend_type"] = "nextjs"
                break

    # Check for Strapi
    strapi_indicators = [
        (project_path / "src" / "api"),
        (project_path / "backend" / "src" / "api"),
    ]
    for indicator in strapi_indicators:
        if indicator.exists():
            result["has_backend"] = True
            result["backend_type"] = "strapi"
            break

    # Check package.json for more info
    package_paths = [
        project_path / "package.json",
        project_path / "frontend" / "package.json",
        project_path / "backend" / "package.json",
    ]
    for pkg_path in package_paths:
        if pkg_path.exists():
            try:
                pkg = json.loads(pkg_path.read_text())
                deps = {**pkg.get("dependencies", {}), **pkg.get("devDependencies", {})}
                if "next" in deps:
                    result["has_frontend"] = True
                    result["frontend_type"] = "nextjs"
                if "@strapi/strapi" in deps:
                    result["has_backend"] = True
                    result["backend_type"] = "strapi"
            except (json.JSONDecodeError, IOError):
                pass

    return result


def analyze_frontend_structure(frontend_path: Path) -> dict:
    """Analyze Next.js frontend structure."""
    structure = {
        "app_routes": [],
        "features": [],
        "components": [],
        "has_src": (frontend_path / "src").exists(),
    }

    src_path = frontend_path / "src" if structure["has_src"] else frontend_path

    # Find app routes
    app_path = src_path / "app"
    if app_path.exists():
        for item in app_path.rglob("page.tsx"):
            route = str(item.parent.relative_to(app_path))
            if route != ".":
                structure["app_routes"].append(route)

    # Find features
    features_path = src_path / "features"
    if features_path.exists():
        structure["features"] = [d.name for d in features_path.iterdir() if d.is_dir()]

    # Find UI components
    ui_path = src_path / "components" / "ui"
    if ui_path.exists():
        structure["components"] = [
            f.stem for f in ui_path.glob("*.tsx") if f.stem != "index"
        ]

    return structure


def analyze_backend_structure(backend_path: Path) -> dict:
    """Analyze Strapi backend structure."""
    structure = {"apis": [], "services": [], "utils": []}

    src_path = backend_path / "src"

    # Find APIs
    api_path = src_path / "api"
    if api_path.exists():
        structure["apis"] = [d.name for d in api_path.iterdir() if d.is_dir()]

    # Find services
    services_path = src_path / "services"
    if services_path.exists():
        structure["services"] = [d.name for d in services_path.iterdir() if d.is_dir()]

    # Find utils
    utils_path = src_path / "utils"
    if utils_path.exists():
        structure["utils"] = [f.stem for f in utils_path.glob("*.ts")]

    return structure


def generate_project_structure_doc(
    project_info: dict, frontend: Optional[dict], backend: Optional[dict]
) -> str:
    """Generate project-structure.md content."""
    content = ["# Project Structure\n", "## Overview\n"]

    # Root structure
    if project_info["monorepo"]:
        content.append(
            """```
project/
├── frontend/          # Next.js App Router
├── backend/           # Strapi CMS
├── docs/              # Documentation
└── openspec/          # Change proposals
```

---
"""
        )
    else:
        content.append(
            """```
project/
├── src/               # Source code
├── docs/              # Documentation
└── openspec/          # Change proposals
```

---
"""
        )

    # Frontend section
    if frontend:
        content.append("## Frontend (Next.js)\n")
        if frontend.get("app_routes"):
            content.append("### Routes\n")
            for route in sorted(frontend["app_routes"]):
                content.append(f"- `/{route}`\n")
            content.append("\n")

        if frontend.get("features"):
            content.append("### Features\n")
            for feature in sorted(frontend["features"]):
                content.append(f"- `{feature}`\n")
            content.append("\n")

        if frontend.get("components"):
            content.append("### UI Components\n")
            content.append(f"Components: {', '.join(sorted(frontend['components'][:10]))}")
            if len(frontend["components"]) > 10:
                content.append(f" (+{len(frontend['components']) - 10} more)")
            content.append("\n\n---\n")

    # Backend section
    if backend:
        content.append("## Backend (Strapi)\n")
        if backend.get("apis"):
            content.append("### APIs\n")
            for api in sorted(backend["apis"]):
                content.append(f"- `api::{api}`\n")
            content.append("\n")

        if backend.get("services"):
            content.append("### Services\n")
            for service in sorted(backend["services"]):
                content.append(f"- `{service}`\n")
            content.append("\n")

        if backend.get("utils"):
            content.append("### Utilities\n")
            for util in sorted(backend["utils"]):
                content.append(f"- `{util}`\n")
            content.append("\n")

    content.append("---\n\n## File Naming\n")
    content.append(
        """
| Type | Convention | Example |
|------|------------|---------|
| Components | `kebab-case.tsx` | `employee-card.tsx` |
| Hooks | `use-{name}.ts` | `use-employees.ts` |
| Controllers | `{name}.ts` | `employee.ts` |
| Services | `{name}.ts` | `employee.ts` |
"""
    )

    return "".join(content)


def main():
    parser = argparse.ArgumentParser(description="Generate docs from existing codebase")
    parser.add_argument("path", help="Project path")
    args = parser.parse_args()

    project_path = Path(args.path).resolve()
    docs_path = project_path / "docs"

    print(f"\n🔍 Analyzing project: {project_path}\n")

    # Detect project type
    project_info = detect_project_type(project_path)
    print(f"  Monorepo: {project_info['monorepo']}")
    print(f"  Frontend: {project_info['frontend_type'] or 'Not detected'}")
    print(f"  Backend: {project_info['backend_type'] or 'Not detected'}")

    # Analyze structures
    frontend_structure = None
    backend_structure = None

    if project_info["has_frontend"]:
        frontend_path = (
            project_path / "frontend" if project_info["monorepo"] else project_path
        )
        frontend_structure = analyze_frontend_structure(frontend_path)
        print(f"\n  Frontend routes: {len(frontend_structure['app_routes'])}")
        print(f"  Frontend features: {len(frontend_structure['features'])}")

    if project_info["has_backend"]:
        backend_path = (
            project_path / "backend" if project_info["monorepo"] else project_path
        )
        backend_structure = analyze_backend_structure(backend_path)
        print(f"\n  Backend APIs: {len(backend_structure['apis'])}")
        print(f"  Backend services: {len(backend_structure['services'])}")

    # Create docs directory
    print("\n📁 Creating documentation...")
    docs_path.mkdir(exist_ok=True)

    # Generate project-structure.md
    structure_doc = generate_project_structure_doc(
        project_info, frontend_structure, backend_structure
    )
    structure_path = docs_path / "project-structure.md"
    structure_path.write_text(structure_doc)
    print(f"  Created: {structure_path}")

    # Create subdirectories
    for subdir in ["design-guidelines", "rules", "user-stories", "bugs"]:
        (docs_path / subdir).mkdir(exist_ok=True)

    print(f"\n✅ Documentation generated at {docs_path}")
    print("\nGenerated files:")
    print("  - docs/project-structure.md (auto-generated from codebase)")
    print("\nNext steps:")
    print("  1. Review and enhance docs/project-structure.md")
    print("  2. Add design-guidelines/design-system.md")
    print("  3. Add rules/code-standards.md")
    print("  4. Create user story modules with add_module.py")


if __name__ == "__main__":
    main()
