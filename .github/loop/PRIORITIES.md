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

1. **Stabilize AtlasCloud agent conformance guarded delegate check** ([#4007](https://github.com/micro/go-micro/issues/4007)) — #4009 closed the prior top streaming gap (#3903), and there are no open `codex` PRs currently carrying builder work. The remaining open Now-phase item is the live-provider conformance seam: AtlasCloud/minimax must reliably exercise the same guarded delegate refusal as the other provider paths without weakening the assertion. This protects the green-CI evaluator that keeps the loop trustworthy and prevents provider-specific behavior from drifting away from the services → agents → workflows contract.
2. **Trace agent runs as OpenTelemetry spans** ([#3908](https://github.com/micro/go-micro/issues/3908)) — the README, website, and recent blog story now align around the operational harness: services become tools, agents are services, flows handle the deterministic paths, and developers should be able to scaffold → run → chat → inspect → deploy. With first-agent wayfinding, no-secret harnesses, and streaming now represented, the next highest-value developer-visible seam is observability: `RunInfo` steps, tool calls, delegation, status, duration, and failures should correlate to spans while defaulting to no-op when tracing is not configured.
_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
