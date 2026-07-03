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

1. **Fix AtlasCloud minimax-m3 tool-call 400s in provider conformance** ([#3742](https://github.com/micro/go-micro/issues/3742)) — PR #3744 closed the empty follow-up/tool-result blocker (#3735), but the latest live provider run now fails earlier: AtlasCloud rejects tool-enabled `minimaxai/minimax-m3` requests with repeatable 400s across the same first-agent, universe, plan/delegate, agent-flow, and A2A fallback harnesses. This is the highest-value Now-phase adoption blocker because the cross-provider contract cannot prove services-as-tools work at all until the provider request shape is valid and diagnosable.
2. **Require AtlasCloud notification side effects in multi-step harnesses** ([#3736](https://github.com/micro/go-micro/issues/3736)) — after AtlasCloud tool-call requests stop failing with 400s, the multi-step universe and plan/delegate harnesses must not accept provider replies that claim completion while skipping the required notify/delegate side effect. This protects the services → agents → workflows lifecycle promised by the first-agent and 0→hero paths: an agent should operate the system, not only narrate success.
3. **Parse multi-event A2A SSE fallback responses in the harness** ([#3662](https://github.com/micro/go-micro/issues/3662)) — once the current AtlasCloud tool-call and side-effect failures are resolved, the harness must accept legitimate multi-event `message/stream` responses instead of concatenating valid SSE events into invalid JSON. This remains a small CI-verifiable harness fix that keeps cross-provider streaming conformance focused on real gateway failures rather than parser brittleness.
4. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — the highest-value remaining resilience gap is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries, live-provider deadline tuning, and delegated-plan completion are in place; the lifecycle still needs cancellation/deadline propagation so work fails safely instead of becoming opaque loops.
5. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, hardened provider conformance, and fixed provider-emitted text tool calls. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures.
6. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, or observability. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
