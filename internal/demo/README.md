# Demo recordings

Terminal demos for the README and website, scripted with
[VHS](https://github.com/charmbracelet/vhs) as recording recipes. They are not evidence of a passing journey: rehearse against
the current CLI, inspect the tool actions, and review each recording before publishing.

## Current chat experience

There is no committed recording of the current chat UI yet. `chat.tape` is a
live-provider recipe, not a simulated showcase. It must be recorded and reviewed
with a real provider before embedding a GIF in the README or website.

Build `micro` from this checkout, export `OPENAI_API_KEY`, and create a disposable
project containing `hello.go`:

```go
package main
import "fmt"
func main() { fmt.Println("Hello, world!") }
```

Copy `chat.tape` into that project, change its `Output` to `chat.gif`, and run
`vhs chat.tape`. The tape uses a light theme. It asks the agent to inspect the
file, edit it after approval, run it after approval, then reopens chat to show
saved history. Provider response timing and tool choices vary: rehearse manually
and adjust the tape to the actual requests. Do not publish a recording with
errors, extra approvals, or a claimed result that wasn't checked on disk.

The provider-free terminal check is `make chat-smoke`; it tests real file effects
with scripted model replies. It is deliberately not presented as a live agent demo.

## Earlier service demos

| Tape | What it shows | Needs a key? |
|------|---------------|--------------|
| `chat.tape` | Current chat: read → approve edit → approve verification → restart | Yes (`OPENAI_API_KEY`) |
| `first-run.tape` | Quick start: `micro new` → `micro run` → `curl` | No |
| `prompt-demo.tape` | Hero demo: `micro run --prompt` designs, builds, and starts services; mid-conversation service generation | Yes (`ANTHROPIC_API_KEY`) |

## Recording

Install VHS and its dependencies (ttyd, ffmpeg), and make sure the `micro`
CLI being demoed is on `PATH`:

```bash
go install github.com/charmbracelet/vhs@latest
make demo-gif                       # records first-run.gif (no key needed)
vhs internal/demo/prompt-demo.tape  # manual: live-provider timing, tune sleeps
```

Record in a scratch directory — the tapes scaffold real services where they run.

## Embedding

Once recorded, embed at the top of the README (below the Overview) and on the
website homepage:

```markdown
![micro run --prompt demo](internal/demo/prompt-demo.gif)
```

Keep GIFs under ~10MB so GitHub renders them inline. `first-run.gif` is the
fallback if the prompt demo is too heavy.
