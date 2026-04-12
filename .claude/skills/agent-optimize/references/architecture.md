# Architecture

## Data flow: Chat → Learning

```
Chat turn completes (fire-and-forget):
  ├── extractMemories() [LLM: gemini-flash]
  │     ├── facts[] → Qdrant memory (preference/correction/convention/tool_pattern)
  │     └── edges[] → Strapi DB knowledge_edges (subject→predicate→object)
  └── extractToolPatterns() [no LLM]
        └── successful GQL calls → Qdrant memory (tool_pattern)
```

## Data flow: Query → Context

```
User query → ragGate (intent + condense) → multiStrategySearch → reranker → context
  → schema force-inject if MCP project missing schema in top results
  → knowledge graph 1-hop edge expansion → edgeContext
  → system prompt = base rules + behaviorRules + queryStrategy + RAG context + edges
```

## Reranker boosts (source_type)

mcp_schema +0.20, skill +0.15, memory +0.12, knowledge +0.10, chat_session +0.08, issue +0.05

## Key files

| File | Purpose |
|------|---------|
| `services/agent/memory.ts` | extractMemories, addMemory, extractToolPatterns |
| `services/agent/memory-lifecycle.ts` | Prune stale memories/edges (6h cron) |
| `services/knowledge-graph/index.ts` | upsertEdge, queryEdges, extractIssueEdges |
| `services/embeddings/reranker.ts` | Source-type boosts |
| `services/agent/mcp-sync.ts` | MCP knowledge sync + schema sections |
| `api/chat/services/chat-prompt-builder.ts` | RAG retrieval, schema force-inject |
| `services/agent/system-prompt.ts` | Layered prompt builder |
| `bootstrap/seeds/domain-templates.ts` | Template defaults (copy-on-apply) |

All paths relative to `forge/strapi/src/`.
