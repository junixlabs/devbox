# Qdrant Queries

Qdrant URL: `http://216.18.217.213:6333`, API key in `forge/strapi/.env` → `QDRANT_API_KEY`.
Collection: `forge_embeddings`. Get project `documentId` from Strapi API first.

## Schema sections (mcp_schema)

```bash
curl -sS -X POST '$QDRANT_URL/collections/forge_embeddings/points/scroll' \
  -H 'Content-Type: application/json' -H "api-key: $QDRANT_API_KEY" \
  -d '{"filter":{"must":[
    {"key":"project_id","match":{"value":"<DOC_ID>"}},
    {"key":"source_type","match":{"value":"mcp_schema"}}
  ]},"limit":50,"with_payload":true,"with_vector":false}'
```

Each MCP server should have multiple sections (split by `## HEADER`). Single entry = re-sync needed.

## Memories (tool_pattern, correction, preference, convention)

```bash
# Same endpoint, filter source_type: "memory"
{"key":"source_type","match":{"value":"memory"}}
```

Check `metadata.category` distribution. Healthy project: 10+ tool_patterns after a few sessions.

## Source type reference

| source_type | Created by | Purpose |
|-------------|-----------|---------|
| `memory` | extractMemories, extractToolPatterns | Learned facts + working GQL patterns |
| `mcp_schema` | syncMcpKnowledge Phase 5 | GraphQL schema sections per MCP server |
| `hub_task/project/comment/config` | syncMcpKnowledge Phases 1-4 | MCP hub data |
| `issue/skill/knowledge/chat_session` | Forge lifecycle hooks | Core project data |
