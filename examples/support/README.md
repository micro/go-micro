# Zero-to-hero support desk

A maintained 0-to-hero reference for the Go Micro lifecycle: scaffold a few
typed services, run them in one process, let an agent chat with those services
as tools, then inspect the agent result. It is one
runnable file and one CI smoke test, so the reference path stays honest as the
framework evolves.

## The path

1. **Scaffold services** — `customers`, `tickets`, and `notify` are ordinary
   typed Go Micro services. Their request/response structs and method comments
   become the tool contract the agent sees.
2. **Run the harness** — the example starts an in-memory registry, broker,
   client, store, services and agent in one process; no external
   dependencies or API key are required for the default run.
3. **Chat through an agent** — the `support` agent receives the ticket as
   a prompt and calls service tools to look up the customer, triage the ticket,
   and draft a reply.
4. **Inspect the agent** — the example prints tool calls, the approval decision
   and the agent response.

## The scenario

A customer files a ticket. A request asks the support agent, which:

1. looks the customer up (`customers` service),
2. sets the ticket's priority (`tickets` service),
3. drafts a reply and emails it (`notify` service) — **but only after passing
   the approval gate.**

```
> ticket: {"id":"ticket-1","customer":"alice@acme.com",...}

    [customers] looked up Alice (pro plan)
    [tickets]   ticket-1 → priority=high status=in_progress
    ▣ approval gate notify_NotifyService_Send(alice@acme.com) — approved
    [notify]    📨 to=alice@acme.com: "Hi Alice — thanks for reaching out..."

✓ ticket triaged and the customer was replied to — handled by the agent
```

## The pieces

- **Services** (`customers`, `tickets`, `notify`) — plain Go Micro services. The
  agent discovers their endpoints as tools automatically.
- **Agent** (`support`) — `micro.NewAgent` with those three services. It reasons
  over the ticket and calls the tools.
- **Guardrail** (`ApproveTool`) — the agent can read and triage freely, but
  emailing a customer (`notify.Send`) passes through the gate first. Return
  `false` to hold it for a person or a policy; the example approves and logs.

## Expected inspect transcript

The provider-free run prints the same visible checkpoints a new developer should
compare against after agent execution. The transcript includes service
tool calls, the approval gate, and the inspect/run-history commands that prove
the agent handled the request.

```text
> ticket: {"customer":"alice@acme.com","id":"ticket-1","subject":"Can't log in"}

    [customers] looked up Alice (pro plan)
    [tickets] ticket-1 → priority=high status=in_progress
  ▣ approval gate notify_NotifyService_Send(alice@acme.com) — approved
    [notify] 📨 to=alice@acme.com: "Hi Alice — thanks for reaching out. We've bumped this to high priority and are on it."

support agent: Triaged ticket-1 for Alice and sent a reply.

inspect transcript:
  micro agent history support

✓ ticket triaged and the customer was replied to — handled by the agent
```

## Run

```bash
go run main.go            # mock model — deterministic, no API key
```

The maintained check is the same deterministic path:

```bash
go test ./examples/support
```

Against a live model, the agent reasons about the ticket itself instead of
following the script:

```bash
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY, GEMINI_API_KEY, ...
go run main.go -provider anthropic
```

## What to change next

- Make the gate real: return `false` from `ApproveTool` for billing actions, or
  route the decision to a human.
- Expose the agent over A2A so another team's agent can file tickets — add
  `agent.WithA2A(":4000")`.
- Add a `kb` (knowledge base) service and watch the agent search it before
  replying.
