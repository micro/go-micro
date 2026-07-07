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

1. **Resume durable agent runs from checkpoints** ([#4202](https://github.com/micro/go-micro/issues/4202)) — the installed first-agent on-ramp shipped in #4207, AtlasCloud guarded-delegate conformance closed in #4211, and retry side-effect dedupe closed in #4215. With the highest Now-phase provider-failure risk resolved, the next most valuable gap is the durable-agent-loop promise from the roadmap and blog: long-running agents should recover like flows, checkpoint progress, and avoid replaying completed tool side effects after interruption.
2. **Broaden provider streaming coverage through chat and A2A** ([#4217](https://github.com/micro/go-micro/issues/4217)) — after durable resume, improve the interactive inner loop: `ai.Stream`, `micro chat`, and A2A streaming should preserve token order, cancellation, and error semantics end to end so a developer can move from local chat to interop without learning separate transport behavior.
3. **Emit OpenTelemetry spans from agent run history** ([#4218](https://github.com/micro/go-micro/issues/4218)) — once durable runs and streaming are guarded, close the observability seam called out by the roadmap and blog: `RunInfo`/history and production traces should tell the same story for model steps, tool calls, retries, delegation, and failures.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
