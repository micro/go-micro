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

1. **Stabilize AtlasCloud guarded-delegation conformance** ([#4157](https://github.com/micro/go-micro/issues/4157)) — #4159 closed the built-in tool-schema 400s from #4151, but the remaining Now-phase provider-conformance gap is AtlasCloud intermittently missing the guarded `delegate` attempt/refusal in the live agent harness. Keep this scoped to deterministic harness/provider behavior so the service-backed agent path remains portable across providers without changing public APIs.
2. **Add durable agent checkpoint resume smoke coverage** ([#4148](https://github.com/micro/go-micro/issues/4148)) — once the live provider-conformance regression is contained, move to the top Next-phase harness gap: prove an interrupted agent run can resume from persisted state with enough run/step history for inspect/debugging. This keeps the lifecycle cohesive by giving agents the same durability story flows already have, without taking on a breaking API redesign.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
