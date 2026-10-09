---
title: "Interactive agent development"
description: "Implementation sequence for a programmable agent through micro chat."
---

The goal is a useful default agent that developers can replace or extend through
Go Micro's existing interfaces. `micro chat` is its interactive client. Mu can
host the same agent components with its own product services.

## Conversations and terminal interaction

Implemented in the first change:

- Store-backed local conversations, selected per working directory.
- `agent.Session(id)` and `micro.AgentSession(id)` for Go callers.
- `agent.WithSession(ctx, id)` for agent RPC callers. Session IDs select state;
  they do not grant authorization. Hosts still enforce access to their agents.
- Isolated RPC history for servers using default store-backed memory.
- Input editing, command completion, `/new`, `/sessions`, and `/resume ID`.
- A responsive prompt while work runs; Ctrl-C or `/stop` cancels that request.
- Saved provider/model settings and a masked credential prompt on first use.

Conversation memory retains the configured history window (50 messages by
default). This is not an unlimited transcript archive. Custom memory backends
can implement `agent.SessionMemory` to supply isolated memory for each session.
Backends without that interface do not advertise RPC session support. Remote servers advertise session support,
and the CLI refuses unsupported servers instead of silently sharing history.

Still needed for the full interaction milestone: multiline composition,
formatted responses, transcript browsing, model selection inside chat, visible
approval requests, and true provider streaming through the tool execution path.
Use `model.Message` and `model.History` to represent conversation content.

Run records remain indexed by agent name across conversations. Use
`micro inspect agent NAME --session ID` or `micro agent history NAME --session ID`
to select a conversation. Go callers can use `agent.LoadHistory` for stored
messages and `agent.RunListOptions{Session: id}` for run summaries. Independent
RPC sessions can run concurrently; requests for the same session are serialized.
Local chat displays retained messages when reopening a conversation. Remote
transcript retrieval and a shared session catalog are still needed.

## Workspace tools

Provide file reading, editing, search, and shell execution through existing
`agent.WithTool` handlers. Define the workspace and tool permissions explicitly.
Connect tool approval to the same UI, with cancellation covering both model
calls and child processes. Load project instructions and skills from files.

Acceptance: in an existing project, the agent reads the code, makes a requested
change, runs a check, and explains the result. The user can deny an action or
cancel without leaving an unmanaged subprocess.

## Work independent of the UI

A running agent should own work and subprocesses. The chat UI attaches to it.
Persist status and completion using the existing store and run records. Reuse
RPC for submitting work, observing it, canceling it, and reconnecting. The web
playground must call the same agent behavior rather than a separate model loop.

Acceptance: detach during a task, reconnect, and see its status and result.
Closing the UI must not repeat or cancel detached work. Stopping work remains
explicit. Host-process crash recovery is separate and must describe the limits
of replaying external actions through existing checkpoints.

## Skills and schedules

Load `SKILL.md` instructions on demand, support saving reusable procedures, and
make previous conversations searchable. Persist schedule definitions and execute
them through flows, recording their results in conversations. Keep channel and
consumer-product behavior in hosts such as Mu.

Acceptance: schedule a task, restart the host, receive its result in the intended
conversation, and reuse a saved skill on a later request.

## End-to-end acceptance

Open `micro chat` in a project, ask for a change, observe tools and output,
interrupt with a correction, detach, reconnect, and continue with the same
conversation. Passing only a mock chat or a service-generation example does not
satisfy this complete milestone.
