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

1. **Make the AtlasCloud plan/delegate harness wait for the delegated notify side effect** ([#3918](https://github.com/micro/go-micro/issues/3918)) — #3920 shipped the first-agent on-ramp, so the queue can drop the completed #3914 item. The freshest master signal is now a live provider-conformance failure where the plan/delegate flow returns before the comms agent produces its required notification. This is Now-phase hardening, protects the CI gate that lets the loop merge safely, and keeps the 0→hero services → agents → workflows story from drifting under a real provider.
2. **Make AtlasCloud conformance deterministically exercise guarded delegation** ([#3917](https://github.com/micro/go-micro/issues/3917)) — the second live AtlasCloud failure is also Now-phase conformance work: the agent calls the echo tool but does not attempt the blocked delegate path. Fixing this keeps guardrail/delegation behavior provider-portable without changing public APIs, and it prevents the harness from claiming plan/delegate safety that the live model path did not actually prove.
3. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — once the live gate is stable, streaming is the highest developer-visible Next-phase seam. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
4. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
