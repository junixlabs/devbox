---
name: research-content-writer
description: SEO blog writing partner for researching, outlining, drafting, and refining blog posts optimized for search engines and AI systems. Use when writing SEO blog posts, how-to guides, explainers, listicles, comparisons, product reviews, checklists, case studies, thought leadership, or newsletters. Also use for keyword research, SERP analysis, entity optimization, content outlines, writing quality audits, adding citations, improving hooks, or any SEO content creation task.
---

# Research Content Writer

Collaborative SEO blog writing partner. Research → Outline → Draft → Polish → Deliver.

## Auto-Flow

| Stage | Action | References |
|-------|--------|------------|
| 1 | Brand context | Gather name, differentiator, voice, audience |
| 2 | Research | web_search keyword, web_fetch 2-3 competitors |
| 3 | Outline + check | Select type from [templates.md](references/templates.md), structure H2s |
| 4 | Draft + check | Apply [writing-rules.md](references/writing-rules.md) |
| 5 | Polish + final | Links, metadata, anti-redundancy per [workflow.md](references/workflow.md) |
| 6 | **Sync to CMS** | Save outputs to keyword folder, run sync script (see below) |

Additional references loaded as needed:
- [anti-patterns.md](references/anti-patterns.md) — banned phrases, patterns, dash rules
- [tone-style.md](references/tone-style.md) — conversational tone, academic drift, voice preservation
- [unique-angle.md](references/unique-angle.md) — multi-page deduplication system
- [quality-checks.md](references/quality-checks.md) — stage-specific checklists
- [templates-extra.md](references/templates-extra.md) — comparison, review, case study, newsletter templates

## Core Philosophy

- **Golden rule:** Every paragraph: named entity + specific attribute + verifiable value.
- **Research rule:** Never write without keyword research and SERP analysis first.
- **Information gain:** 1-2 elements the existing SERP does not offer.
- **People-first:** "Would I write this if search engines didn't exist?"
- **Voice:** Preserve user's voice. Suggest, don't dictate.

## Article Type Selection

Search keyword first, analyze SERP, then choose format:

| SERP Pattern | Article Type |
|--------------|-------------|
| "how to" | How-to Guide |
| "what is" | Explainer |
| "best", "top" | Listicle |
| Two options compared | X vs Y |
| "[product] review" | Product Review |
| "checklist" | Checklist |

## Stage 6: Sync to CMS (MANDATORY)

After fatal checks pass, save all outputs:

```
.claude/skills/research-content-writer/output/<keyword-slug>/
  article.md    # complete finished article
  meta.json     # SEO metadata and sources
  body.md       # article body only (H2 onward)
```

Run sync: `node scripts/sync-to-cms.mjs --instance-id <id> --api-key <key>`

## Fatal Checks (run at every stage)

1. Zero dashes (em/en) in body
2. Zero banned AI phrases
3. Zero banned patterns (whether...or, not only...but also)
4. No "Introduction" heading
5. Exactly one H1
6. H1 → H2 → H3, never skip
