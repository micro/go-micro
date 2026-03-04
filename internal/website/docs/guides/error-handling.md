---
layout: default
title: Error Handling for AI Agents
---

# Error Handling for AI Agents

When AI agents call your services through MCP, they need to understand errors well enough to recover or inform the user. This guide covers how to write services that give agents useful error information.

## Use Typed Errors

Go Micro's `errors` package provides structured errors that the MCP gateway forwards to agents with status codes and detail messages.

```go
import "go-micro.dev/v5/errors"

func (s *Users) Get(ctx context.Context, req *GetRequest, rsp *GetResponse) error {
    if req.ID == "" {
        return errors.BadRequest("users.Get", "id is required")
    }

    user, err := s.db.FindUser(req.ID)
    if err != nil {
        return errors.NotFound("users.Get", "user %s not found", req.ID)
    }

    rsp.User = user
    return nil
}
```

Agents receive structured error responses like:

```json
{
  "error": {
    "id": "users.Get",
    "code": 404,
    "detail": "user abc-123 not found",
    "status": "Not Found"
  }
}
```

This gives the agent enough context to decide: retry with a different ID, ask the user, or report the problem.

## Error Types and When to Use Them

| Error | Code | Use When |
|-------|------|----------|
| `errors.BadRequest` | 400 | Missing or invalid input — agent should fix the request |
| `errors.Unauthorized` | 401 | Missing auth — agent needs credentials |
| `errors.Forbidden` | 403 | Insufficient permissions — agent can't do this |
| `errors.NotFound` | 404 | Resource doesn't exist — agent should try something else |
| `errors.Conflict` | 409 | Duplicate or version conflict — agent should retry or adjust |
| `errors.InternalServerError` | 500 | Server bug — agent should report to user, don't retry |

## Write Error Messages for Agents

Error messages should tell the agent **what went wrong** and **what to do about it**.

### Bad: Vague Errors

```go
return fmt.Errorf("invalid request")
return errors.BadRequest("users", "failed")
```

Agents can't recover from these — they don't know what's wrong.

### Good: Actionable Errors

```go
return errors.BadRequest("users.Create", "email is required — provide a valid email address")
return errors.BadRequest("users.Create", "email '%s' is already registered — use a different email", req.Email)
return errors.NotFound("users.Get", "no user with id '%s' — use users.List to find valid IDs", req.ID)
```

The agent now knows exactly what to fix or which tool to call next.

## Validation Patterns

Validate inputs at the top of your handler before doing any work:

```go
// CreateOrder places a new order for a user. The user must exist
// and at least one item is required.
//
// @example {"user_id": "u-1", "items": [{"product_id": "p-1", "quantity": 1}]}
func (s *Orders) CreateOrder(ctx context.Context, req *CreateRequest, rsp *CreateResponse) error {
    // Validate required fields
    if req.UserID == "" {
        return errors.BadRequest("orders.CreateOrder", "user_id is required")
    }
    if len(req.Items) == 0 {
        return errors.BadRequest("orders.CreateOrder", "at least one item is required")
    }

    // Validate each item
    for i, item := range req.Items {
        if item.ProductID == "" {
            return errors.BadRequest("orders.CreateOrder",
                "item[%d].product_id is required", i)
        }
        if item.Quantity <= 0 {
            return errors.BadRequest("orders.CreateOrder",
                "item[%d].quantity must be positive, got %d", i, item.Quantity)
        }
    }

    // All validations passed — do the work
    // ...
}
```

## Document Error Cases

Tell agents what errors to expect in your doc comments:

```go
// Transfer moves funds between two accounts. Both accounts must exist
// and the source account must have sufficient balance.
// Returns an error if the source balance is too low.
//
// @example {"from": "acc-1", "to": "acc-2", "amount": 100}
func (s *Accounts) Transfer(ctx context.Context, req *TransferRequest, rsp *TransferResponse) error {
```

The description "returns an error if the source balance is too low" helps agents anticipate failure modes and plan accordingly.

## Don't Expose Internal Details

Agents (and the users they serve) shouldn't see stack traces, database errors, or internal paths.

```go
// Bad — leaks internals
return fmt.Errorf("pq: duplicate key value violates unique constraint \"users_email_key\"")

// Good — clear message, no internals
return errors.Conflict("users.Create", "a user with email '%s' already exists", req.Email)
```

## Idempotency for Retries

Agents may retry failed operations. Design critical operations to be idempotent:

```go
// CreateOrUpdate upserts a config value. Safe to call multiple times
// with the same key — it will create on first call, update on subsequent calls.
//
// @example {"key": "theme", "value": "dark"}
func (s *Config) CreateOrUpdate(ctx context.Context, req *SetRequest, rsp *SetResponse) error {
```

When an operation is naturally idempotent, say so in the doc comment. Agents will learn they can safely retry.

## Next Steps

- [Tool Descriptions Guide](tool-descriptions.md) - Write documentation that agents can use effectively
- [MCP Security Guide](mcp-security.md) - Auth and scopes for restricting agent access
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
