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

1. **Require delegated notify before plan-delegate completion** ([#3972](https://github.com/micro/go-micro/issues/3972)) — #3981 closed the first-agent broker isolation gap, leaving this as the latest live AtlasCloud plan/delegate harness regression. A tasks-complete-but-notify-missing run undermines evaluator trust at the agents → workflows seam, so it stays at the top until the harness can prove delegated work only completes after the required notification.
2. **Surface the first-agent and 0→hero example paths in the CLI** ([#3983](https://github.com/micro/go-micro/issues/3983)) — the README, website docs, and examples now describe a strong on-ramp, but adoption still depends on users finding those paths after install. Current goal is developer adoption, so the queue should keep a CI-verifiable CLI wayfinding task near the top instead of drifting entirely into internal conformance and observability work.
3. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — streaming remains the highest developer-visible Next-phase seam after the current conformance and wayfinding gaps. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
4. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
