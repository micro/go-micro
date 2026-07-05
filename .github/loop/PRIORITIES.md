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

1. **Surface a no-secret first-agent demo command** ([#4036](https://github.com/micro/go-micro/issues/4036)) — The previous top adoption/operability item shipped in #4034 and closed #4031/#4033 by adding `micro agent doctor` for scaffold → run → chat → inspect recovery. The remaining open `codex` PR is #4022, a human-review blog/changelog draft, so it should not occupy the autonomous builder queue. With the README, website roadmap, and blog all telling the same story — agents are services on one runtime, and the first developer success path matters as much as hardening — the next highest-value gap is CLI discoverability before a user even knows which repository example or doc to open. A provider-free `micro` affordance that surfaces the maintained first-agent/support demo path keeps the on-ramp walkable from the installed binary, links the no-secret path to live-provider chat and inspect/debugging docs, and is CI-verifiable without API keys or broad API changes.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
