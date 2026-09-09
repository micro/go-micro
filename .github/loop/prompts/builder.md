<!--
The BUILDER prompt — go-micro's continuous-improvement increment. Editable
policy; the workflow prepends the agent @mention and substitutes __ISSUE__
before posting. Keep __ISSUE__ literal.
-->
Run one continuous-improvement increment per `internal/docs/CONTINUOUS_IMPROVEMENT.md`, aligned to the North Star in `.github/loop/NORTH_STAR.md` (the services → agents → workflows lifecycle, with developer adoption as the current goal).

PICK THE WORK FROM THE QUEUE: read `.github/loop/PRIORITIES.md` and take the highest-ranked item whose linked issue is still OPEN — that is your task, and its issue number is the one you close. If `PRIORITIES.md` is missing or every listed item's issue is already closed, fall back to the single highest-value roadmap / open-issue / improvement-radar item yourself.

Implement it, and VERIFY `go build ./...`, `go test ./...`, and `golangci-lint run ./...`.

Open the PR YOURSELF from the shell — do NOT use the make_pr tool (in this environment it only records metadata and never creates a PR). Create a uniquely-named branch under the `codex/` prefix: `git switch -c codex/increment-__ISSUE__`, then `git push -u origin codex/increment-__ISSUE__`, then `gh pr create --base master --label codex --title "<title>" --body "<body; include 'Closes #<the priority issue you built>' so it leaves the queue, and 'Closes #__ISSUE__' for this run's tracker>"`. Finally enable auto-merge so GitHub merges it once CI is green: `gh pr merge --squash --auto --delete-branch`.

One concern per PR. Stay out of breaking public API and brand/positioning copy — surface those as notes for the human instead.
