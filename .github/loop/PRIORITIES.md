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

1. **Accept the buyer alias in universe concierge notifications** ([#3693](https://github.com/micro/go-micro/issues/3693)) — the first-agent/examples map shipped, so the top open item returns to a live Now-phase conformance failure. This small harness fix keeps the support-style services → agent → notification workflow from failing because a provider uses the stable buyer alias (`buyer`) instead of the full test email, while preserving recipient rejection and dedupe semantics.
2. **Expose `fallback_echo` during A2A streaming fallback conformance** ([#3560](https://github.com/micro/go-micro/issues/3560)) — this remains the next scoped Now-phase interop/conformance gap: it protects the A2A streaming promise developers see in the README and site by ensuring the non-native streaming fallback path receives the tool surface, while staying behind the immediate live-provider side-effect blocker.
3. **Parse multi-event A2A SSE fallback responses in the harness** ([#3662](https://github.com/micro/go-micro/issues/3662)) — once the fallback tool path succeeds, the harness must accept legitimate multi-event `message/stream` responses instead of concatenating valid SSE events into invalid JSON. This is a small CI-verifiable harness fix that keeps cross-provider streaming conformance focused on real gateway failures rather than parser brittleness.
4. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — the highest-value remaining Now-phase resilience gap after the live-provider side-effect fixes is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries and live-provider deadline tuning are in place; the lifecycle still needs cancellation/deadline propagation so work fails safely instead of becoming opaque loops.
5. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, hardened provider conformance, and fixed provider-emitted text tool calls. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures.
6. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, or observability. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
