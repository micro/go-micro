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

1. **Execute provider-emitted text tool calls through the normal agent tool path** ([#3546](https://github.com/micro/go-micro/issues/3546)) — the live AtlasCloud/MiniMax conformance run exposed a current Now-phase interop seam: a provider can produce the right tool call as text JSON instead of structured `tool_calls`, leaving the agent to return raw JSON rather than operate the service. Fixing this first protects the core promise that typed Go Micro services are reliable agent tools across providers, while also hardening the A2A SSE fallback reader caught by the same harness.
2. **Raise live-provider harness call deadlines so slow correct models do not false-fail conformance** ([#3547](https://github.com/micro/go-micro/issues/3547)) — conformance is only useful if red means broken. The plan/delegate live run now fails on an internal RPC timeout while the model is still making correct progress, so the harness needs live-run-aware per-call deadlines before the queue can trust hourly cross-provider results as an architectural signal.
3. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — after the conformance harness is trustworthy, the highest-value remaining Now-phase resilience gap is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries are in place, but the lifecycle still needs cancellation/deadline propagation so services → agents → workflows fail safely instead of becoming opaque loops.
4. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, and hardened provider conformance. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures. This keeps `micro runs` useful while making the harness observable in the systems developers already run.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
