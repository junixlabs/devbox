# Test Matrix

Run 6-8 queries covering all intents. For each, check: iterations, GQL calls, schema calls, reply quality.

## Query template

```bash
curl -sS -g -X POST 'https://forge-api.sidcorp.co/api/chat' \
  -H 'Content-Type: application/json' -H 'X-Forge-API-Key: <key>' \
  --data-raw '{"message":"<query>","stream":false,"hubToken":"<token>"}' --max-time 180
```

Parse response: `iterations`, `toolCalls[].name`, `toolCalls[].isError`, `reply` length.

## Expected behavior per intent

| Intent | Max iters | Schema calls | Pattern |
|--------|-----------|-------------|---------|
| CHAT | 1 | 0 | No tool calls |
| LOOKUP | 2-3 | 0 | 1-2 GQL with filters |
| SEARCH | 2-3 | 0 | GQL with search param |
| SUMMARY | 3-5 | 0 | Statistics queries or batched aliases |
| SUMMARY-complex | 4-6 | 0 | Alias batch + code_run |
| ACTION | 1-3 | 0 | Uses conversation context |

## Red flags

- `graphql_schema` in toolCalls → schema cache not working
- Same GQL query called 4+ times → alias batching rule missing
- `code_run` with `language: "python"` failing → use JS instead
- Reply 0 chars → output token exhaustion, reduce data payload
- 502 response → >100s Cloudflare timeout, too many iterations
