# Demo recordings

Terminal demos for the README and website, scripted with
[VHS](https://github.com/charmbracelet/vhs) so they are reproducible and never
rot: edit the `.tape`, re-record, commit the new GIF.

## Tapes

| Tape | What it shows | Needs a key? |
|------|---------------|--------------|
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
