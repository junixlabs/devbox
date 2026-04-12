# Changelog Rules

Guidelines for maintaining the project changelog following [Keep a Changelog](https://keepachangelog.com/).

---

## Format

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- New feature description

### Changed
- Modified behavior description

### Fixed
- Bug fix description

---

## [1.0.0] - 2026-02-01

### Added
- Initial release features
```

---

## Section Types

| Section | Use When |
|---------|----------|
| **Added** | New features or capabilities |
| **Changed** | Changes to existing functionality |
| **Deprecated** | Features to be removed in future |
| **Removed** | Features removed in this release |
| **Fixed** | Bug fixes |
| **Security** | Security vulnerability fixes |

---

## When to Update

| Event | Action |
|-------|--------|
| Feature merged | Add to `[Unreleased]` → `Added` |
| Bug fixed | Add to `[Unreleased]` → `Fixed` |
| Behavior changed | Add to `[Unreleased]` → `Changed` |
| Release created | Move `[Unreleased]` to new version |

---

## Entry Format

### Good Examples

```markdown
### Added
- User can now export reports as PDF (#123)
- Dark mode support for dashboard

### Fixed
- Login redirect loop on expired sessions (#456)
- Incorrect total calculation in invoices
```

### Bad Examples

```markdown
### Added
- Fixed bug           # Wrong section
- Updated package     # Too vague
- PR #123             # Missing description
```

---

## Versioning

Follow [Semantic Versioning](https://semver.org/):

| Version | When |
|---------|------|
| **MAJOR** (X.0.0) | Breaking changes |
| **MINOR** (0.X.0) | New features, backwards compatible |
| **PATCH** (0.0.X) | Bug fixes, backwards compatible |

---

## Release Workflow

1. Review `[Unreleased]` section
2. Determine version bump (major/minor/patch)
3. Create new version section with date
4. Move items from `[Unreleased]`
5. Commit with message: `chore: release vX.Y.Z`

```markdown
## [Unreleased]

(empty - ready for new changes)

---

## [1.2.0] - 2026-02-15

### Added
- Feature from unreleased
```

---

## Linking

Link version headers to compare URLs:

```markdown
## [1.2.0] - 2026-02-15

[1.2.0]: https://github.com/org/repo/compare/v1.1.0...v1.2.0
```
