# North Star

> **Edit this file.** It is the single source of direction the loop aligns every
> increment to. The planner ranks work against it; the builder builds toward it.
> Be concrete — vague direction produces vague increments.

## Mission

<One or two sentences: the problem this repository solves and who it's for.>

## Right now

<The current priority — what "better" means this month. The planner weights the
queue toward this.>

## Guardrails

- One concern per PR; small and reversible.
- The gate is green CI, not a human review — keep the test/lint suite strong,
  because the loop is only as good as its evaluator.
- **Off-limits without a human** (surface as notes, never auto-merge): breaking
  public API changes, brand/positioning/marketing copy, new dependencies,
  architectural rewrites, product-default changes with broad behavioral impact.
