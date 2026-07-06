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

1. **Add a scheduled provider-conformance live matrix** ([#4110](https://github.com/micro/go-micro/issues/4110)) — the top remaining Now-phase hardening gap is to make cross-provider behavior continuously visible instead of episodic. Reuse the existing provider-conformance harnesses, gate live providers on configured secrets with explicit skips, preserve the deterministic mock path, and document local maintainer commands.
2. **Add CLI examples wayfinding for first-agent paths** ([#4115](https://github.com/micro/go-micro/issues/4115)) — keep developer adoption weighted alongside hardening by making the maintained no-secret first-agent, debugging, and 0→hero examples discoverable from the CLI itself. Align CLI output, README, and website guide references so the scaffold → run → chat → inspect path stays copy/pasteable after install.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
