# Multi-Service Example

Run multiple services in a single binary — the **modular monolith** pattern.

## What It Shows

- Two independent services (`users` and `orders`) in one process
- Each service gets isolated server, client, store, and cache
- Shared registry and broker for inter-service communication
- Coordinated lifecycle with `micro.NewGroup()`

## Run It

```bash
go run .
```

This starts both services:
- **Users** on `:9001` — provides `Users.Lookup`
- **Orders** on `:9002` — provides `Orders.Create`

## Call the Services

From another terminal:

```bash
# Look up a user
micro call users Users.Lookup '{"id": "1"}'

# Create an order
micro call orders Orders.Create '{"user_id": "1"}'
```

## How It Works

```go
// Create isolated services
users := micro.New("users", micro.Address(":9001"))
orders := micro.New("orders", micro.Address(":9002"))

// Register handlers
users.Handle(new(Users))
orders.Handle(new(Orders))

// Run together — stops all when one exits
g := micro.NewGroup(users, orders)
g.Run()
```

Each service gets its own server and client, so they can be split into separate binaries later without code changes. The group handles coordinated startup and graceful shutdown.

## When to Use This Pattern

- **Early development** — run everything locally in one process
- **Testing** — integration tests without Docker or networking
- **Small teams** — fewer moving parts until you need to scale independently
- **Gradual migration** — start monolith, split services one at a time

## Splitting Later

When you're ready to run services independently, just move each into its own `main.go`:

```go
func main() {
    svc := micro.New("users", micro.Address(":9001"))
    svc.Handle(new(Users))
    svc.Run()
}
```

No handler code changes needed — the registry handles discovery automatically.
