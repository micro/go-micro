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

1. **Stabilize first-agent example broker setup under race test** ([#3977](https://github.com/micro/go-micro/issues/3977)) — #3975 shipped the examples wayfinding work and closed #3969, but the follow-on CI signal shows the first-agent example can still fail under the race/coverage harness because broker state is not isolated. This is both adoption-facing and evaluator-facing: the smallest no-secret first-agent path must be reliable before deeper internal work takes the top slot.
2. **Require delegated notify before plan-delegate completion** ([#3972](https://github.com/micro/go-micro/issues/3972)) — the latest live AtlasCloud plan/delegate harness found a tasks-complete-but-notify-missing failure after #3964/#3967 landed. This is a CI/evaluator trust issue in a core agents → workflows seam, so it should interrupt the Next-phase queue, but stay behind the first-agent reliability item to avoid letting internal conformance fully crowd out the on-ramp.
3. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — with the AtlasCloud plan/delegate conformance issue closed by #3964 and the on-ramp harness closed by #3967, streaming remains the highest developer-visible Next-phase seam once the new first-agent and delegated-notify regressions are fixed. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
4. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
