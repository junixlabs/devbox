# Token Optimization

Reduce token cost on two fronts: what's stored (embeddings) and what's sent to agent (retrieval).

## A. BehaviorRules Compression

Target: <3000 chars total.

| Rule content | Qdrant source | Action |
|-------------|---------------|--------|
| Inline status ID enums | `mcp_schema` ENUMS section | Remove IDs, keep name mappings only |
| Full GQL examples | `tool_pattern` memories | Compress to query signatures |
| "call graphql_schema" | Schema cache + force-inject | Change to "check schema in context" |

## B. RAG Context Optimization

**Key files:**
- `services/rag-formatter/index.ts` — formats entries before injection
- `services/agent/system-prompt.ts` `layerRelevantContext()` — MAX_CHARS=6000, 800c/entry

## C. Verification

After changes, test the same queries from Phase 4:
- Schema calls must stay at 0
- Iterations should not increase (≤2 more)
- Reply quality must not degrade
- Measure total inputTokens from response — should decrease
