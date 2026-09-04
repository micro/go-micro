# Codex Maintainer Playbook

Go Micro has 6 months OpenAI Codex access. Use it to boost maintainer throughput without lowering bar for review, tests, or design.

## Operating principles

1. **Humans set direction; Codex accelerates execution.** Maintainers choose issue/constraints/criteria. Codex drafts/investigates/verifies.
2. **Small, reviewable changes win.** Prefer focused PRs over large speculative rewrites.
3. **Keep the contract green.** Preserve CLI getting-started, harnesses, `make test`, and `make lint`.
4. **Document while coding.** Update examples, guides, release notes in the same branch if behavior changes.
5. **No blind merges.** Review Codex output like any contributor; back by tests, check API impact.

## Coordination with Claude Code

Go Micro is maintained by Codex and Claude Code (see [CLAUDE.md](CLAUDE.md)) plus human maintainer.

- **Lanes / branches.** Codex on `codex/*`, Claude on `claude/*`. Never share branches.
- **Base PRs on `master`.** Don't stack PRs on other agent branches. Wait or branch off `master`. Push to existing PR branch rather than opening stacked PRs.
- **One concern per PR.** Single-purpose PRs only.
- **Cross-review before merge.** Claude reviews your PRs; you review its with `@codex review`.
- **Dispatch.** Triggered via `@codex <instruction>` on issue/PR (`@codex review` is review). One task at a time to a clean, green PR.
- **CI is the gate.** `go build`, `go test`, `golangci-lint`, `make harness` must pass. `internal/harness/` and `examples/` excluded from errcheck.
- **Backlog = GitHub issues** with acceptance criteria.

## Best uses

### 1. PR review and triage

- Summarize PR surface area, API impact, test status.
- Ask for targeted reviews (concurrency, cancellation, security, compat, docs).
- Convert findings into patches or comments.

### 2. Issue reproduction

- Turn bugs into failing tests or repro scripts.
- Isolate fakes for dependencies.
- Attach exact repro command.

### 3. Release support

- Draft changelogs from commits.
- Verify `README.md`, `ROADMAP.md`, docs, examples, `CHANGELOG.md` agree.
- Run dry-release commands.

### 4. Docs and examples

- Keep 0→1 path current.
- Keep 0→hero example current (multi-agent, services, flows, MCP, A2A, observability).
- Add runnable examples before prose.

### 5. Hardening backlog

Break roadmap items into small PRs:
- cross-provider conformance
- timeout, cancellation, retry, rate-limit
- durable loops
- streaming across `ai.Stream`
- OpenTelemetry spans

## Suggested weekly loop

1. Pick maintenance lane.
2. Get branch plan with criteria/commands.
3. Implement smallest slice.
4. Run checks locally/CI.
5. Review and merge/send back.
6. Log recurring patterns.

## First two weeks

Start with roadmap maintenance.

### Day 1: set up the review loop

1. Pick 3 recent PRs (feature, bug, docs).
2. Review using template.
3. Compare with maintainer judgment.
4. Save review prompt.

### Days 2-3: make bugs reproducible

1. Pick open bug/flake.
2. Get failing test only (no fix).
3. Verify contract.
4. Fix in second branch with smallest patch.

### Days 4-5: audit the getting-started contract

Run 0→1 path; patch first break.

```sh
make test
make harness
make lint
go run ./examples/hello-world
go run ./internal/harness/universe
```

## Week 2: choose one roadmap slice

Pick hardening item. Recommended order:
1. Provider conformance skeleton.
2. Cancellation audit: trace `context.Context` propagation.
3. Docs drift audit: compare `README.md`, `ROADMAP.md`, website docs, and examples.
4. Release checklist dry run.

## Standing task queue

| Priority | Task | Acceptance criteria |
| --- | --- | --- |
| P0 | PR first-pass review | Summary, risks, required changes, verification commands. |
| P0 | Bug reproduction | Failing test committed before fix. |
| P0 | 0→1 docs check | Fresh-checkout commands work or patch fixes break. |
| P1 | Cross-provider conformance | Runs against fakes by default, real when keys exist. |
| P1 | Cancellation hardening | Tests prove timeout/cancel behavior. |
| P1 | Release audit | Changelog, docs, examples agree before tagging. |
| P2 | Example polish | Runnable, linked, checked. |

## What not to use Codex for yet

- Broad rewrites without test/benchmark.
- Unvetted public API changes.
- Large unrun docs.
- Provider-specific behavior not checked against `ai.Model`.

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