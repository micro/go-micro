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

1. **Verify 0-to-hero lifecycle transcript in CI** ([#4711](https://github.com/micro/go-micro/issues/4711)) — #4707 closed the first-agent wayfinding contract, so the highest-value Now-roadmap adoption gap moves to the full 0→hero services → agents → workflows lifecycle. The README, website, and v6.6.0 story now promise scaffold → run → chat → inspect → flow history → deploy dry-run as one walkable path; make that transcript a no-secret CI contract so the on-ramp cannot drift while internals keep hardening.
2. **Add first-agent debugging golden transcript coverage** ([#4712](https://github.com/micro/go-micro/issues/4712)) — Developer adoption still depends on recovery, not just happy-path setup. The CLI now points users through `micro agent demo`, quickcheck/debug, examples, chat, and inspect; verify those outputs against the maintained first-agent example so a new user who stalls can recover without learning package internals. Keep this scoped to docs/CLI contract coverage, not command redesign.
3. **Gate mock provider plan-delegate resume scenarios** ([#4713](https://github.com/micro/go-micro/issues/4713)) — #4709 closed the live AtlasCloud nested tool-call rejection gap, but the underlying agent-loop risk remains valuable enough to keep near the top: completed plan steps, notifications, unsafe fallback parsing, and resume semantics must stay deterministic without provider keys. Capture the recent provider failures in mock-based regression coverage before broadening into live-provider conformance.
4. **Migrate store/postgres from pgx/v4 to pgx/v5** ([#4556](https://github.com/micro/go-micro/issues/4556)) — Security upkeep matters to the service-framework half of the harness, and #4556 is the remaining open enhancement/security item. Keep it below the adoption and agent-loop contracts because it is narrower than the current developer-adoption goal, but do not let reachable dependency risk drift indefinitely.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
