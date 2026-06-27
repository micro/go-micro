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

## Now (ranked)

1. **End-to-end agent streaming** (#3200) — complete the roadmap streaming slice:
   usable `ai.Stream` beyond the current OpenAI path, A2A `message/stream` chunk
   delivery, cancellation/error semantics, and docs so chat and long-task UX are
   real across the harness boundary. Roadmap → *Next* streaming, but it now ranks
   first because durable resume, provider conformance, registry readiness, 0→hero,
   and agent OpenTelemetry have shipped; streaming is the largest remaining seam
   in the services → agents → workflows interaction loop.
2. **Execution lifecycle hooks & metadata** (#2980) — before/after-tool, retry,
   and failure hooks; first reconcile the issue with the shipped `AgentWrapTool`,
   structured refusal reasons, `RunInfo`, and OpenTelemetry spans, then close it or
   scope only a CI-verifiable gap that wrappers plus tracing cannot express.
   Roadmap → resilience/operability, but ranked after streaming because most of
   the originally requested lifecycle metadata now exists and the remaining value
   is validation/scoping rather than a missing harness primitive.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
