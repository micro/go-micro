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

1. **Make plan-delegate harness complete delegated notification steps** ([#4138](https://github.com/micro/go-micro/issues/4138)) — the duplicate-notification fix and first-agent tutorial harness have both shipped, but the live atlascloud plan-delegate matrix still exposed a higher-severity Now-phase contract break: a delegated notification can succeed or be reused while the persisted plan step remains incomplete. Fixing that keeps the provider conformance evaluator trustworthy without relaxing the services → agents → workflows lifecycle gate.
2. **Verify no-secret agent debugging walkthrough** ([#4142](https://github.com/micro/go-micro/issues/4142)) — keep adoption pressure balanced with hardening by extending the maintained 0→1 path past scaffold/run/chat into inspect and debugging. A provider-free smoke check for the documented first-agent debug sequence will catch command drift at the exact seam where new developers need confidence after the first conversation behaves unexpectedly.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
