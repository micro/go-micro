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

1. **Make universe checkout conformance send exactly one concierge notification** ([#3633](https://github.com/micro/go-micro/issues/3633)) — durable checkout/universe is the strongest 0→hero proof for services → agents → workflows, and resume idempotency must be boring. Keep this first because duplicate buyer notifications are user-visible side effects in the adoption story and the plan/delegate duplicate-side-effect issue is now closed by PR #3664.
2. **Make the universe A2A reachability check deterministic** ([#3653](https://github.com/micro/go-micro/issues/3653)) — the latest live-provider scan shows the final universe A2A smoke can trigger extra concierge notifications and time out even after checkout durability assertions pass. Rank it with the universe side-effect work because gateway reachability should prove interop without adding more side effects or making CI flaky.
3. **Expose `fallback_echo` during A2A streaming fallback conformance** ([#3560](https://github.com/micro/go-micro/issues/3560)) — this remains the next scoped Now-phase interop/conformance gap: it protects the A2A streaming promise developers see in the README and site by ensuring the non-native streaming fallback path receives the tool surface, without letting protocol depth outrank the on-ramp or side-effect safety.
4. **Parse multi-event A2A SSE fallback responses in the harness** ([#3662](https://github.com/micro/go-micro/issues/3662)) — once the fallback tool path succeeds, the harness must accept legitimate multi-event `message/stream` responses instead of concatenating valid SSE events into invalid JSON. This is a small CI-verifiable harness fix that keeps cross-provider streaming conformance focused on real gateway failures rather than parser brittleness.
5. **Propagate agent run cancellation and deadlines through model and tool calls** ([#3544](https://github.com/micro/go-micro/issues/3544)) — the highest-value remaining Now-phase resilience gap after the live-provider side-effect fixes is predictable failure semantics across agent runs, model calls, tool calls, plan/delegate, and flow handoffs. Tool retries and live-provider deadline tuning are in place; the lifecycle still needs cancellation/deadline propagation so work fails safely instead of becoming opaque loops.
6. **Emit OpenTelemetry spans for agent run timelines** ([#3525](https://github.com/micro/go-micro/issues/3525)) — recent work made runs inspectable, correlated trace metadata through scheduled dispatch, verified restart resume, added opt-in tool retries, hardened provider conformance, and fixed provider-emitted text tool calls. The next Next-phase step is to turn that RunInfo foundation into standard OTel spans for agent runs, model calls, tool calls, checkpoint/resume, cancellation/deadlines, and failures.
7. **Add an AP2 mandate layer over A2A and x402** ([#3552](https://github.com/micro/go-micro/issues/3552)) — this is a forward interop investment, not a Now-phase blocker: Go Micro already has A2A agents and x402 paid tools, so a small signed-mandate foundation can keep agent payments aligned with the open-protocol story without pulling the queue away from adoption, resilience, or observability. Keep it additive and opt-in while the AP2/FIDO work settles.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
