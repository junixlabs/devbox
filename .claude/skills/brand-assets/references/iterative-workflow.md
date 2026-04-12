# Iterative Workflow Details

How the AI-driven brand suggestion process works across multiple iterations, including context carrying, prompt structure, and convergence.

## Context Carrying Between Iterations

Each iteration builds on everything that came before. The AI does not start fresh — it accumulates:

1. **Approved elements** — locked decisions from previous rounds (e.g., "navy #1B2A4A approved as primary color in iteration 2")
2. **Rejected directions** — what didn't work and why (e.g., "pastel palette rejected — too soft for fintech audience")
3. **User preferences** — expressed tastes and patterns (e.g., "user prefers high contrast, dislikes gradients")
4. **Refinement trajectory** — the direction changes are moving (e.g., "each round gets bolder and more geometric")

When generating the next iteration, reference this accumulated context explicitly so suggestions converge toward the user's vision rather than exploring randomly.

## Prompt Templates

### Initial Generation (Iteration 1)

```
Based on the following brand parameters:
- Industry: {industry}
- Mood: {mood}
- Audience: {audience}
- Constraints: {constraints}
- Existing assets: {existing}

Generate brand element suggestions for:
1. Color palette (primary, secondary, accent, neutrals) with hex codes and rationale
2. Typography (heading + body font pairing) with weight/size recommendations
3. Imagery style direction (illustration style, photo treatment, icon approach)
4. Brand voice keywords (3-5 adjectives)
```

### Refinement (Iteration 2+)

```
Previous iteration context:
- Approved: {locked_elements}
- Rejected: {rejected_with_reasons}
- User feedback: {verbatim_feedback}
- Iteration number: {n}

Refine the following elements (keep approved elements unchanged):
{elements_to_refine}

Explain what changed from the previous iteration and why.
```

### Final Consolidation

```
All brand elements are approved. Consolidate into a brand guide:
- Locked colors: {colors}
- Locked typography: {typography}
- Locked imagery direction: {imagery}
- Voice keywords: {keywords}

Write the complete brand specification with usage rules for each element.
```

## Convergence Criteria

Stop iterating when any of these are true:

- **Explicit approval** — user says they're happy with all elements
- **Diminishing changes** — the delta between iterations is cosmetic (shade adjustments, minor weight changes)
- **Element lock-out** — all major elements (palette, typography, imagery) are individually locked
- **Iteration ceiling** — after 4-5 rounds without convergence, summarize the options and ask the user to make final picks

## Handling Conflicting Feedback

When user feedback contradicts previous approved directions:

1. **Acknowledge the conflict** — "In iteration 2 you approved the warm palette, but this feedback suggests cooler tones. Should we unlock the palette?"
2. **Offer options** — present both directions with the conflicting element adjusted
3. **Don't silently regress** — never change a locked element without explicit user confirmation
