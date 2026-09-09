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

1. **A2A external-client conformance** ([#4815](https://github.com/micro/go-micro/issues/4815)) — make the gateway easier for non-go-micro agents to discover and stream from by serving the well-known agent card path and spec SSE events.
2. **AP2 mandate foundation for agent payments** ([#4841](https://github.com/micro/go-micro/issues/4841)) — add opt-in checkout/payment mandate signing and verification so A2A-carried payment authority can settle over x402 without changing defaults.
3. **Kubernetes CRD reconciler foundation** ([#4842](https://github.com/micro/go-micro/issues/4842)) — turn the shipped alpha `Agent`, `Service`, and `Flow` CRDs into a minimally runnable native deployment path with workload reconciliation and status conditions.

### In flight — do not re-queue

_None right now._

### Background — hardening & DX (roadmap: Ongoing; capped)

_Background hardening is intentionally empty right now. Recent work covered first-agent
wayfinding, plan/delegate recovery, provider fallback repair, streaming, memory
compaction, retry controls, provider-failure inspection, x402 buyer safety, gRPC-reflection MCP,
MCP result conformance, and the alpha Kubernetes CRD surface. Further churn in those
areas should be marked `needs-human` unless it unlocks a clear user-visible capability._
