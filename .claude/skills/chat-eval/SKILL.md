---
name: chat-eval
description: |
  Evaluate chat agent quality by analyzing chat logs for a project.
  Identifies knowledge gaps, tool errors, deflections, slow responses,
  and repeated questions. Outputs stats + actionable recommendations.
user_invocable: true
---

# Chat Eval Skill

Analyze chat logs for a Forge project to identify gaps and improve agent quality.

## Quick Start

```bash
cd forge/strapi
npx tsx scripts/eval-chat-gaps.ts --project <slug> --user thanhnh --pass 'Falconx@@123' [--limit 200] [--days 30]
```

## Known Projects

| Project | Slug | API Key | Service Token |
|---------|------|---------|---------------|
| Light Human | `portal-lh` | `fk_0186f0a601488027ac1661698e9a6126e720274a47e96759` | `3289\|5GxOjXcWVbxKmtUHNIDUL5eKeF1oefbR4u3A1wfQ73ebbde9` |

## What It Does

1. **Fetches chat logs** from Strapi API (paginated, JWT auth)
2. **Analyzes gaps** across categories: `error`, `tool_error`, `no_rag_context`, `weak_reply`, `deflection`, `excessive_iterations`, `slow_response`
3. **Finds repeated questions** across sessions (knowledge gaps)
4. **Computes stats**: intent/tool/source distribution, latency, token usage
5. **LLM-powered analysis** (optional, uses LiteLLM) for recommendations

## Auth Options

| Method | Flags | Notes |
|--------|-------|-------|
| JWT (preferred) | `--user <user> --pass <pass>` | Uses `/api/auth/local` |
| API key | env `EVAL_API_KEY` | Global forge API key |

## Priority Actions by Gap Category

1. **tool_error (high)** — Fix broken tool integrations
2. **error (high)** — Agent crashes. Check logs for stack traces
3. **no_rag_context (medium)** — Knowledge base missing content
4. **deflection (medium)** — Agent needs new tools or better system prompt
5. **weak_reply (medium)** — Check if RAG returned context but agent didn't use it
6. **excessive_iterations (medium)** — Agent looping on unexpected tool data
7. **slow_response (low)** — Check model latency, tool call chains

## Improvement Checklist

- [ ] Verify MCP service `apiKey` in project `mcpServers` config
- [ ] Test each MCP tool directly via curl
- [ ] Check `intentExamples` count — 15-30 examples covering all 6 intents
- [ ] Verify `queryStrategies` covers all 6 intents
- [ ] Add behavior rules referencing exact query names and fields
- [ ] Add FAQ entries for repeated questions found in eval

## Script Location

`forge/strapi/scripts/eval-chat-gaps.ts`
