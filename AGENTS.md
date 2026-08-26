# Repository agent instructions

Repo-wide instructions.

## Pull requests from Codex tasks

For PR-focused Codex tasks:

1. Focus changes on issue/prompt.
2. Run verification (`go build ./...`, `go test ./...`, `golangci-lint run ./...`).
3. Check `git status --short` and diff.
4. Branch under `codex/` (not `master` or `work`):

   ```sh
   git switch -c codex/<issue-number>-<short-slug>
   ```
5. Stage and commit.
6. Open PR via GitHub CLI (`origin` set), enable auto-merge:

   ```sh
   git push -u origin HEAD
   gh pr create --base master --label codex \
     --title "<concise title>" \
     --body "<summary of the change and testing, including 'Closes #<issue>'>"
   gh pr merge --squash --auto --delete-branch
   ```

Branch `codex/`, PR label `codex`. Auto-merge waits for CI (build, tests, golangci-lint) — never merge manually before CI green.

Don't just claim PR opened. DO NOT use `make_pr` tool (`make_pr` only records title/body, never pushes/creates). Task incomplete until `gh pr create` opens real PR and prints URL.