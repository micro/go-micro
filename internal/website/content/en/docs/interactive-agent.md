---
title: "Interactive agent development"
description: "Build and extend an agent through Go Micro and micro chat."
---

`micro chat` exposes the framework's agent, model, memory, tools, and RPC
interfaces. Developers can replace or extend those components in Go. Mu can host
them with its own product services.

## Start working

```bash
export OPENAI_API_KEY=your-api-key
micro chat
```

With no registered agents, chat starts a local development agent. With one
registered agent, it connects directly without needing a local model key.
`micro chat NAME` selects a host when several agents are registered.

Use `/model` and `/provider` to change the local configuration without clearing
history. `/models` queries the live provider catalog; `/model ID` works without
catalog support. Settings and entered credentials are scoped to the project and
provider endpoint. Environment keys are not copied into settings. Remote agents
advertise their model but retain control of their configuration.

The optional `model.ModelLister` interface lets providers implement discovery.
Catalogs currently cover OpenAI, Anthropic, Gemini, Groq, Mistral, and Together.
Catalog IDs do not guarantee account access or suitability for tool use.

## Interaction and conversations

- `/paste` composes multiple lines until `/send`; `/cancel` discards them.
- Requests entered during work are queued. `/queue` shows the pending requests.
- `/steer MESSAGE` cancels the current request and runs the correction next.
- Ctrl-C or `/stop` cancels work and clears the queue. Completed tool actions
  are not rolled back.
- `/new`, `/sessions`, and `/resume ID` manage conversations.
- `/history` shows retained context; `/search TEXT` searches retained and archived
  messages in the current conversation; `/compact` keeps recent messages and
  summarizes older context.
- `/agents` shows connected agents. Local delegation displays the target and task.

Local chat uses `agent.CompactMemory(50, 20)` and explicit compaction keeps 12
recent messages. The default summary is deterministic and lossy; original
archived messages remain searchable. Search is case-insensitive substring
matching, not semantic retrieval. Custom memory backends can implement
`agent.MemoryRecall` and `agent.MemoryCompactor`.

Go callers use `agent.Session(id)`, `agent.LoadHistory`, `agent.SearchHistory`,
`agent.ListSessions`, and `agent.CompactHistory`. RPC clients use
`agent.WithSession(ctx, id)` and the generated `AgentSessions` client. Custom
memory needs `agent.SessionMemory` for isolated RPC conversations. The CLI checks
capabilities across every registered node during rolling upgrades.

A session ID selects state; it does not authorize access. Hosts apply the same
access policy to conversation, task, and schedule endpoints as to chat.

## Streaming and tools

`model.WithTokenHandler` receives incremental text during `Generate`, preserving
the provider's tool loop. OpenAI, Anthropic, Gemini, Groq, Mistral, Together, and
MiniMax support this path. Atlas Cloud, Ollama, and custom providers without this
option return their completed reply through the same agent stream. Responses
retain their newlines and indentation.

`agent.StreamAsk` emits text and local tool events. The CLI also displays
incremental foreground command output. Remote `StreamChat` carries reply text;
remote tool events and interactive remote approval prompts are not transported.
Visible partial model output is not automatically retried.

`agent/workspace` supplies reads, search, exact edits, writes, command execution,
skills, and managed background commands as ordinary `agent.WithTool` handlers.
Root instructions load at startup; reading a file includes its parent-directory
`AGENTS.md` instructions. Agents are instructed to read before editing.

Skills load on demand from `.agents/skills/NAME/SKILL.md`. The CLI also reads
`~/.agents/skills`; project skills take precedence. `workspace_save_skill` creates
a reusable procedure after approval and refuses to overwrite an existing skill.
Hosts can add read-only libraries with `workspace.WithSkills`.

```go
work, err := workspace.New(".")
if err != nil { return err }
defer work.Close()
options := append(work.Tools(), agent.WithApproval(approve))
assistant := agent.New(options...)
```

Foreground commands have a two-minute limit and bounded output. A command with
`background: true` belongs to the workspace until it exits or is stopped with
`workspace_process`. Go callers use `Workspace.Start`, `Processes`, `Stop`, and
`Close`. Hosts must close the workspace on shutdown. These processes survive a
request's cancellation, not a host crash.

File tools are restricted to the workspace. Commands use host permissions by
default. `workspace.WithContainer(image)` or `micro chat --sandbox IMAGE` runs
commands through Docker with a writable workspace mount, a read-only root,
networking disabled, dropped capabilities, and resource limits. The image needs
`sh`; Docker must be installed. Service and web tools retain their host policies.

## Work independent of chat

```bash
# Keep this host running in one terminal. --yes permits unattended tool actions.
micro chat --host project --yes
# Connect from another terminal:
micro chat project
```

The CLI host listens on loopback. Without `--yes`, it permits read-only workspace
tools and refuses unattended actions requiring approval.

Use `/background MESSAGE`, `/tasks`, `/task ID`, and `/cancel ID`. Exiting chat
leaves submitted work on its host. The recorded owner address is used when
reconnecting; failed submissions are not automatically retried. Stopping the
host cancels active tasks. A crash does not automatically replay unfinished
work or its external actions.

Applications use `agent.StartTask`, `GetTask`, and `StopTask`, or the generated
`AgentTasks` RPC client. Task IDs are the corresponding agent run IDs. Results
are stored in the selected conversation and remain inspectable after completion.

`/schedule 1h MESSAGE` saves a recurring request; `/schedules` lists definitions
and `/unschedule ID` removes future dispatches. Go callers use
`agent.ScheduleTask`, `ListSchedules`, and `RemoveSchedule`. Each dispatch runs
through `flow.Scheduled` and creates an inspectable background agent task.

Schedule definitions and next dispatch times survive restarts. Intervals are at
least one minute. Missed intervals coalesce into one dispatch; the next time is
saved before dispatch so an interrupted task is not automatically replayed.
One host owns scheduling for an agent/store namespace. This is an interval
scheduler, not a distributed cron service. Removing a schedule does not cancel
an already-running task.

## Web and extension points

`agent/web` provides bounded HTTP(S) fetching and an optional `SearchFunc`.
The CLI enables Brave search with `BRAVE_SEARCH_API_KEY`. Hosts can supply their
own search service. Fetched pages are untrusted data and tool approvals still
apply. These are web research tools, not JavaScript browser automation.

Services remain tools discoverable through the registry. Developers can provide
models, memory, approval policies, tool wrappers, web adapters, and workspace
options without using the CLI. No product-specific assistant runtime is required.

## Feature coverage

| Area | Current scope |
|---|---|
| Model/provider selection | In-chat selectors, live catalogs, endpoint-scoped settings |
| Instructions and skills | Nested instructions, project/personal loading, approved skill creation |
| Terminal interaction | Editing, completion, multiline composition, queues, cancellation and correction |
| Live output | Provider text and local command output; some providers retain buffered fallback |
| Conversations | Saved context, remote catalog/history, reconnect to a conversation |
| Memory | Automatic/manual compaction and lexical archive search |
| Background commands | Host-owned processes with list/stop/cleanup |
| Work after UI exit | Submitted tasks on a running agent host |
| Scheduled work | Persisted intervals dispatched through flows |
| Isolated commands | Optional Docker execution; host tools have separate policies |
| Programmatic extension | Existing interfaces plus optional capabilities and ordinary tools |
| Delegation | Existing agent delegation with visible local target/task and agent listing |
| Web research | Fetch and configurable search; full browser automation remains external |

The terminal preserves plain text and Markdown syntax; it is not a full-screen
rich Markdown UI. Remote approvals/tool-event forwarding, live streaming for
all providers, distributed scheduling, and full browser automation remain limits.
