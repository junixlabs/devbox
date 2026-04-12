# Config Review

## Fetch project agentConfig

```bash
curl -sS -g 'https://forge-api.sidcorp.co/api/projects?filters[slug][$eq]=<slug>&fields[0]=agentConfig&fields[1]=documentId' \
  -H 'X-Forge-API-Key: <api-key>'
```

## Update agentConfig

Requires JWT auth (API key is read-only for projects):

```bash
JWT=$(curl -sS -g 'https://forge-api.sidcorp.co/api/auth/local' \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"thanhnh","password":"Falconx@@123"}' | python3 -c "import json,sys; print(json.load(sys.stdin)['jwt'])")

curl -sS -g -X PUT "https://forge-api.sidcorp.co/api/projects/<DOC_ID>" \
  -H 'Content-Type: application/json' -H "Authorization: Bearer $JWT" \
  -d '{"data":{"agentConfig":{...}}}'
```

## Checklist

**behaviorRules** — most impactful for MCP projects:
- [ ] References exact GQL query names and field names (not generic advice)
- [ ] Includes working GQL snippets for common patterns
- [ ] Has alias batching rule with specific example
- [ ] Has aggregation strategy: "1-2 GQL calls + code_run (JS)"
- [ ] Domain terminology defined (acronyms, status mappings)

**queryStrategies** — all 6 intents covered:
- [ ] LOOKUP: mentions filters, fields, MCP tools
- [ ] SEARCH: mentions search parameter, code lookups
- [ ] SUMMARY: mentions statistics queries, date defaults
- [ ] CREATE: mentions confirmation before creating
- [ ] ACTION: mentions conversation history
- [ ] CHAT: no tool calls needed

**intentExamples** — 15-30 examples, multilingual if users speak multiple languages

**enabledTools** — `code_run` required for aggregations, `forge_language` for language detection

**Note**: Domain templates are copy-on-apply. Updating the seed template does NOT update existing projects. Always update the project's config directly via API.
