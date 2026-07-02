<!--
The TRIAGE prompt — go-micro's CI-failure feedback path. Editable policy; the
workflow prepends the agent @mention and substitutes __ISSUE__ (this tracking
issue) and __RUNURL__ (the failed run) before posting. Keep both literal.
-->
Triage the failed CI run at __RUNURL__. It may be the linter (Lint), the unit/integration tests (Run Tests), or the provider-conformance harness (Harness (E2E)).

Read the logs and root-cause each distinct failure. DEDUPE against open issues — if a failure matches an existing issue, comment "recurred" there instead of filing a duplicate.

For each genuine, self-contained defect, file a scoped issue (`gh issue create --label codex --label enhancement --title "<scoped fix>" --body "<root cause, where, acceptance criteria>"`) so the increment loop builds it and the next CI/harness run verifies it. A lint or test failure on master is a real regression — file it so it is fixed promptly; do NOT ignore it.

IGNORE only genuine transient flakes — live-model latency, provider outages, rate limits, network timeouts with no code cause (mostly relevant to the harness). Anything needing a breaking or architectural change: file it as `needs-human` and describe it, rather than auto-queuing it as a routine fix.

Close this issue (`gh issue close __ISSUE__`) when triage is done. Open any PR yourself from the shell with `gh`; do not use the make_pr tool.
