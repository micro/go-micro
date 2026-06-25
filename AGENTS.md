# Repository agent instructions

These instructions apply to the entire repository.

## Pull requests from Codex tasks

When a Codex task makes repository changes and the requested outcome is a PR:

1. Keep the change focused on the assigned issue or prompt.
2. Run the relevant verification commands and capture their results
   (`go build ./...`, `go test ./...`, `golangci-lint run ./...`).
3. Check `git status --short` and review the diff before finishing.
4. Stage the intended files and create a local git commit on the current branch
   (not `master`).
5. Open the pull request yourself with the GitHub CLI, which is installed in the
   environment and whose `origin` points at this repository:

   ```sh
   git push -u origin HEAD
   gh pr create --base master \
     --title "<concise title>" \
     --body "<summary of the change and testing, including 'Closes #<issue>'>"
   ```

Do not just say that a PR was opened, and do **not** rely on the `make_pr` tool:
in this environment `make_pr` only records the title/body and never pushes a
branch or creates a PR. The task is not complete until `gh pr create` has opened
a real pull request and printed its URL.
