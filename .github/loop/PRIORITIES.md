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

1. **Polish resume and inspect breadcrumbs for agent runs** ([#4569](https://github.com/micro/go-micro/issues/4569)) — Durable runs, streaming, and human-input pauses are now part of the core story, but the developer inner loop still has to make the next command obvious when an agent pauses or needs inspection. Keep this first to match the current developer-adoption goal: chat → inspect → resume should feel like one walkable workflow, not an internal operations exercise.
2. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — With the universe notify timeout gap closed, the remaining live AtlasCloud seam is the A2A reachability probe intermittently timing out. Keep this next so agent reachability over the interop gateway does not produce false negatives once the durable services → agents → workflows path itself has finalized.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
