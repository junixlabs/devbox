---
name: forge-release
description: "Merge approved issue code to production branch and trigger deployment. Use this skill when an issue has been approved at staging and needs to be released — squash-merges the ISS-* feature branch to the production branch, triggers Coolify deploy, and closes the issue. Triggers on: /forge-release, releasing an issue, merging to production, deploying to production. Also use when the pipeline needs to move an issue from released to closed."
user_invocable: true
arguments: "documentId"
---

# Forge Release

The final step in the issue pipeline: `released → closed`. Squash-merges the ISS-* feature branch to the production branch and triggers deployment.

The ISS-* branch is the single source of truth for the issue's changes. baseBranch (staging) may have commits from other issues — we never merge baseBranch to production. We merge ISS-* directly.

## Usage

```
/forge-release <documentId>
```

## Tools

- **forge_issues** — get issue data, update status
- **forge_comments** — post release comment
- **forge_config** — get baseBranch, productionBranch, Coolify config
- **forge_coolify_deploy** — trigger production deploy (if configured)
- **Bash** — git merge, push, branch cleanup

## Workflow

1. **Fetch Issue & Config** — verify status is `released`, read `productionBranch` (fallback: `master`) and `baseBranch`
2. **Confirm Git State** — `git branch --show-current && git status`. Stash if dirty.
3. **Check Sibling Issues** — if related issues on same branch not at `released`, stop
4. **Find ISS-* Branch** — `git fetch origin && git branch -r --list 'origin/ISS-*'`
5. **Diff Audit** — compare ISS-* against production, flag unexpected files
6. **Squash Merge to Production:**
   ```bash
   git checkout <productionBranch>
   git pull origin <productionBranch>
   git merge --squash origin/ISS-XX-short-title
   git commit -m "ISS-XX: <issue title>"
   git push origin <productionBranch>
   ```
7. **Deploy** — `forge_coolify_deploy → deploy → {}` if Coolify configured
8. **Clean Up Feature Branch** — `git push origin --delete ISS-XX-short-title`
9. **Post Comment & Close** — comment then `forge_issues → update → { status: "closed" }`

## Output Rules (Save Tokens)

- **Zero narration.** Just execute the steps.
- **One-line status only.** "Merged ISS-42 to master, deploy triggered. Closed." — nothing more.
