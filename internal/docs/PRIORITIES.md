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

1. **Agent: streaming Ask with tool-execution events** ([#3341](https://github.com/micro/go-micro/issues/3341)) — the Next-phase Streaming roadmap item, made concrete for the agent loop. `Ask` runs tools but returns the whole answer; `Stream` streams tokens but runs no tools — so no run can both orchestrate tools and stream its final answer. Add an additive, tool-aware streaming entry point that emits ToolStart/ToolEnd events as tools run and streams the final turn's tokens. Unblocks consumers replacing bespoke plan/execute/synthesize pipelines with the go-micro agent on streaming endpoints.
2. **Add memory compaction and retrieval for long-running agents** ([#3321](https://github.com/micro/go-micro/issues/3321)) — with durable resume, broad streaming, CLI run inspection, and RunInfo trace correlation now shipped, tackle the leading Later-roadmap gap: bounded, store-backed memory that keeps long agent runs useful without turning Go Micro into a prompt-layer framework.
3. **Add human-in-the-loop pause and resume for agent workflows** ([#3329](https://github.com/micro/go-micro/issues/3329)) — after memory is bounded, close the next operability gap for scheduled/looping work: durable pending-input states that let humans approve or provide context and then resume the same service/agent/workflow runtime.
4. **gateway/a2a: multiple typed skills per agent card** ([#3342](https://github.com/micro/go-micro/issues/3342)) — the a2a gateway now has the full task lifecycle (send/stream/get/cancel/resubscribe, push config, multi-turn, input-required); the remaining gap is skill granularity. `Card` only ever advertises one synthetic "chat" skill with services flattened into tags. Let an agent advertise N typed skills and route per skill, so domain-routing agents expose their real capabilities over A2A through the gateway instead of a hand-rolled handler.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
