---
name: forge-skill
description: Forge skill management — push, pull, and sync skills to the Forge platform.
---

# Forge Skill

Manage skills on the Forge platform.

## Usage

Use the `forge_skills` MCP tool to manage skills:

```
forge_skills → list          # List all skills
forge_skills → get           # Get skill by name
forge_skills → push          # Push skill to Forge
forge_skills → pull          # Pull skills to local
forge_skills → sync          # Sync to devices
forge_skills → check         # Check versions
forge_skills → changelog     # Version history
forge_skills → rollback      # Revert to previous version
```

## Pushing a Skill

```
forge_skills → push → {
  name: "<skill-name>",
  skillMd: "<content of SKILL.md>",
  description: "<optional override>",
  files: [{ path: "references/file.md", content: "...", encoding: "utf8" }]
}
```

## Pulling Skills

```
forge_skills → pull → { skillsDir: ".claude/skills" }
```

This generates a local sync script. Run it to write skill files to disk.
