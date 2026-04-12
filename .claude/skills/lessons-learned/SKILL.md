---
name: lessons-learned
description: Skill for adding technical insights to coding standards as WRONG/CORRECT examples. Use when discovering patterns, gotchas, or rules worth documenting.
---

# Lessons Learned

Capture technical insights as modular rule files (30-80 lines each).

## When to Use

- Discovered a non-obvious pattern or gotcha
- Found a rule that prevents common errors
- Learned framework-specific behavior

## Quick Reference

| Action | Command |
|--------|---------||
| Add rule | `python3 scripts/add_standard.py` |

## Target Files

Rules are stored in `.claude/skills/{stack}/rules/`:

| Stack | Examples |
|-------|----------|
| Next.js | `component-size.md`, `hooks-modular.md`, `page-extraction.md` |
| Strapi | `ctrl-tenant-context.md`, `svc-modular.md`, `svc-workflow-engine.md` |

## Adding a Rule

```bash
# Create a new rule file
python3 .claude/skills/lessons-learned/scripts/add_standard.py \
  --file "strapi/ctrl-new-pattern.md" \
  --title "New Controller Pattern" \
  --wrong "// bad pattern" \
  --correct "// good pattern" \
  --why "Prevents common errors"

# Append to existing rule file
python3 .claude/skills/lessons-learned/scripts/add_standard.py \
  --file "nextjs/hooks-modular.md" \
  --title "Query Key Pattern" \
  --wrong "queryKey: ['items']" \
  --correct "queryKey: itemKeys.list(filters)" \
  --why "Type-safe and consistent"
```

## Output Format

New file:
```markdown
# Rule Title

**Incorrect:**
```lang
code
```

**Correct:**
```lang
code
```

Why: explanation
```

Append:
```markdown
## Additional Pattern

**Incorrect:**
...
```
