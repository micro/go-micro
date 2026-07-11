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

1. **Harden atlascloud plan-delegate plan persistence** ([#4630](https://github.com/micro/go-micro/issues/4630)) — Recent adoption PRs closed the smallest first-agent transcript and next-step breadcrumb gap, leaving this Now/Next reliability seam as the highest-value open work. Keep it scoped to existing plan/delegate harness behavior so provider-specific persistence becomes durable without a public-API or architectural rewrite.
2. **Add CI-verified first-agent chat/inspect fixture** ([#4644](https://github.com/micro/go-micro/issues/4644)) — Developer adoption stays close behind hardening: the README, website, examples index, first-agent example, and no-secret guide now align around the provider-free path, but the remaining on-ramp seam is proving the real CLI-shaped chat → inspect loop for the smallest agent without keys. Add a focused harness fixture before expanding deeper 0→hero or provider work.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
