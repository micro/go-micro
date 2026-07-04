---
layout: default
---

# No-secret first-agent transcript

This is the fastest first-agent success path when you do not have a provider key
handy. It starts from the maintained `examples/support` app and uses the
repository harness that CI already runs: real Go Micro services, registry,
broker, client, store, agent loop, flow handoff, and guardrail code with only the
LLM provider mocked.

Use it before the live-provider [Your First Agent](your-first-agent.html)
walkthrough when you want to see the services → agents → workflows lifecycle run
end to end with no secrets.

## What this proves

- **Services** expose typed `customers`, `tickets`, and `notify` endpoints.
- **The `support` agent** discovers those endpoints as tools and uses them to
  triage a ticket.
- **The `intake` flow** turns a `ticket.created` event into an agent run.
- **The approval gate** intercepts the customer email action before the tool
  executes.

## Transcript

From a fresh clone of the repository, first run the smallest service-backed agent:

```sh
git clone https://github.com/micro/go-micro.git
cd go-micro
go run ./examples/first-agent
```

Then run the maintained support-agent transcript that exercises the full lifecycle:

```sh
go run ./examples/support
```

The default provider is `mock`, so the command does not need `ANTHROPIC_API_KEY`,
`OPENAI_API_KEY`, or any other secret. A healthy run prints the event, service
calls, guardrail decision, and final support-agent reply in one terminal:

```text
> event: events.ticket.created {"id":"ticket-1","customer":"alice@acme.com",...}

    [customers] looked up Alice (pro plan)
    [tickets]   ticket-1 → priority=high status=in_progress
    ▣ approval gate notify_NotifyService_Send(alice@acme.com) — approved
    [notify]    📨 to=alice@acme.com: "Hi Alice — thanks for reaching out..."

support agent: Hi Alice — thanks for reaching out...

✓ ticket triaged and the customer was replied to — triggered by an event
```

That single run is the no-secret version of the first-agent loop: a service
capability exists, an agent calls it as a tool, and workflow infrastructure can
trigger and inspect the work.

## CI-backed check

Run the same deterministic paths as focused tests:

```sh
go test ./examples/first-agent -run TestRunFirstAgent -count=1
go test ./examples/support -run TestRunSupportMockSmoke -count=1
```

For the broader no-secret contract that also checks scaffold, chat/inspect CLI
boundaries, flow history, deploy dry-run, and mock provider conformance, run:

```sh
make harness
```

## Equivalent scaffold → run → chat → inspect path

When you are ready to build the smaller live-agent version yourself, follow
[Your First Agent](your-first-agent.html). The command shape is the same, but a
live `micro chat` turn needs a provider key because the model is no longer
mocked:

```sh
micro agent preflight
micro run
micro chat assistant
micro inspect agent assistant
```

CI keeps those CLI boundaries present with:

```sh
go test ./cmd/micro -run TestFirstAgentWalkthroughCLIBoundaries -count=1
```

## Debug transcript checkpoint

A successful first chat turn should always leave an inspectable trail. After the
chat command finishes, continue the same terminal transcript with the inspection
and history commands before changing prompts or provider settings:

```sh
micro chat assistant --prompt "Triage ticket-1 for Alice"
micro inspect agent assistant --limit 1
micro agent history assistant
```

The inspection output is the checkpoint that the runnable loop did not stop at
chat: it should show a recent agent run with a status, event count, last event,
and trace breadcrumb when tracing is configured. `micro agent history assistant`
then confirms the conversation memory that future turns will reuse. If either
command is empty after a successful chat turn, keep the failing transcript and
use [Debugging your agent](debugging-agents.html) to check provider failures, run
history, memory, and tool-call inspection before changing application code.

If `micro agent preflight` reports a missing provider key, you can still use this no-secret path because it runs against the mock model; the command now prints this guide as the next step for that failure. If chat behaves unexpectedly, continue to
[Debugging your agent](debugging-agents.html) for provider checks, run history,
memory, and tool-call inspection.
