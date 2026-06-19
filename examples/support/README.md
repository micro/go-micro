# Support desk

A real-world agent built the Go Micro way: a few services, an agent that
manages them, an event that triggers it, and a human-in-the-loop gate on the
one action that touches a customer. It's the "zero to hero" shape in one
runnable file.

## The scenario

A customer files a ticket. A `ticket.created` event triggers the support
agent, which:

1. looks the customer up (`customers` service),
2. sets the ticket's priority (`tickets` service),
3. drafts a reply and emails it (`notify` service) — **but only after passing
   the approval gate.**

```
> event: events.ticket.created {"id":"ticket-1","customer":"alice@acme.com",...}

    [customers] looked up Alice (pro plan)
    [tickets]   ticket-1 → priority=high status=in_progress
    ▣ approval gate notify_NotifyService_Send(alice@acme.com) — approved
    [notify]    📨 to=alice@acme.com: "Hi Alice — thanks for reaching out..."

✓ ticket triaged and the customer was replied to — triggered by an event
```

## The pieces

- **Services** (`customers`, `tickets`, `notify`) — plain Go Micro services. The
  agent discovers their endpoints as tools automatically.
- **Agent** (`support`) — `micro.NewAgent` with those three services. It reasons
  over the ticket and calls the tools.
- **Flow** (`intake`) — triggers on `events.ticket.created` and hands the event to
  the agent: *the event is the prompt*. No human types anything.
- **Guardrail** (`ApproveTool`) — the agent can read and triage freely, but
  emailing a customer (`notify.Send`) passes through the gate first. Return
  `false` to hold it for a person or a policy; the example approves and logs.

## Run

```bash
go run main.go            # mock model — deterministic, no API key
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
