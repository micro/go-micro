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

1. **Add CI-verified smallest first-agent transcript** ([#4639](https://github.com/micro/go-micro/issues/4639)) — PR #4637 closed #4634 by making first-agent chat wayfinding part of the docs guard, so the next developer-adoption gap is proving the fastest provider-free agent success path itself. The README, website getting-started path, examples index, zero-to-hero guide, and v6.3.15 blog now all point at `examples/first-agent` before the richer support-desk 0→hero app; make that smallest transcript and its next chat/inspect/debug breadcrumbs CI-verifiable so a new developer can get from scaffold/run to a real service-backed agent without keys before graduating to 0→hero.
2. **Harden atlascloud plan-delegate plan persistence** ([#4630](https://github.com/micro/go-micro/issues/4630)) — Keep this Now/Next hardening item close behind adoption work: recent AtlasCloud repair fixes reduced provider brittleness, but plan/delegate durability still has a provider-specific persistence seam. Scope it to the existing plan/delegate harness and CI-verifiable behavior so services → agents → workflows remains reliable without broad public-API or architecture changes.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
