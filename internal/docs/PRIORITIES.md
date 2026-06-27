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

## Now (ranked)

1. **Durable agent loop** (#3010) — checkpoint and resume an agent run via
   `flow.Checkpoint` so long-running agents survive restarts without replaying
   completed tool calls. Roadmap → *Next*; promoted because the Now hardening
   items (0→hero and provider matrix) have shipped, and durability is the biggest
   remaining harness-operability gap.
2. **Streaming end to end** (#3012) — `ai.Stream` through `micro chat`, the agent
   RPC, and A2A `message/stream`; scope to chat + one provider first. Roadmap →
   *Next*.
3. **Registry disconnection detection** (#2956) — readiness/health when a service
   silently loses its registry connection. Community-requested production
   reliability; keeps the service substrate operable because agents depend on
   discovery.
4. **Execution lifecycle hooks & metadata** (#2980) — before/after-tool, retry,
   and failure hooks; first check overlap with the shipped run-timeline /
   OpenTelemetry work and scope to what's not already covered.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
