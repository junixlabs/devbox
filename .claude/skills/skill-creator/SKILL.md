---
name: skill-creator
description: Guide for creating effective skills, adding skill references, skill scripts or optimizing existing skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, frameworks, libraries or plugins usage, or API and tool integrations.
license: Complete terms in LICENSE.txt
---

# Skill Creator

This skill provides guidance for creating effective skills.

## About Skills

Skills are modular, self-contained packages that extend Claude's capabilities by providing specialized knowledge, workflows, and tools.

**IMPORTANT:**
- Skills are not documentation, they are practical instructions for Claude Code to use tools, packages, plugins or APIs.
- Each skill teaches Claude how to perform a specific development task.
- Claude Code can activate multiple skills automatically.

### Anatomy of a Skill

```
.claude/skills/
└── skill-name/
    ├── SKILL.md (required)
    │   ├── YAML frontmatter (name, description, version)
    │   └── Markdown instructions
    └── Bundled Resources (optional)
        ├── scripts/          - Executable code (Python/Bash/etc.)
        ├── references/       - Documentation loaded into context as needed
        └── assets/           - Files used in output (templates, icons, fonts)
```

### Requirements

- `SKILL.md` should be **less than 100 lines**
- Each script or referenced markdown file should be **less than 100 lines** (**progressive disclosure**)
- Descriptions should be concise but include enough use cases for auto-activation

## Skill Creation Process

### Step 1: Understand Concrete Examples
Clarify usage patterns: what triggers the skill, what does the user ask, what should Claude do?

### Step 2: Plan Reusable Contents
For each example, identify what scripts, references, and assets would help.

### Step 3: Initialize
```bash
scripts/init_skill.py <skill-name> --path <output-directory>
```

### Step 4: Edit the Skill
- Start with reusable contents (`scripts/`, `references/`, `assets/`)
- Update SKILL.md using imperative/infinitive form
- SKILL.md = quick reference guide, details go in references/

### Step 5: Package
```bash
scripts/package_skill.py <path/to/skill-folder>
```

### Step 6: Iterate
Use on real tasks, notice inefficiencies, update and re-test.

## Progressive Disclosure

Skills use a 3-level loading system:
1. **Metadata** (name + description) — always in context
2. **SKILL.md body** — when skill triggers
3. **Bundled resources** — loaded as needed by Claude

## References

- `scripts/init_skill.py` — Initialize new skill from template
- `scripts/package_skill.py` — Package skill for distribution
- `scripts/quick_validate.py` — Validate skill structure
