<!--
The SECURITY prompt — go-micro's security audit. Editable policy; the workflow
prepends the agent @mention and substitutes __ISSUE__ before posting. Keep
__ISSUE__ literal.

Deliberately conservative: it does NOT auto-merge fixes, and it does NOT publish
exploit details in public issues (responsible disclosure).
-->
Act as the security reviewer for go-micro. Audit for real, exploitable vulnerabilities — skip theoretical or lint-style noise.

GO-MICRO ATTACK SURFACE — weight these:
- **MCP gateway** (`gateway/mcp`) and **A2A gateway** (`gateway/a2a`) — untrusted input from agents/tools: auth/scope enforcement, injection into downstream RPC, SSRF via tool/agent URLs, rate-limit/circuit-breaker bypass, info leak in errors.
- **x402 payments** (`wrapper/x402`) — payment verification and settlement: signature/mandate validation, replay, budget-reservation races, facilitator auth (CDP bearer) handling, amount/network confusion.
- **Auth** (`auth/jwt`, `wrapper/auth`) — token validation, algorithm confusion, scope/priority rule bypass, missing checks on endpoints.
- **AI providers** (`ai/*`) — base-URL and endpoint handling: SSRF via config-controlled `BaseURL`, API keys leaking into logs/errors, TLS verification.
- **Agent tool loop** (`agent/`) — prompt injection reaching real tool calls, guardrail (`MaxSteps`/`LoopLimit`/`ApproveTool`) bypass, delegate/plan side effects.
- **Trust boundaries** — `server` RPC handlers, `broker` consumers, `store`/`registry` inputs, `transport` TLS defaults (v6 verifies by default — confirm nothing regressed).
- **The loop itself** — `.github/workflows/loop-*.yml`: the `CODEX_TRIGGER_TOKEN` PAT must never be echoed/leaked; workflow inputs must not enable script injection.
- **Dependencies** — run `govulncheck ./...` (install if needed) and inspect `go.mod` for known CVEs.

DEDUPE against open issues first.

HOW TO REPORT:
- **Known/public dependency CVEs**: file a `security` issue referencing the CVE + module; you MAY open a PR bumping to the patched version. Do NOT enable auto-merge.
- **Novel, exploitable vulnerabilities in this code** (not yet public): do NOT post an exploit or PoC in a public issue. File a CONCISE `security` + `needs-human` issue naming the class, location (file/function), and impact only — and note it should go through GitHub private vulnerability reporting. Do NOT open a public fix PR that reveals it.
- **Low-risk hardening**: a normal `security` issue is fine.

NEVER auto-merge a security change. Never weaken a control to make a test pass. Architectural/breaking fixes → `needs-human` with the tradeoff.

Post a summary as a comment on this issue (#__ISSUE__) — findings by severity, what you filed, what needs a human — then close it (`gh issue close __ISSUE__`). If you open a dependency-bump PR: `git switch -c loop/security-__ISSUE__`, `git push -u origin loop/security-__ISSUE__`, `gh pr create --base master --label codex --label security --title "<title>" --body "<summary, Closes #__ISSUE__>"` — then STOP, do NOT run `gh pr merge --auto`. Do not use the make_pr tool.
