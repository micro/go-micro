# North Star

The direction the loop aligns every increment to. Depth lives in
[`internal/docs/THESIS.md`](../../internal/docs/THESIS.md); this is the short,
operative version the planner and builder read each run.

## Mission

Make building an **agent** as easy as building a **service**, on one runtime.
Go Micro is a holistic agent harness and service framework encapsulating the
lifecycle of **services → agents → workflows** — pluggable, progressive, and
AI-native by default.

## Right now — developer adoption

The framework's depth is strong; the **on-ramp** is the gap. Weight the developer
experience — a walkable first-agent tutorial, discoverable examples, docs
wayfinding, install friction, debugging, the 0→1 and 0→hero path — **at least as
highly as internal hardening**. A developer succeeding on their first agent
matters more right now than another conformance/observability/interop increment.
Do not let the queue fill entirely with internal depth work.

## Guardrails

- One concern per PR; small and reversible.
- The gate is green CI (`go build`, `go test`, `golangci-lint`, `make harness`),
  not human review — keep the suite strong; the loop is only as good as its evaluator.
- **Off-limits without a human** (surface as notes, never auto-merge): breaking
  public-API changes, brand/positioning/marketing copy, new dependencies,
  architectural rewrites, product-default changes with broad behavioral impact.
- Stay on `claude/*` / `codex/*` branches; base PRs on `master`. See
  [`CODEX.md`](../../CODEX.md) and [`internal/docs/CONTINUOUS_IMPROVEMENT.md`](../../internal/docs/CONTINUOUS_IMPROVEMENT.md).
