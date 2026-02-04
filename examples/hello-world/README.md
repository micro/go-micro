# Hello World Example

The simplest go-micro service demonstrating core concepts.

## What It Does

This example creates a basic RPC service that:
- Listens on port 8080
- Exposes a `Greeter.Hello` method
- Returns a greeting message
- Demonstrates both programmatic and HTTP access

## Run It

```bash
go run main.go
```

The service will start and make test calls to itself, then wait for incoming requests.

## Test It

### Using curl

```bash
curl -X POST http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -H 'Micro-Endpoint: Greeter.Hello' \
  -d '{"name": "Alice"}'
```

Expected response:
```json
{"message": "Hello Alice"}
```

### Using the micro CLI

```bash
micro call greeter Greeter.Hello '{"name": "Bob"}'
```

## Code Walkthrough

1. **Define types** - Request and Response structures
2. **Implement handler** - The `Greeter` service with `Hello` method
3. **Create service** - Using `micro.New()` with options
4. **Register handler** - Link the handler to the service
5. **Run service** - Start listening for requests

## Key Concepts

- **RPC Pattern**: Method signature `func(ctx, req, rsp) error`
- **Service Discovery**: Automatic registration
- **Multiple Transports**: Works over HTTP, gRPC, etc.
- **Type Safety**: Strongly typed requests/responses

## Next Steps

- See [pubsub-events](../pubsub-events/) for event-driven patterns
- See [production-ready](../production-ready/) for a complete example
- Read the [Getting Started Guide](../../internal/website/docs/getting-started.md)
