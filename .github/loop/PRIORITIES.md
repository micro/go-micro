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

1. **Stabilize AtlasCloud guarded delegate conformance** ([#3917](https://github.com/micro/go-micro/issues/3917)) — #3948 strengthened AtlasCloud delegate prompting and tagged fallback coverage, but the issue remains open and this failure has recurred. Keep it first until master's live provider harness proves the guarded delegate reaches the approval-refusal path deterministically, then drop it immediately rather than re-queueing completed conformance work.
2. **Execute AtlasCloud plan-delegate task tool calls** ([#3935](https://github.com/micro/go-micro/issues/3935)) — #3942 appears to have moved the live plan-delegate path past the tagged task-call failure, and #3953 fixed the follow-on completion-state bug tracked by now-closed #3946. Keep this as the second Now-phase conformance item until maintainers confirm the task-call side is fixed by live harness evidence.
3. **CI-verify the first-agent on-ramp** ([#3955](https://github.com/micro/go-micro/issues/3955)) — the North Star, README, website, and roadmap now all make developer adoption the current goal: install → scaffold → run → chat → inspect must be a maintained contract. Add a no-secret, CI-verifiable first-agent harness that walks the documented examples and fails when docs drift from runnable commands, so the queue does not collapse entirely into provider hardening.
4. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — once the live Now-phase conformance gates are green, streaming is the highest developer-visible Next-phase seam. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
5. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
