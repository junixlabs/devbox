# Common Fixes

## Agent calls graphql_schema every conversation
- Check Qdrant for `mcp_schema` entries — missing = trigger sync
- Reranker boost is +0.20 for mcp_schema (highest)
- Force-inject in chat-prompt-builder ensures schema in context for MCP projects
- If schema sections are stale, re-sync: `POST /api/knowledge/sync`

## N+1 graphql_query calls (same query, different IDs)
- Add alias batching rule to behaviorRules with **specific example** using actual query names
- Generic: "NEVER loop graphql_query. Use aliases: `{ a1: query(id:1){...} a2: query(id:2){...} }`"
- Batch up to 20 aliases per call

## 502 timeout (>100s Cloudflare limit)
- Agent doing too many iterations before responding
- Add behaviorRule with the efficient pattern: "1-2 GQL calls + code_run"
- Ensure `code_run` in `enabledTools`
- Check if query needs a new aggregation endpoint on the portal side

## Empty reply (0 chars)
- Check `qualitySignals.iterationLogs` in chat-log DB for the failing query
- Common causes:
  1. **Large tool results** — GQL returns 30k+ chars, fills context, no room for text output
  2. **TodoWrite burns iterations** — agent plans instead of acting (stream:false has no UI benefit)
  3. **stopReason=max_tokens** on final iter — output budget spent on tool_use JSON
  4. **stopReason=end_turn with textChars=0** — model chose to stop without text (model bug)
- Diagnosis query: `GET /api/chat-logs?sort=createdAt:desc&filters[projectSlug][$eq]=<slug>` (JWT auth)
- Fix: add rule to fetch minimal fields, then aggregate with code_run
- Prefer `language: "javascript"` (python3 sometimes not found on prod)

## Tool patterns not stored
- Check `extractToolPatterns` wired in `chat.ts` and `channels/handler.ts` post-turn
- Only stores successful `*__graphql_query` calls with result > 50 chars
- Dedup at 0.85 cosine — similar queries won't duplicate (expected)

## Knowledge graph edges empty for MCP projects
- Normal for projects without Forge issues
- Edges from chat only appear when users discuss structural relationships
- Tool patterns (in Qdrant) are more valuable for MCP query optimization

## Rule writing tips
- Reference exact query names and field names from the schema
- Include a working GQL snippet, not just "use the right query"
- One concern per rule — don't mix alias batching with field naming
- Test the rule by running the exact GQL in the snippet before adding
