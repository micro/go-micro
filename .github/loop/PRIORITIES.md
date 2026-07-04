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

1. **Mark AtlasCloud delegated plan steps complete** ([#3946](https://github.com/micro/go-micro/issues/3946)) — the latest live Harness (E2E) run got farther than the earlier task-call blocker: AtlasCloud/minimax-m3 created all three tasks and sent the delegated notification (`tasks=3 notify=1`), but the persisted conductor plan still left the delegation step unfinished. This is the top Now-phase item because it is the active red master conformance gate and directly tests whether services → agents → workflows state stays coherent after real delegation side effects.
2. **Stabilize AtlasCloud guarded delegate conformance** ([#3917](https://github.com/micro/go-micro/issues/3917)) — #3948 just merged stronger AtlasCloud delegate prompting and tagged fallback coverage, so this may be resolved by the next live run, but the issue is still open and has recurred multiple times. Keep it high until master's provider harness proves the guarded delegate reaches the approval-refusal path deterministically, then drop it immediately rather than re-queueing done conformance work.
3. **Execute AtlasCloud plan-delegate task tool calls** ([#3935](https://github.com/micro/go-micro/issues/3935)) — #3942 appears to have moved the live plan-delegate path past the tagged task-call failure, but the issue remains open. Leave it below the newer #3946 completion-state failure and close/drop it once the maintainers confirm the task-call side is fixed by the same live harness evidence.
4. **Broaden provider streaming and keep chat/A2A streaming end to end** ([#3903](https://github.com/micro/go-micro/issues/3903)) — once the live Now-phase conformance gates are green, streaming is the highest developer-visible Next-phase seam. Real chat and long-running A2A tasks need token streaming to stay coherent from provider → `ai.Stream` → `micro chat` → A2A `message/stream`, with mock/default CI coverage plus key-gated live provider checks and safe fallback for non-streaming providers.
5. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the blog/README/roadmap story promises an operable harness, and the developer on-ramp now includes chat, inspect, and run-history checkpoints. The next observability gap is production-grade trace correlation for `RunInfo`: steps, tool calls, delegation, status, durations, and failures should be visible as spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
