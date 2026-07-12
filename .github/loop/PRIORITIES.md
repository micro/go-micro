# Priorities

The ranked work queue for the autonomous improvement loop. The **planner** ranks
it; the **builder** works the top item whose linked issue is still open. Direction
comes from the human (the [roadmap](../../ROADMAP.md) and the
[gap audit](../../internal/docs/GAP_AUDIT.md)) — the planner does NOT invent work,
it ranks the curated backlog. The builder executes well-defined issues; it does
not improvise.

**Advances the strategy vs. grooms a proxy.** Rank by whether an item advances
the actual strategy — real integration/exposure, real capability — not by whether
it produces a green increment. Making MCP speak MCP to an external client, A2A
interoperate with a real external agent, and x402 actually settle is real work.
Guarding docs the loop wrote or chasing a weak provider's quirks is grooming —
`needs-human` it.

**Reading / editing.** An item is done when its linked issue closes (the PR adds
`Closes #<issue>`). The human reorders this list or the issues at any time.

**Off-limits to the loop** (never auto-merged): brand/positioning copy, breaking
public-API changes, architectural rewrites. Items labelled `needs-human` below are
1:1 development work — the loop must not auto-build them.

## Work queue (ranked)

### Strategic spine — make integration/exposure actually robust (gap audit)

These are the surfaces the strategy rests on, and they're the least externally-proven
part of the codebase. Highest value.

1. **MCP: stdio/ws tool results must be JSON + `isError`, with a stdio test** ([#4813](https://github.com/micro/go-micro/issues/4813)) — the path Claude Desktop uses currently returns Go `%v` map-syntax instead of JSON and misreports tool errors. Cheapest, highest-impact fix.
2. **x402: fix the budget-cap bypass + require a real Settler** ([#4814](https://github.com/micro/go-micro/issues/4814)) — a malformed amount defeats the spend cap; verify-only serves the resource for free. Hardens the safety the flagship relies on.
3. **A2A: conform to external clients — well-known path + spec SSE events** ([#4815](https://github.com/micro/go-micro/issues/4815)) — an external A2A SDK 404s on discovery and can't parse the stream. Real cross-framework interop.
4. **Agents that pay — wire the x402 buyer into the agent runtime** ([#4786](https://github.com/micro/go-micro/issues/4786)) — the flagship capability; the buyer `Client` exists but is wired into nothing (the agent's "spend budget" is bookkeeping that never pays).

### Capability — reach & deployment (roadmap: Next; builds on the spine)

5. **gRPC-reflection MCP** ([#4796](https://github.com/micro/go-micro/issues/4796)) — expose external reflected gRPC services as MCP tools.
6. **Kubernetes operator + CRDs foundation** ([#4797](https://github.com/micro/go-micro/issues/4797)) — `Agent`/`Service`/`Flow` as native K8s resources.

### Human-led — real 1:1 development (needs-human; the loop must NOT auto-build these)

- **MCP transport unification** ([gap audit](../../internal/docs/GAP_AUDIT.md) item 2) — mount the JSON-RPC handler as the HTTP transport and run all transports through one pre-call pipeline (auth→rate→breaker→payment). Architectural.
- **Durable agentic workflow: HITL pause + per-tool-call checkpointing** ([#4816](https://github.com/micro/go-micro/issues/4816)) — the convergence leg; core primitive design.
- **In-process dispatch fast-path** ([#4817](https://github.com/micro/go-micro/issues/4817)) — a local transport so in-process calls skip codec + network hop.

_Evidence base: [`internal/docs/GAP_AUDIT.md`](../../internal/docs/GAP_AUDIT.md) (this session's code audit) + the "requirements discovered from Mu" notes. Restocked by Claude Code; the planner ranks, it does not invent._
