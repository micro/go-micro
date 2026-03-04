---
layout: doc
title: Data Model
permalink: /docs/model.html
description: "Typed data model layer with CRUD operations, queries, and pluggable backends"
---

# Data Model

The `model` package provides a typed data model layer for Go Micro services. Define Go structs, tag your fields, and get type-safe CRUD operations with queries, filtering, ordering, and pagination.

## Quick Start

```go
package main

import (
    "context"

    "go-micro.dev/v5"
    "go-micro.dev/v5/model"
)

type Task struct {
    ID     string `json:"id" model:"key"`
    Title  string `json:"title"`
    Done   bool   `json:"done"`
    Owner  string `json:"owner" model:"index"`
}

func main() {
    service := micro.New("tasks")

    // Create a typed model backed by the service's database
    tasks := model.New[Task](service.Model())

    ctx := context.Background()

    // Create a record
    tasks.Create(ctx, &Task{ID: "1", Title: "Ship it", Owner: "alice"})

    // Read by key
    task, _ := tasks.Read(ctx, "1")

    // Update
    task.Done = true
    tasks.Update(ctx, task)

    // List with filters
    aliceTasks, _ := tasks.List(ctx, model.Where("owner", "alice"))

    // Delete
    tasks.Delete(ctx, "1")
}
```

## Defining Models

Models are plain Go structs. Use struct tags to control storage behavior:

| Tag | Purpose | Example |
|-----|---------|---------|
| `model:"key"` | Primary key field | `ID string \`model:"key"\`` |
| `model:"index"` | Create an index on this field | `Email string \`model:"index"\`` |
| `json:"name"` | Column name in the database | `Name string \`json:"name"\`` |

If no `model:"key"` tag is found, the package defaults to a field with `json:"id"` or a field named `ID`.

Table names are auto-derived from the struct name (lowercased + "s"), e.g. `User` → `users`. Override with `model.WithTable("custom_name")`.

```go
type User struct {
    ID        string `json:"id" model:"key"`
    Name      string `json:"name"`
    Email     string `json:"email" model:"index"`
    Age       int    `json:"age"`
    CreatedAt string `json:"created_at"`
}

// Auto-derived table: "users"
users := model.New[User](db)

// Custom table name
users := model.New[User](db, model.WithTable("app_users"))
```

## CRUD Operations

```go
// Create — inserts a new record (returns ErrDuplicateKey if key exists)
err := users.Create(ctx, &User{ID: "1", Name: "Alice"})

// Read — retrieves by primary key (returns ErrNotFound if missing)
user, err := users.Read(ctx, "1")

// Update — modifies an existing record (returns ErrNotFound if missing)
user.Name = "Alice Smith"
err = users.Update(ctx, user)

// Delete — removes by primary key (returns ErrNotFound if missing)
err = users.Delete(ctx, "1")
```

## Queries

Use query options to filter, order, and paginate results:

### Filters

```go
// Equality
results, _ := users.List(ctx, model.Where("email", "alice@example.com"))

// Operators: =, !=, <, >, <=, >=, LIKE
results, _ = users.List(ctx, model.WhereOp("age", ">=", 18))
results, _ = users.List(ctx, model.WhereOp("name", "LIKE", "Ali%"))

// Multiple filters (AND)
results, _ = users.List(ctx,
    model.Where("owner", "alice"),
    model.WhereOp("age", ">", 25),
)
```

### Ordering

```go
results, _ := users.List(ctx, model.OrderAsc("name"))
results, _ = users.List(ctx, model.OrderDesc("created_at"))
```

### Pagination

```go
results, _ := users.List(ctx,
    model.Limit(10),
    model.Offset(20),
)
```

### Counting

```go
total, _ := users.Count(ctx)
active, _ := users.Count(ctx, model.Where("active", true))
```

## Backends

The model layer uses Go Micro's pluggable interface pattern. All backends implement `model.Database`.

### Memory (Default)

Zero-config, in-memory storage. Data doesn't persist across restarts. Ideal for development and testing.

```go
service := micro.New("myservice")
tasks := model.New[Task](service.Model()) // memory backend by default
```

Or create directly:

```go
import "go-micro.dev/v5/model/memory"

db := memory.New()
tasks := model.New[Task](db)
```

### SQLite

File-based database. Good for local development or single-node production.

```go
import "go-micro.dev/v5/model/sqlite"

db, err := sqlite.New(model.WithDSN("file:app.db"))
service := micro.New("myservice", micro.Model(db))
```

### Postgres

Production-grade with connection pooling.

```go
import "go-micro.dev/v5/model/postgres"

db, err := postgres.New(model.WithDSN("postgres://user:pass@localhost/myapp?sslmode=disable"))
service := micro.New("myservice", micro.Model(db))
```

## Service Integration

The `Service` interface provides `Model()` alongside `Client()` and `Server()`:

```go
service := micro.New("users", micro.Address(":9001"))

// Access the three core components
client := service.Client()  // Call other services
server := service.Server()  // Handle requests
db     := service.Model()   // Data persistence

// Create typed models from the shared database
users := model.New[User](db)
posts := model.New[Post](db)

// Use in your handler
service.Handle(&UserHandler{users: users, posts: posts})
service.Run()
```

A handler that uses all three:

```go
type OrderHandler struct {
    orders  *model.Model[Order]
    client  client.Client
}

// CreateOrder saves an order and notifies the shipping service
func (h *OrderHandler) CreateOrder(ctx context.Context, req *CreateReq, rsp *CreateRsp) error {
    // Save to database via Model
    order := &Order{ID: req.ID, Item: req.Item, Status: "pending"}
    if err := h.orders.Create(ctx, order); err != nil {
        return err
    }

    // Call another service via Client
    shipClient := proto.NewShippingService("shipping", h.client)
    _, err := shipClient.Ship(ctx, &proto.ShipRequest{OrderID: order.ID})

    return err
}
```

## Error Handling

The model package returns two sentinel errors:

```go
import "go-micro.dev/v5/model"

// Check for not found
user, err := users.Read(ctx, "missing")
if errors.Is(err, model.ErrNotFound) {
    // record doesn't exist
}

// Check for duplicate key
err = users.Create(ctx, &User{ID: "1", Name: "Alice"})
err = users.Create(ctx, &User{ID: "1", Name: "Bob"})
if errors.Is(err, model.ErrDuplicateKey) {
    // key "1" already exists
}
```

## Swapping Backends

Follow the standard Go Micro pattern — use in-memory for development, swap to a real database for production:

```go
func main() {
    var db model.Database

    if os.Getenv("ENV") == "production" {
        db, _ = postgres.New(model.WithDSN(os.Getenv("DATABASE_URL")))
    } else {
        db = memory.New()
    }

    service := micro.New("myservice", micro.Model(db))
    // ... same application code regardless of backend
}
```
