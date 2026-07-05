# Quick Start

Get up and running with go-micro in under 5 minutes.

## Install

The recommended way is the precompiled binary — no Go toolchain required:

```bash
curl -fsSL https://go-micro.dev/install.sh | sh
```

Or, if you have Go and prefer to build from source:

```bash
go install go-micro.dev/v6/cmd/micro@latest
```

If the installer finishes but your shell cannot find `micro`, open [Install troubleshooting](guides/install-troubleshooting.html) before creating your first service.

## Create Your First Service

```bash
# Create a new service
micro new helloworld
cd helloworld

# Review the generated code
ls -la

# Run locally with hot reload
micro run

# Test it
curl -X POST http://localhost:8080/api/helloworld/Helloworld.Call \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'
```

## Next Steps

You now have the service half of the services → agents → workflows lifecycle running locally. Keep the on-ramp going in this order:

1. **[Install troubleshooting](guides/install-troubleshooting.html)** - verify the binary installer or `go install`, `PATH`, `micro --version`, and the no-secret smoke path.
2. `micro agent demo` - print the provider-free first-agent demo command and the next docs steps from the installed CLI.
3. **[Smallest first-agent example](https://github.com/micro/go-micro/tree/master/examples/first-agent)** - run a mock-model, no-secret agent before adding provider keys.
4. **[No-secret first-agent transcript](guides/no-secret-first-agent.html)** - run a useful support agent with a mock model before setting up a provider key.
5. **[Your First Agent](guides/your-first-agent.html)** - turn this service into an agent-callable tool, chat with it, and learn the `micro agent preflight` → `micro run` → `micro chat` loop.
6. **[Debugging your agent](guides/debugging-agents.html)** - inspect service registration, tool calls, run history, memory, provider failures, and flow handoffs when the agent does something surprising.
7. **[0→hero Reference](guides/zero-to-hero.html)** - walk the maintained scaffold → run → chat → inspect → deploy dry-run path that proves services, agents, and workflows together.

After that first-agent path, branch out to:

- **[Full Tutorial](getting-started.html)** - In-depth guide
- **[Examples](examples/)** - Runnable examples mapped to services, agents, and workflows
- **[API Reference](https://pkg.go.dev/go-micro.dev/v6)** - Complete API docs
- **[Deployment](deployment.html)** - Deploy to production

## Common Patterns

### RPC Service
```go
package main

import (
    "context"

    "go-micro.dev/v6"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

func main() {
    service := micro.NewService("greeter")
    service.Handle(new(Greeter))
    service.Run()
}
```

### Pub/Sub Event Handler
```go
import (
    "context"

    "go-micro.dev/v6"
)

func main() {
    service := micro.NewService("subscriber")
    
    // Subscribe to events
    micro.RegisterSubscriber("user.created", service.Server(), 
        func(ctx context.Context, event *UserCreatedEvent) error {
            // Handle the event here.
            return nil
        },
    )
    
    service.Run()
}
```

### Publishing Events
```go
publisher := micro.NewEvent("user.created", client)
publisher.Publish(ctx, &UserCreatedEvent{
    Email: "user@example.com",
})
```

## Get Help

- **[Discord Community](https://discord.gg/G8Gk5j3uXr)** - Chat with other users
- **[GitHub Issues](https://github.com/micro/go-micro/issues)** - Report bugs or request features
- **[Documentation](https://go-micro.dev/docs/)** - Complete docs
