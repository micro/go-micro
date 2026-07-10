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

1. **Complete AtlasCloud plan-delegate after successful notify timeout** ([#4573](https://github.com/micro/go-micro/issues/4573)) — The newly shipped provider-gated conformance matrix exposed the highest-value live Now failure: the 0→hero-style plan/delegate path can create the expected task and notification side effects, then keep retrying into an approval pause after AtlasCloud 408s. Put this first because it sits directly on the services → agents → workflows adoption story: side effects must finalize idempotently before developers can trust delegated agents under real provider latency.
2. **Finalize AtlasCloud universe notify after agent timeout** ([#4572](https://github.com/micro/go-micro/issues/4572)) — The same live conformance run showed the universe checkout flow can complete payment/order side effects but leave notify/tool-wrapper evidence and durable flow completion ambiguous after an agent-backed notify timeout. This is the next failure/resilience seam because it threatens the durable workflow contract before the A2A probe even runs.
3. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The universe checkout flow can complete, but the A2A reachability probe still intermittently times out under AtlasCloud. Keep this directly behind the earlier universe finalization defect: once the flow itself is durably complete, the agent must be reachable over the interop gateway without false negatives.
4. **Polish resume and inspect breadcrumbs for agent runs** ([#4569](https://github.com/micro/go-micro/issues/4569)) — Durable runs, streaming, and human-input pauses are now part of the core story, but the developer inner loop still has to make the next command obvious when an agent pauses or needs inspection. This keeps adoption pressure in the queue alongside hardening: chat → inspect → resume should feel like one walkable workflow, not an internal operations exercise.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
