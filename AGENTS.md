# Repository agent instructions

These instructions apply to the entire repository.

## Pull requests from Codex tasks

When a Codex task makes repository changes and the requested outcome is a PR:

1. Keep the change focused on the assigned issue or prompt.
2. Run the relevant verification commands and capture their results
   (`go build ./...`, `go test ./...`, `golangci-lint run ./...`).
3. Check `git status --short` and review the diff before finishing.
4. Create a uniquely-named branch under the `codex/` prefix (do not work on
   `master`, and do not use a generic name like `work`):

   ```sh
   git switch -c codex/<issue-number>-<short-slug>
   ```
5. Stage the intended files and commit on that branch.
6. Open the pull request yourself with the GitHub CLI, which is installed in the
   environment and whose `origin` points at this repository, then enable
   auto-merge so GitHub merges it once the required CI checks pass:

   ```sh
   git push -u origin HEAD
   gh pr create --base master --label codex \
     --title "<concise title>" \
     --body "<summary of the change and testing, including 'Closes #<issue>'>"
   gh pr merge --squash --auto --delete-branch
   ```

The branch should start with `codex/` and the PR should carry the `codex`
label. Auto-merge waits for the required status checks (build, tests,
golangci-lint) — never merge a PR manually before CI is green.

Do not just say that a PR was opened, and do **not** rely on the `make_pr` tool:
in this environment `make_pr` only records the title/body and never pushes a
branch or creates a PR. The task is not complete until `gh pr create` has opened
a real pull request and printed its URL.
