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

1. **Fix AtlasCloud agent conformance guarded delegation** ([#4296](https://github.com/micro/go-micro/issues/4296)) — The first-agent docs wayfinding guard shipped in #4303, so the top remaining Now-phase defect is the latest live provider conformance failure: AtlasCloud agent runs missing guarded delegate behavior within the retry budget. Guarded delegation is core to safe services → agents handoff and should be CI-verifiable without public API changes.
2. **Fix AtlasCloud A2A streaming fallback tool-call reporting** ([#4297](https://github.com/micro/go-micro/issues/4297)) — The same live provider run found the A2A streaming fallback path reporting neither tool-call evidence nor run metadata (`tool=false runInfo=false`). Rank it just behind guarded delegation: A2A streaming is an important interop seam, but the issue is narrower than the base agent conformance failure and should stay focused on surfacing tool/run evidence in the existing harness.
3. **Propagate cancellation and retry signals through provider model calls** ([#4273](https://github.com/micro/go-micro/issues/4273)) — After the current live AtlasCloud failures are stable, the remaining Now-phase resilience seam is provider-boundary cancellation/deadline/retry behavior across the agent loop. Keep it focused on fakes/tests and avoid public API or architecture changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
