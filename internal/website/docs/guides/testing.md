---
layout: default
---

# Testing Micro Services

The `testing` package provides utilities for testing micro services in isolation.

## Quick Start

```go
import (
    "testing"
    "go-micro.dev/v5/test"
)

func TestGreeter(t *testing.T) {
    h := test.NewHarness(t)
    defer h.Stop()

    h.Name("greeter").Register(new(GreeterHandler))
    h.Start()

    var rsp HelloResponse
    err := h.Call("GreeterHandler.Hello", &HelloRequest{Name: "World"}, &rsp)
    if err != nil {
        t.Fatal(err)
    }

    if rsp.Message != "Hello World" {
        t.Errorf("expected 'Hello World', got '%s'", rsp.Message)
    }
}
```

## How It Works

The harness creates isolated instances of:
- **Registry** - In-memory registry for service discovery
- **Transport** - HTTP transport for RPC
- **Broker** - In-memory broker for events

This allows your service to run without affecting or being affected by other services.

## API

### Creating a Harness

```go
h := test.NewHarness(t)
defer h.Stop()  // Always stop to clean up
```

### Configuring

```go
h.Name("myservice")      // Set service name (default: "test")
h.Register(handler)      // Set the handler
h.Start()                // Start the service
```

### Making Calls

```go
// Simple call
err := h.Call("Handler.Method", &request, &response)

// With context
err := h.CallContext(ctx, "Handler.Method", &request, &response)
```

### Assertions

```go
// Check service is running
h.AssertServiceRunning()

// Check call succeeds
h.AssertCallSucceeds("Handler.Method", &req, &rsp)

// Check call fails
h.AssertCallFails("Handler.Method", &req, &rsp)
```

### Advanced Access

```go
// Get the client for custom calls
client := h.Client()

// Get the server
server := h.Server()

// Get the registry
reg := h.Registry()
```

## Example: Testing a User Service

```go
package users

import (
    "context"
    "testing"
    "go-micro.dev/v5/test"
)

type UsersHandler struct {
    users map[string]*User
}

type User struct {
    ID   string
    Name string
}

type CreateRequest struct {
    Name string
}

type CreateResponse struct {
    User *User
}

func (h *UsersHandler) Create(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
    user := &User{ID: "123", Name: req.Name}
    h.users[user.ID] = user
    rsp.User = user
    return nil
}

func TestUsersCreate(t *testing.T) {
    h := test.NewHarness(t)
    defer h.Stop()

    handler := &UsersHandler{users: make(map[string]*User)}
    h.Name("users").Register(handler)
    h.Start()

    var rsp CreateResponse
    h.AssertCallSucceeds("UsersHandler.Create", &CreateRequest{Name: "Alice"}, &rsp)

    if rsp.User == nil {
        t.Fatal("user is nil")
    }
    if rsp.User.Name != "Alice" {
        t.Errorf("expected Alice, got %s", rsp.User.Name)
    }

    // Verify the user was stored
    if _, ok := handler.users["123"]; !ok {
        t.Error("user not stored in handler")
    }
}
```

## Limitations

Due to go-micro's global defaults, each harness should test **one service**. If you need to test service-to-service communication, consider:

1. **Integration tests** - Run services as separate processes
2. **Mock clients** - Mock the client calls to dependent services
3. **Contract tests** - Test service interfaces separately

## Tips

1. **Always defer Stop()** - Ensures cleanup even if test fails
2. **Use meaningful names** - `h.Name("users")` makes logs clearer
3. **Test edge cases** - Use `AssertCallFails` for error paths
4. **Keep handlers simple** - Complex handlers are harder to test
