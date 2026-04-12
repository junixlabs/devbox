# Decision Framework

## Auto-approve (no escalation)
- Plan tier is Medium or Lightweight
- All blockers resolved (at `tested`, `pass`, `staging`, `released`, or `closed`)
- Plan has clear steps and matches existing project patterns
- QA report shows all acceptance criteria PASS

## Escalate to owner
- Plan changes architecture or adds new external dependencies
- QA has failures requiring design decisions
- Issue scope grew beyond original intent
- Conflicting approaches between issues
- Any security concern

## Phase Gate Rules
Phase Review issues (ISS-30, ISS-38, ISS-45, ISS-50, ISS-54):
- Always create detailed review comment summarizing all work in the phase
- Recommend to owner whether to proceed
- **Owner must explicitly approve** before merging to main
- Do NOT auto-advance Phase Review issues

## Review Quality Checklist
When reviewing `waiting` issues, verify in plan:
- [ ] Approach is clear and specific (not vague)
- [ ] Affected files are listed
- [ ] Implementation steps are defined and ordered
- [ ] Risks are noted with mitigations
- [ ] No architecture changes without escalation
- [ ] No new external dependencies without escalation

## Status Transition Rules
- `waiting` → `approved`: only after plan review passes
- `tested` → `pass`: only at Phase Review, by owner
- `open` (unblocked) → trigger triage via agent session
- `reopen` → trigger fix via agent session
