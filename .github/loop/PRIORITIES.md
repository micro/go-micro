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

1. **Stabilize AtlasCloud guarded delegate conformance** ([#3917](https://github.com/micro/go-micro/issues/3917)) — #3937 added a guarded-delegate retry, but the live AtlasCloud/minimax-m3 run recurred after that merge. Because this is still a Now-phase cross-provider conformance gate and directly protects the services → agents → workflows promise under a real model, it returns to the top until the harness proves the delegate attempt deterministically reaches the approval refusal path. #3942 appears to have addressed the separate tagged text task-call blocker from #3935, so that done work should not remain ahead of an open recurring failure.
2. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — once the live conformance gate is stable, streaming is the highest developer-visible Next-phase seam. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
3. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
