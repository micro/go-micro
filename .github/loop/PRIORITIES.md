# Priorities

The ranked work queue for the autonomous improvement loop. The **planner** owns
this file: each run it turns the [roadmap](../../ROADMAP.md) plus an internal scan
into a single ordered list — highest-value first — each item linked to a tracking
issue. The **builder** works the top item whose issue is still open. So the
planner decides *what*, the builder *builds* it.

**Bias to capability, not busy-work.** The top of this queue is net-new capability
from the roadmap's *Now/Next* items. Hardening/conformance/DX polish is background
work (roadmap *Ongoing*) — kept low here and capped, never allowed to crowd out
capability. If an area has had several increments with no user-visible gain, it is done
for now; rank real-headroom capability instead.

**Reading / editing.** An item is done when its linked issue closes (the PR that
builds it adds `Closes #<issue>`). The human can reorder this list or the issues at
any time — direction always wins.

**Off-limits to the loop** (planner proposes as notes, never auto-merged queue
items): brand/positioning copy, breaking public-API changes, architectural
rewrites.

## Work queue (ranked)

### Capability — the headline (roadmap: Now / Next)

1. **Agents that pay — wire the x402 buyer into the agent runtime** ([#4786](https://github.com/micro/go-micro/issues/4786)) — the flagship. The buyer `x402.Client`/`Payer`/budget already exists; wire it so an agent autonomously settles a payment-required tool within budget and retries, opt-in and observable. Makes go-micro a runtime for autonomous agent commerce.
2. **Agent spend observability** ([#4787](https://github.com/micro/go-micro/issues/4787)) — surface x402 spend in `RunInfo` and OpenTelemetry so payments are inspectable like every other agent action. (Follows #4786.)
3. **Example: an agent that pays for a paid tool** ([#4788](https://github.com/micro/go-micro/issues/4788)) — the runnable artifact that makes it real for a developer, against a mock facilitator (no live funds). (Follows #4786/#4787.)
4. **gRPC-reflection MCP** ([#4796](https://github.com/micro/go-micro/issues/4796)) — expose external reflected gRPC services as MCP tools, not only go-micro-native handlers. A large jump in what agents can operate without requiring teams to rewrite existing services.
5. **Kubernetes operator + CRDs foundation** ([#4797](https://github.com/micro/go-micro/issues/4797)) — add the first opt-in `Agent`, `Service`, and `Flow` resource foundation so the services → agents → workflows lifecycle has a native deployment path for Kubernetes users.

### Background — hardening & DX (roadmap: Ongoing; capped)

_Background hardening is intentionally empty right now. Recent work covered first-agent
wayfinding, plan/delegate recovery, provider fallback repair, streaming, memory
compaction, retry controls, and provider-failure inspection. Further churn in those
areas should be marked `needs-human` unless it unlocks a clear user-visible capability._
