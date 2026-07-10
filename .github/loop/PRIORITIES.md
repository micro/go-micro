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

1. **Add provider-gated agent conformance matrix** ([#4568](https://github.com/micro/go-micro/issues/4568)) — The README, website, and recent blog story now promise one harness where services, agents, workflows, MCP/A2A, guardrails, memory, and providers behave as one runtime. The roadmap still names cross-provider conformance as the top Now hardening gap, and developer adoption depends on trust that the first real provider behaves like the no-secret mock path. Keep this first because it turns the lived agent-harness claim into a repeatable, CI/scheduled contract without adding install friction for local users.
2. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The universe checkout flow completes, but the A2A reachability probe can time out under AtlasCloud. This remains a direct cross-framework operability seam in the services → agents → workflows lifecycle and is the highest-value open interop gap after provider conformance.
3. **Polish resume and inspect breadcrumbs for agent runs** ([#4569](https://github.com/micro/go-micro/issues/4569)) — Durable runs, streaming, and human-input pauses are now part of the core story, but the developer inner loop still has to make the next command obvious when an agent pauses or needs inspection. This ranks after the active Now hardening items, while keeping adoption pressure in the queue: chat → inspect → resume should feel like one walkable workflow, not an internal operations exercise.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
