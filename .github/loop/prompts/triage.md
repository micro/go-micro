<!--
The TRIAGE prompt — go-micro's CI-failure feedback path. Editable policy; the
workflow prepends the agent @mention and substitutes __ISSUE__ (this tracking
issue) and __RUNURL__ (the failed run) before posting. Keep both literal.
-->
Triage the failed CI run at __RUNURL__. It may be the linter (Lint), the unit/integration tests (Run Tests), the vulnerability gate (govulncheck), or the provider-conformance harness (Harness (E2E)).

Read the logs and root-cause each distinct failure. DEDUPE hard against open AND recently-closed issues — if a failure matches an existing or recurring one, comment "recurred" on that issue rather than filing a new one.

WHAT TO FILE:
- **Lint, Run Tests, or govulncheck failing on master** — a real regression. File a scoped issue (`gh issue create --label codex --label enhancement --title "<scoped fix>" --body "<root cause, where, acceptance>"`) so it is fixed promptly.
- **A genuinely NEW, distinct provider-conformance defect** — file it.

WHAT NOT TO FILE (this cap matters):
- **Another instance of a class the agent already tolerates** — a weak provider (e.g. AtlasCloud) emitting malformed / text-rendered / partial tool calls, or another plan/delegate notify/side-effect edge case. These have been hardened repeatedly with diminishing returns. Do NOT auto-file yet another routine robustness patch. Comment "recurred — repeated class, capped" on the nearest existing issue and, if it seems genuinely worth more investment, label it `needs-human` for a human to decide. The loop should not keep chasing one weak provider's output shape.
- **Transient flakes** — live-model latency, provider outages, rate limits, network timeouts with no code cause. Ignore.
- **Anything needing a breaking or architectural change** — label `needs-human` and describe it.

Close this issue (`gh issue close __ISSUE__`) when triage is done. Open any PR yourself from the shell with `gh`; do not use the make_pr tool.
