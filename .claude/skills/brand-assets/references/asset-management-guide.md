# Asset Management Guide

Best practices for organizing, naming, and versioning project branding assets.

## Directory Structure

Store brand assets under `docs/brand/` with a clear hierarchy:

```
docs/brand/
├── brand-guide.md          # Master brand guidelines document
├── colors.md               # Color palette definitions
├── typography.md           # Typography system
├── imagery.md              # Visual style and imagery direction
├── iterations/             # Historical iteration snapshots
│   ├── v1-initial.md
│   ├── v2-refined.md
│   └── v3-final.md
└── assets/                 # Generated or referenced visual files
    ├── logos/
    ├── icons/
    └── samples/
```

## Naming Conventions

- Use lowercase kebab-case for all files and directories
- Prefix iteration snapshots with version number: `v1-`, `v2-`, `v3-`
- Name assets descriptively: `primary-logo-dark.svg`, not `logo2.svg`
- Include color mode when relevant: `*-light`, `*-dark`

## Usage Guidelines

- **Primary palette** — use for main UI surfaces, CTAs, and key brand touchpoints
- **Secondary palette** — supporting elements, backgrounds, section differentiation
- **Accent colors** — sparingly, for highlights, alerts, and interactive states
- **Neutral palette** — text, borders, backgrounds, and structural elements

Always reference colors by their semantic name (e.g., `brand-primary`) rather than raw hex values in implementation code.

## Versioning

Track brand evolution through iteration snapshots:

- **Save each iteration** — copy the current state to `iterations/vN-description.md` before starting a new round of refinement
- **Tag major changes** — when the palette, typography, or visual direction shifts significantly, bump the version number
- **Lock approved elements** — once the user approves an element (e.g., the color palette), mark it as locked in the brand guide to prevent regression in later iterations
- **Date entries** — include the date in iteration files for historical context

## When to Lock vs. Iterate

| Signal | Action |
|--------|--------|
| User says "I love this palette" | Lock colors, continue iterating typography |
| Feedback is minor tweaks | One more iteration, then lock |
| User wants to start over | Archive current as a version, restart |
| No feedback on an element | Treat as soft approval, confirm before locking |
