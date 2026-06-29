# Priorities

The ranked work queue for the autonomous improvement loop. The
**architecture-review** pass (the *architect*) owns this file: each run it turns
the [roadmap](../../ROADMAP.md) plus an internal scan (gaps in the
services → agents → workflows lifecycle, API coherence, drift, tech debt, test and
DX friction) into a single ordered list — highest-value first — and links each
item to a tracking issue. The hourly **continuous-improvement** pass works the
**top item whose issue is still open**. So the architect decides *what*, and the
increment loop *builds* it.

**Reading / editing.** An item is done when its linked issue closes (the increment
that builds it adds `Closes #<issue>`). Roadmap phase (Now → Next → Later) is the
primary ordering; internal findings are interleaved by value, not kept in a
separate list. The human can reorder this list — or the issues — at any time to
redirect the loop; direction always wins.

**Off-limits to the loop** (the architect proposes these as notes, never as queue
items the loop can auto-merge): brand/positioning copy, breaking public-API
changes, architectural rewrites. Those go to the human.

## Developer experience (ranked)

1. **Broaden provider-backed AI streaming coverage** ([#3315](https://github.com/micro/go-micro/issues/3315)) — durable agent resume APIs have shipped, so the highest-value Next-roadmap seam is making `ai.Stream` behave consistently across providers; `micro chat`, A2A streaming, and long-running agent interactions should not depend on adapter-specific gaps.
2. **Emit agent RunInfo as OpenTelemetry spans** ([#3316](https://github.com/micro/go-micro/issues/3316)) — once streaming paths are consistent, turn agent run timelines into correlated spans so operators can inspect model calls, tool calls, failures, and run IDs through the same observability surface as services and flows.
3. **Add memory compaction and retrieval for long-running agents** ([#3321](https://github.com/micro/go-micro/issues/3321)) — after the Next-roadmap streaming and observability seams, tackle the leading Later-roadmap gap: bounded, store-backed memory that keeps long agent runs useful without making Go Micro a prompt-layer framework.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
