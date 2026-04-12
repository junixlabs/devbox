---
name: agent-optimize
description: |
  Audit and optimize a Forge chat agent for MCP projects.
  Tests queries, checks schema/memory/patterns in Qdrant,
  updates behaviorRules, and validates improvements.
  Covers: schema cache, alias batching, tool patterns, knowledge graph.
user_invocable: true
argument_description: "<project-slug> [hubToken]"
---

# Agent Optimize

Audit + optimize a Forge MCP chat agent in 7 phases.

## Phases

1. **Baseline** — Run `/chat-eval` to get gap metrics
2. **Inspect** — Check Qdrant for schema sections, tool patterns, memories. See `references/qdrant-queries.md`
3. **Config** — Fetch project agentConfig, review behaviorRules/strategies. See `references/config-review.md`
4. **Test** — Run diverse queries (LOOKUP/SEARCH/SUMMARY/ACTION/CHAT). See `references/test-matrix.md`
5. **Fix** — Update behaviorRules via API for failed patterns. See `references/rule-writing.md`
6. **Slim** — Reduce tokens: compress behaviorRules + optimize RAG context. See `references/token-optimization.md`
7. **Re-test** — Same queries, compare iterations/GQL calls/schema calls
8. **Eval** — Run `/chat-eval` again, compare with baseline

## Key Targets

| Metric | Target |
|--------|--------|
| graphql_schema calls | 0 |
| Avg iterations | ≤6 |
| 502 timeouts | 0 |
| Empty replies | 0 |
| Tool patterns stored | Growing over time |
| behaviorRules chars | Minimized (no redundancy with Qdrant) |

## Known Projects

| Project | Slug | API Key | Hub Token |
|---------|------|---------|-----------||
| Light Human | `portal-lh` | `fk_0186f0a601488027ac1661698e9a6126e720274a47e96759` | `3291\|PNi8lVJ3PhALqYflnvvrZe33gHLOxCtqrobuVgPEcfcd762e` |

## Quick Commands

```bash
# Chat-eval baseline
cd forge/strapi && npx tsx scripts/eval-chat-gaps.ts --project <slug> --user thanhnh --pass 'Falconx@@123' --limit 200

# Test a query
curl -sS -g -X POST 'https://forge-api.sidcorp.co/api/chat' \
  -H 'Content-Type: application/json' -H 'X-Forge-API-Key: <key>' \
  --data-raw '{"message":"<query>","stream":false,"hubToken":"<token>"}' --max-time 180

# Trigger MCP sync
curl -sS -X POST 'https://forge-api.sidcorp.co/api/knowledge/sync' -H 'X-Forge-API-Key: <key>' -d '{"projectSlug":"<slug>"}'
```

## References

- `references/qdrant-queries.md` — Qdrant scroll/search commands for schema, memories, patterns
- `references/config-review.md` — How to fetch/update agentConfig, what to check
- `references/test-matrix.md` — Test query matrix and expected behavior per intent
- `references/architecture.md` — Memory/edge/pattern data flow and key files
- `references/common-fixes.md` — Solutions for schema loops, N+1, timeouts, empty replies
- `references/token-optimization.md` — Compress behaviorRules without losing performance
