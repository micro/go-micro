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

1. **Harden agent and flow failure semantics** ([#3461](https://github.com/micro/go-micro/issues/3461)) — scheduled cross-provider 0→hero conformance shipped in #3459, so the remaining Now-phase hardening gap is failure behavior: timeouts, cancellation, rate limits, and transient provider/tool failures must be predictable across agents, flows, and gateways. This is the highest-value queue item because the mission is not just making agents easy to start, but making the services → agents → workflows lifecycle operable when real distributed-system failure modes appear.

2. **Emit OpenTelemetry spans for agent RunInfo timelines** ([#3455](https://github.com/micro/go-micro/issues/3455)) — flows now have trace-oriented optimization feedback, durable agent runs can resume, and conformance can run on a schedule, but the agent side still needs first-class production traces. Translating `RunInfo` / run timeline events into spans closes a Next-phase observability seam across agent runs, tool calls, model calls, retries, failures, and checkpoint/resume events without changing public APIs.

3. **Complete end-to-end chat and A2A streaming coverage** ([#3456](https://github.com/micro/go-micro/issues/3456)) — provider streaming conformance and A2A fallback work recently shipped, but the mission is one runtime where agents can operate as services, which means streaming must be dependable through the whole path: provider tokens → chat / `Agent.Chat` → A2A. This remains behind failure semantics and observability because it is a Next-phase depth item, but it is the next user-visible seam in the developer inner loop and interop story.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
