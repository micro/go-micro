# Quick Start

Get up and running with go-micro in under 5 minutes.

## Install

```bash
go install go-micro.dev/v5/cmd/micro@v5.16.0
```

> **Note:** Use a specific version instead of `@latest` to avoid module path conflicts. See [releases](https://github.com/micro/go-micro/releases) for the latest version.

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

- **[Full Tutorial](getting-started.md)** - In-depth guide
- **[Examples](examples/)** - Learn by example
- **[API Reference](https://pkg.go.dev/go-micro.dev/v5)** - Complete API docs
- **[Deployment](deployment.md)** - Deploy to production

## Common Patterns

### RPC Service
```go
package main

import "go-micro.dev/v5"

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

func main() {
    service := micro.New("greeter")
    service.Handle(new(Greeter))
    service.Run()
}
```

### Pub/Sub Event Handler
```go
import "go-micro.dev/v5"

func main() {
    service := micro.New("subscriber")
    
    // Subscribe to events
    micro.RegisterSubscriber("user.created", service.Server(), 
        func(ctx context.Context, event *UserCreatedEvent) error {
            log.Infof("User created: %s", event.Email)
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

- **[Discord Community](https://discord.gg/jwTYuUVAGh)** - Chat with other users
- **[GitHub Issues](https://github.com/micro/go-micro/issues)** - Report bugs or request features
- **[Documentation](https://go-micro.dev/docs/)** - Complete docs

