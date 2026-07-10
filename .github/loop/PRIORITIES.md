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

1. **Prevent checkpointed tool calls from panicking** ([#4594](https://github.com/micro/go-micro/issues/4594)) — Highest-value Now-phase resilience gap: a live AtlasCloud plan/delegate run completed the intended side effects but then crashed inside checkpointed tool handling. The services → agents → workflows lifecycle cannot feel operable if malformed or repaired provider output can SIGSEGV the harness instead of returning a classified error, retrying, or failing safely.
2. **Thread first-agent quickcheck through docs wayfinding** ([#4599](https://github.com/micro/go-micro/issues/4599)) — Keep developer adoption weighted alongside hardening. The CLI quickcheck breadcrumbs shipped in #4597, but the README/website first-agent path should make the recovery command discoverable before a new user falls into the full docs tree. This preserves the scaffold → run → chat → inspect on-ramp as a contract, not tribal CLI knowledge.
3. **Handle incomplete AtlasCloud agent-flow workspace tool calls** ([#4595](https://github.com/micro/go-micro/issues/4595)) — Next resilience gap from the same live provider-conformance run: the event-driven agent-flow path achieved the expected workspace/notification side effects, then still surfaced a failed flow on an incomplete repaired text tool call. Fix after the panic guard so successful/suppressed side effects do not leave the workflow story looking failed.
4. **Make AtlasCloud universe A2A reachability probe deterministic** ([#4504](https://github.com/micro/go-micro/issues/4504)) — The remaining live AtlasCloud interop seam is the universe A2A reachability probe intermittently timing out. Keep it in the queue, but behind the crash/tool-call correctness issues and the adoption quickcheck follow-through, because false-negative reachability matters most once the core live flow exits cleanly.

_Seeded by Claude Code from the roadmap + open issues; thereafter maintained by the
architecture-review pass._
