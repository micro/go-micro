---
layout: default
---

# `micro loop` quickstart

`micro loop` scaffolds the autonomous improvement loop that Go Micro uses on
this repository: GitHub Actions workflows for planning, building, evaluation
feedback, coherence, security, and release. Use it when you want a repository to
continuously turn a ranked queue into small PRs while CI remains the merge gate.

## 1. Initialize the loop

Run the default loop from the repository root:

```bash
micro loop init
```

For every role used by Go Micro itself, scaffold all workflows:

```bash
micro loop init --roles all
```

The command writes:

- `.github/loop/NORTH_STAR.md` — the direction every increment should optimize.
- `.github/loop/PRIORITIES.md` — the ranked queue; the builder takes the top open issue.
- `.github/loop/prompts/*.md` — editable policy for planner, builder, triage, coherence, and security roles.
- `.github/workflows/loop-*.yml` — generated GitHub Actions mechanics.

Edit the files under `.github/loop/` to steer the loop. Re-run
`micro loop init --roles all --force` only when you want to regenerate workflow
mechanics from the installed CLI.

## 2. Configure the dispatch token

The scheduled builder needs a repository secret containing a token from a user
account that the coding agent will answer. Go Micro names that secret
`CODEX_TRIGGER_TOKEN` by default. If you use another secret name, pass it when
you initialize the loop:

```bash
micro loop init --agent @codex --token-secret LOOP_TOKEN --roles all
```

The token needs enough repository permission to open issues, comment, push
branches, create pull requests, and enable auto-merge. Run `gh auth setup-git` in
the environment that will push branches so `git push` uses the same credentials
as `gh`.

## 3. Make CI the gate

The loop should not be its own reviewer. Protect the default branch so PRs merge
only after the required checks pass. At minimum, require the same commands the
Go Micro loop verifies locally and in CI:

```bash
go build ./...
go test ./...
golangci-lint run ./...
```

If your repository has a harness or end-to-end grader, make that required too.
Keep human approval requirements out of the autonomous path unless you intend the
loop to pause for review.

## 4. Verify the wiring

After editing the North Star, queue, prompts, token secret, and branch
protection, run:

```bash
micro loop verify
```

`micro loop verify` checks that the loop direction, queue, prompts, role
workflows, and non-loop CI gate are present. Fix any reported missing items
before relying on scheduled increments.

## 5. Operate the queue

Keep one ranked list in `.github/loop/PRIORITIES.md`. Each item should link a
scoped issue and be small enough for one PR. The builder closes both the priority
issue and the per-run tracker issue in the PR body, for example:

```text
Closes #1234
Closes #5678
```

Use the North Star to keep the queue honest: favor small improvements that move
developers through the services → agents → workflows lifecycle, and surface
breaking API or brand/positioning decisions for humans instead of auto-merging
them.
