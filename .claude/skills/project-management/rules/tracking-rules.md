# Tracking Update Rules

Rules for when and how to update project tracking documents.

---

## Story Status Updates

| Event | Update | File |
|-------|--------|------|
| Sprint planning | Add stories to sprint | `sprints/sprint-XX.md` |
| Start work | Status → In Progress | `user-stories/xxx.md` |
| Open PR | Add PR link | `user-stories/xxx.md` |
| PR merged | Status → Done | `user-stories/xxx.md` |
| Deployed | Add deploy date | `user-stories/xxx.md` |

---

## Sprint Updates

| When | Action |
|------|--------|
| Sprint start | Create `sprint-XX.md` |
| Daily | Update story status in sprint file |
| Sprint end | Mark sprint complete, calculate velocity |
| Sprint end | Update roadmap progress |

---

## Changelog Updates

| When | Action |
|------|--------|
| Feature complete | Add to `[Unreleased]` section |
| Deploy to prod | Move to new version section |
| Bug fix | Add to `### Fixed` |
| Breaking change | Add to `### Changed` with migration notes |

---

## Roadmap Updates

| When | Action |
|------|--------|
| Milestone complete | Update status to ✅ |
| Progress change | Update progress bar |
| New quarter | Add new phase |
| Scope change | Update milestone items |

---

## Update Automation

### After PR Merge
```bash
# Update story status to Done
python3 scripts/update_status.py <story-id> done --pr <pr-number>
```

### After Deployment
```bash
# Update story status to Deployed
python3 scripts/update_status.py <story-id> deployed
```

---

## File Locations

| Document | Location |
|----------|----------|
| Roadmap | `docs/tracking/roadmap.md` |
| Current Sprint | `docs/tracking/sprints/current.md` → symlink |
| Sprint History | `docs/tracking/sprints/sprint-XX.md` |
| Changelog | `docs/tracking/changelog.md` |
| User Stories | `docs/user-stories/{module}/{id}.md` |
