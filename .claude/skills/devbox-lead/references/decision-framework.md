# Decision Framework

## Auto-approve (no escalation)
- Plan tier is Medium or Lightweight
- All blockers resolved
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
When reviewing `tested` issues, verify in comments:
- [ ] Triage: complexity assessed, relations identified
- [ ] Plan: approach clear, files listed, steps defined, risks noted
- [ ] Code: branch created, files changed, tests written, build passes
- [ ] Review: code review done, findings addressed
- [ ] QA: all acceptance criteria tested, pass/fail documented

## Status Transition Rules
- `tested` issues: do NOT advance to `pass` — that happens at Phase Review
- `waiting` (complex): approve only if plan is solid and blockers resolved
- `open` (unblocked): trigger triage via agent session
- `reopen`: trigger fix via agent session
