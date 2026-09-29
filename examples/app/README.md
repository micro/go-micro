# An app and an agent using the same service

From the repository root:

```sh
go run ./examples/app
```

Open **http://127.0.0.1:8080**. Save a note. The UI calls the `app-notes` service
through two explicit HTTP routes supplied by the example host. The `app` package
serves only the interface and assets.

If multicast discovery is unavailable, run `go run ./examples/app -local` to use
in-process discovery. The UI works in this mode, but a separate CLI cannot
discover its service. The integration test below exercises the agent in-process.

With normal discovery, the same capability is discoverable as agent tools. With the CLI installed and
a provider key available in another terminal:

```sh
export OPENAI_API_KEY=your-api-key
micro chat --provider openai
```

Ask it to list notes from `app-notes`, or add a note using that service. Refresh
the page to see the change. If other agents are registered, CLI chat routes to
those agents; select an agent configured with `AgentServices("app-notes")` or
run the example in a separate development environment.

The app definition declares `Notes.List` and `Notes.Add`. Startup checks these
endpoints against the registry. `web.Service` handles HTTP hosting and discovery;
the service owns the notes, and the agent uses the service's RPC endpoints.

This demo binds to loopback and stores notes in memory. Restarting clears them.
No model is invoked by starting the demo or using the UI. The executable needs
port 8080; stop another local gateway before starting it.

The deterministic integration test starts the RPC service, adds one note through
the HTTP route and another through the agent harness with a mock model, and
checks that the UI reads both:

```sh
go test -race ./app ./examples/app
```
