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

1. **Add human-in-the-loop pause and resume for agent workflows** ([#3329](https://github.com/micro/go-micro/issues/3329)) — the highest-value remaining Later-phase operability gap now that provider conformance, failure hardening, the getting-started contract, streaming, durable resume/checkpoints, A2A task lifecycle, and `RunInfo` tracing have shipped. Scheduled and looping work still needs durable pending-input states that let humans approve, correct, or add context and then resume the same service/agent/workflow runtime.
2. **gateway/a2a: multiple typed skills per agent card** ([#3342](https://github.com/micro/go-micro/issues/3342)) — the A2A gateway now has the full task lifecycle (send/stream/get/cancel/resubscribe, push config, multi-turn, input-required); the remaining interop gap is skill granularity. `Card` only ever advertises one synthetic "chat" skill with services flattened into tags. Let an agent advertise N typed skills and route per skill, so domain-routing agents expose their real capabilities over A2A through the gateway instead of a hand-rolled handler.
3. **Add a maintained 0-to-hero reference example** ([#3368](https://github.com/micro/go-micro/issues/3368)) — keep the mission legible in code, not only docs: one CI-verifiable example should walk scaffold → run → chat → inspect across typed services, an agent, and a durable flow. This closes the remaining DX/coherence seam between the README promise, website roadmap, and the lived developer inner loop.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
