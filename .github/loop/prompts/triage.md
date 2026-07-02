<!--
The TRIAGE prompt — go-micro's harness-failure feedback path. Editable policy;
the workflow prepends the agent @mention and substitutes __ISSUE__ (this
tracking issue) and __RUNURL__ (the failed run) before posting. Keep both literal.
-->
Triage the failed provider-conformance harness run at __RUNURL__.

Read the logs and root-cause each distinct failure. DEDUPE against open issues — if a failure matches an existing issue, comment "recurred" there instead of filing a duplicate.

For each genuine, self-contained defect, file a scoped issue (`gh issue create --label codex --label enhancement --title "<scoped fix>" --body "<root cause, where, acceptance criteria>"`) so the increment loop builds it and the next harness run verifies it.

IGNORE transient flakes — live-model latency, provider outages, rate limits, timeouts with no code cause. Anything needing a breaking or architectural change: file it as `needs-human` and describe it, rather than auto-queuing it as a routine fix.

Close this issue (`gh issue close __ISSUE__`) when triage is done. Open any PR yourself from the shell with `gh`; do not use the make_pr tool.
