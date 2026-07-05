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

1. **Align architecture docs with the agent harness lifecycle** ([#4092](https://github.com/micro/go-micro/issues/4092)) — the first-agent on-ramp, install troubleshooting, preflight, and after-run doctor seams are now covered, but the website architecture page still reads like a pre-agent distributed-systems overview. Refresh it so newcomers see one coherent services → agents → workflows runtime: registry/server/client as the service substrate, `model`/`store` as state, `ai`/`agent` as the tool-calling loop, `flow` as durable deterministic orchestration, and MCP/A2A gateways as interop. Add a focused docs/wayfinding assertion so the architecture story keeps pointing back to AI integration, first-agent, and 0→hero.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
