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

1. **x402 buyer safety hard stop** ([#4814](https://github.com/micro/go-micro/issues/4814)) — close the budget-cap bypass and require a real settler for paid configs before autonomous paid-tool use becomes the next thing users copy into real agents.
2. **Kubernetes operator + CRDs foundation** ([#4797](https://github.com/micro/go-micro/issues/4797)) — add the first opt-in `Agent`, `Service`, and `Flow` resource foundation so the services → agents → workflows lifecycle has a native deployment path for Kubernetes users.
3. **A2A external-client conformance** ([#4815](https://github.com/micro/go-micro/issues/4815)) — make the gateway easier for non-go-micro agents to discover and stream from by serving the well-known agent card path and spec SSE events.
4. **MCP stdio/ws result conformance** ([#4813](https://github.com/micro/go-micro/issues/4813)) — return JSON tool results with explicit `isError` semantics across transports, backed by a stdio round-trip test.

### In flight — do not re-queue

- **gRPC-reflection MCP lint follow-up** ([#4824](https://github.com/micro/go-micro/issues/4824), [PR #4826](https://github.com/micro/go-micro/pull/4826)) — fixes the lint fallout from the merged reflected-gRPC MCP work.

### Background — hardening & DX (roadmap: Ongoing; capped)

_Background hardening is intentionally empty right now. Recent work covered first-agent
wayfinding, plan/delegate recovery, provider fallback repair, streaming, memory
compaction, retry controls, and provider-failure inspection. Further churn in those
areas should be marked `needs-human` unless it unlocks a clear user-visible capability._
