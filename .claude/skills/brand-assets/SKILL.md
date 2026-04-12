---
name: brand-assets
description: Generate and refine project branding assets through an iterative AI-driven feedback loop. Use this skill when the user asks to create brand identity, visual identity, brand guidelines, color palettes, typography systems, or brand kits. Each iteration builds on previous AI output and user feedback to progressively refine brand elements.
---

This skill guides the creation of project branding assets through an iterative AI suggestion process. Each cycle generates brand elements, collects user feedback, and refines the output — carrying context forward so suggestions improve with every iteration.

## When to Use

- User asks to create or update brand identity, brand guidelines, or a brand kit
- User wants AI-generated suggestions for colors, typography, imagery style, or logo concepts
- User needs to establish visual identity for a new project
- User wants to refine existing brand assets with AI assistance

## Iterative Workflow

### Step 1: Gather Brand Parameters

Collect initial context:
- **Industry/domain** — what sector or field
- **Brand mood** — desired feeling (bold & modern, warm & approachable, elegant & minimal)
- **Target audience** — who will see this
- **Existing assets** — any current colors, logos, fonts to respect or evolve
- **Constraints** — accessibility requirements, platform targets, print vs. digital

### Step 2: Generate Initial Suggestions

Produce the first round of brand elements:
- **Color palette** — primary, secondary, accent, and neutral colors with hex codes + rationale
- **Typography** — heading and body font pairings with weight/size recommendations
- **Imagery style** — visual direction for photos, illustrations, or icons
- **Voice keywords** — 3-5 adjectives that capture brand's visual personality

### Step 3: Collect User Feedback

Ask what they like, what doesn't feel right, and any new direction.

### Step 4: Refine with Context

Generate next iteration incorporating approved elements, feedback adjustments, and accumulated context.

### Step 5: Repeat Until Approved

Continue feedback-refine cycle (2-4 iterations typical).

### Step 6: Output Brand Spec

Write final approved brand assets to `docs/brand/`:
```
docs/brand/
├── brand-guide.md
├── colors.md
├── typography.md
└── imagery.md
```

## References

- `references/asset-management-guide.md` — Asset organization, naming, versioning
- `references/iterative-workflow.md` — Iteration mechanics, prompt templates, convergence
- `references/example-workflows.md` — Worked examples with 2-3 iteration cycles
