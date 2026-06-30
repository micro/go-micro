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

## Work queue (ranked)

1. **Add a scheduled agent run harness contract** ([#3486](https://github.com/micro/go-micro/issues/3486)) — with conformance, failure/cancellation hardening, checkpoint resume, memory summarization, and A2A continuity now shipped, the highest-value remaining gap is proving unattended agents as first-class operational-harness workloads. This should compose existing services, agents, flows, store-backed memory, verification, conformance, and run inspection so the scaffold → run → chat → inspect lifecycle extends to scheduled/looping work without adding a hosted scheduler, graph DSL, or breaking public APIs.

2. **Export agent `RunInfo` as OpenTelemetry spans** ([#3501](https://github.com/micro/go-micro/issues/3501)) — once agents can run unattended, operators need production-grade visibility that lines up with the existing tracing stack. Map run lifecycle, checkpoints/resume, tool/delegate steps, and terminal failure/cancellation metadata into OpenTelemetry rather than creating a separate observability surface.

3. **Broaden provider-backed `ai.Stream` conformance** ([#3502](https://github.com/micro/go-micro/issues/3502)) — A2A/chat streaming is now a visible UX seam, but the trust story depends on every provider adapter behaving consistently for streaming deltas, cancellation, and errors. Keep this as a conformance extension with mock/no-secret coverage plus provider-gated checks, so interop hardening stays CI-verifiable.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
