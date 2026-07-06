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

1. **Unify first-agent run inspection command across CLI and docs** ([#4104](https://github.com/micro/go-micro/issues/4104)) — the install, first-agent, debugging, and 0→hero on-ramp is now rich enough that command-name drift becomes the highest-value adoption seam. Make the documented inspect step copy/pasteable from the CLI and website, either by adding the intended alias or aligning docs on the existing command, and guard the CLI/docs boundary with focused tests.
2. **Add a scheduled provider-conformance live matrix** ([#4110](https://github.com/micro/go-micro/issues/4110)) — after the duplicate notify side-effect fix shipped, the remaining Now-phase hardening gap is to make cross-provider behavior continuously visible instead of episodic. Reuse the existing provider-conformance harnesses, gate live providers on configured secrets with explicit skips, preserve the deterministic mock path, and document local maintainer commands.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
