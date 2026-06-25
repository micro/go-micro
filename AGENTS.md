# Repository agent instructions

These instructions apply to the entire repository.

## Pull requests from Codex tasks

When a Codex task makes repository changes and the requested outcome is a PR:

1. Keep the change focused on the assigned issue or prompt.
2. Run the relevant verification commands and capture their results.
3. Check `git status --short` and review the diff before finishing.
4. Stage the intended files and create a local git commit on the current branch.
5. Use the Codex `make_pr` tool to open the pull request with a concise title and a body that summarizes the change and testing.

Do not just say that a PR was opened. If local changes exist, the task is not complete until the changes are committed and the `make_pr` tool has been called. A GitHub token in the shell environment is not a substitute for the Codex `make_pr` tool in this environment.
