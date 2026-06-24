# Codex Maintainer Playbook

Go Micro has six months of Codex access through OpenAI's Codex for Open Source
program. Use it to increase maintainer throughput without changing the project's
bar for review, tests, or design taste.

## Operating principles

1. **Humans set direction; Codex accelerates execution.** Maintainers choose the
   issue, constraints, and acceptance criteria. Codex drafts, investigates, and
   verifies.
2. **Small, reviewable changes win.** Prefer focused PRs that can be understood
   in one sitting over large speculative rewrites.
3. **Keep the contract green.** Every Codex-assisted change should preserve the
   CLI-first getting-started flow, the harnesses, `make test`, and `make lint`.
4. **Document while coding.** If behavior changes, ask Codex to update examples,
   guides, and release notes in the same branch.
5. **No blind merges.** Codex output is treated like any contributor output:
   reviewed by a maintainer, backed by tests, and checked for public API impact.

## Best uses

### 1. PR review and triage

- Summarize a PR: changed surface area, public API impact, tests added or missing.
- Ask for targeted review passes: concurrency, cancellation, security, backwards
  compatibility, docs drift, and examples.
- Convert review findings into small patch suggestions or issue comments.

### 2. Issue reproduction

- Turn bug reports into failing tests or runnable reproduction scripts.
- Minimize flakes by isolating registry, broker, store, transport, and AI-provider
  dependencies behind deterministic fakes where possible.
- Attach the exact command that reproduces the failure to the issue.

### 3. Release support

- Draft changelog entries from merged commits, grouped by feature, fix, docs, and
  compatibility notes.
- Check that `README.md`, `ROADMAP.md`, website docs, examples, and `CHANGELOG.md`
  agree before tagging.
- Run dry-run release commands and summarize blockers.

### 4. Docs and examples

- Keep the 0→1 path current: scaffold, run, call, chat, inspect.
- Keep the 0→hero example current: a realistic multi-agent system that exercises
  agents, services, flows, MCP, A2A, and observability.
- Add runnable examples for new primitives before adding broad prose.

### 5. Hardening backlog

Use Codex to break roadmap items into small PRs, especially:

- cross-provider conformance scenarios for all supported AI providers;
- timeout, cancellation, retry, and rate-limit behavior;
- durable agent loops on top of the existing checkpoint model;
- streaming across `ai.Stream` and A2A;
- agent run metadata mapped to OpenTelemetry spans.

## Suggested weekly loop

1. Pick one maintenance lane: reviews, bugs, release prep, docs, or hardening.
2. Ask Codex for a branch-sized plan with acceptance criteria and test commands.
3. Have Codex implement the smallest valuable slice.
4. Run the relevant checks locally and in CI.
5. Review the diff as maintainer-owned code, then merge or send it back.
6. Record any recurring prompt, check, or failure mode in this playbook.


## First two weeks

Do not start with a giant feature. Start by making Codex pay rent on maintenance
work that is already on the roadmap and easy to review.

### Day 1: set up the review loop

1. Pick three recent PRs or commits: one feature, one bug fix, and one docs-only
   change.
2. Ask Codex to review each using the PR review template below.
3. Compare Codex findings with maintainer judgment. Keep the checks that found
   real issues; delete the noisy ones.
4. Turn the final review prompt into a saved project note or issue comment
   template.

Success means Codex can produce a useful first-pass review in under ten minutes
without blocking a maintainer on false positives.

### Days 2-3: make bugs reproducible

1. Pick one open bug or flaky area.
2. Ask Codex for a failing test only. Do not allow a fix in the first pass.
3. Review the test for whether it captures the real contract.
4. In a second branch, ask Codex to fix the failure with the smallest patch.

Success means every accepted bug fix starts with a regression test or deterministic
harness case.

### Days 4-5: audit the getting-started contract

Run through the 0→1 path from a clean checkout and ask Codex to patch only the
first broken or confusing step. The target is not new prose; it is a runnable
path that works exactly as documented.

Candidate checks:

```sh
make test
make harness
make lint
go run ./examples/hello-world
go run ./internal/harness/universe
```

### Week 2: choose one roadmap slice

Pick one hardening item and break it into PRs that each land independently. The
best first slice is usually test infrastructure, not product code.

Recommended order:

1. **Provider conformance skeleton**: define one deterministic agent scenario and
   gate real-provider runs on credentials.
2. **Cancellation audit**: trace `context.Context` propagation through one package
   at a time.
3. **Docs drift audit**: compare `README.md`, `ROADMAP.md`, website docs, and
   examples for one shipped feature.
4. **Release checklist dry run**: have Codex build a release-blocker list from the
   diff since the previous tag.

## Standing task queue

Keep Codex busy on tasks with clear acceptance criteria:

| Priority | Task | Acceptance criteria |
| --- | --- | --- |
| P0 | PR first-pass review | Summary, risks, required changes, and exact verification commands. |
| P0 | Bug reproduction | A failing test or harness case committed before the fix. |
| P0 | 0→1 docs check | Fresh-checkout commands work as written or a patch fixes the first break. |
| P1 | Cross-provider conformance | One scenario runs against fakes by default and real providers when keys exist. |
| P1 | Cancellation hardening | Tests prove timeout/cancel behavior for the touched package. |
| P1 | Release audit | Changelog, docs, examples, and migration notes agree before tagging. |
| P2 | Example polish | Example is runnable, linked from docs, and covered by a lightweight check. |

## What not to use Codex for yet

- Broad rewrites without a failing test, benchmark, or public design note.
- Public API changes before a maintainer writes the compatibility story.
- Large generated docs that nobody has run.
- Provider-specific behavior that is not checked against the shared `ai.Model`
  contract.

## Prompt templates

### PR review

```text
Review this PR for Go Micro. Focus on public API compatibility, cancellation and
context propagation, concurrency safety, tests, and docs drift. Return: summary,
risks, required changes, optional improvements, and exact commands to verify.
```

### Bug reproduction

```text
Reproduce this issue in the smallest Go test or harness change possible. Do not
fix it yet. Explain the failing path and provide the exact command that fails.
```

### Branch implementation

```text
Implement the smallest branch that satisfies this issue. Keep the API compatible
unless explicitly required, update docs/examples when behavior changes, and run
`make test`, `make harness`, and `make lint` or explain any environment blocker.
```

### Release audit

```text
Audit this release branch. Compare CHANGELOG, README, ROADMAP, website docs, and
examples against the diff since the last tag. List inconsistencies, missing
migration notes, and checks to run before tagging.
```
