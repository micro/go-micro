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

1. **Make AtlasCloud conformance deterministically request the echo tool** ([#3924](https://github.com/micro/go-micro/issues/3924)) — #3926 shipped the previous plan/delegate notify wait fix, so the queue should drop the completed #3918 item. The freshest live harness signal now fails even earlier: AtlasCloud/minimax-m3 completes without invoking the required `conformance_echo` tool. This is the highest Now-phase item because the getting-started and 0→hero contract depends on providers actually using service tools, not just returning prose, and the CI gate needs an actionable deterministic failure path.
2. **Make AtlasCloud conformance deterministically exercise guarded delegation** ([#3917](https://github.com/micro/go-micro/issues/3917)) — after the echo-tool path is stable, the same live provider must attempt the blocked delegate path so guardrails are proven under a real model, not only mocks. This keeps plan/delegate safety provider-portable without public API changes and protects the services → agents → workflows story from drifting at the delegation seam.
3. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — once the live gate is stable, streaming is the highest developer-visible Next-phase seam. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
4. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
