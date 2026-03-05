---
layout: doc
title: Data Model
permalink: /docs/model.html
description: "Structured data model layer with CRUD operations, queries, and pluggable backends"
---

# Data Model

The `model` package provides a structured data model layer for Go Micro services. Define Go structs, tag your fields, and get CRUD operations with queries, filtering, ordering, and pagination.

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

    // Register your type with the service's model backend
    db := service.Model()
    db.Register(&Task{})

    ctx := context.Background()

    // Create a record
    db.Create(ctx, &Task{ID: "1", Title: "Ship it", Owner: "alice"})

    // Read by key
    task := &Task{}
    db.Read(ctx, "1", task)

    // Update
    task.Done = true
    db.Update(ctx, task)

    // List with filters
    var aliceTasks []*Task
    db.List(ctx, &aliceTasks, model.Where("owner", "alice"))

    // Delete
    db.Delete(ctx, "1", &Task{})
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

// Register with auto-derived table: "users"
db.Register(&User{})

// Custom table name
db.Register(&User{}, model.WithTable("app_users"))
```

## CRUD Operations

```go
// Create — inserts a new record (returns ErrDuplicateKey if key exists)
err := db.Create(ctx, &User{ID: "1", Name: "Alice"})

// Read — retrieves by primary key (returns ErrNotFound if missing)
user := &User{}
err = db.Read(ctx, "1", user)

// Update — modifies an existing record (returns ErrNotFound if missing)
user.Name = "Alice Smith"
err = db.Update(ctx, user)

// Delete — removes by primary key (returns ErrNotFound if missing)
err = db.Delete(ctx, "1", &User{})
```

## Queries

Use query options to filter, order, and paginate results:

### Filters

```go
var results []*User

// Equality
db.List(ctx, &results, model.Where("email", "alice@example.com"))

// Operators: =, !=, <, >, <=, >=, LIKE
db.List(ctx, &results, model.WhereOp("age", ">=", 18))
db.List(ctx, &results, model.WhereOp("name", "LIKE", "Ali%"))

// Multiple filters (AND)
db.List(ctx, &results,
    model.Where("owner", "alice"),
    model.WhereOp("age", ">", 25),
)
```

### Ordering

```go
db.List(ctx, &results, model.OrderAsc("name"))
db.List(ctx, &results, model.OrderDesc("created_at"))
```

### Pagination

```go
db.List(ctx, &results,
    model.Limit(10),
    model.Offset(20),
)
```

### Counting

```go
total, _ := db.Count(ctx, &User{})
active, _ := db.Count(ctx, &User{}, model.Where("active", true))
```

## Backends

The model layer uses Go Micro's pluggable interface pattern. All backends implement `model.Model`.

### Memory (Default)

Zero-config, in-memory storage. Data doesn't persist across restarts. Ideal for development and testing.

```go
service := micro.New("myservice")
db := service.Model() // memory backend by default
db.Register(&Task{})
```

Or create directly:

```go
import "go-micro.dev/v5/model"

db := model.NewModel()
db.Register(&Task{})
```

### SQLite

File-based database. Good for local development or single-node production.

```go
import "go-micro.dev/v5/model/sqlite"

db := sqlite.New("app.db")
service := micro.New("myservice", micro.Model(db))
```

### Postgres

Production-grade with connection pooling.

```go
import "go-micro.dev/v5/model/postgres"

db := postgres.New("postgres://user:pass@localhost/myapp?sslmode=disable")
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

// Register your types
db.Register(&User{})
db.Register(&Post{})

// Use in your handler
service.Handle(&UserHandler{db: db})
service.Run()
```

A handler that uses all three:

```go
type OrderHandler struct {
    db     model.Model
    client client.Client
}

// CreateOrder saves an order and notifies the shipping service
func (h *OrderHandler) CreateOrder(ctx context.Context, req *CreateReq, rsp *CreateRsp) error {
    // Save to database via Model
    order := &Order{ID: req.ID, Item: req.Item, Status: "pending"}
    if err := h.db.Create(ctx, order); err != nil {
        return err
    }

    // Call another service via Client
    shipClient := proto.NewShippingService("shipping", h.client)
    _, err := shipClient.Ship(ctx, &proto.ShipRequest{OrderID: order.ID})

    return err
}
```

## Error Handling

The model package returns sentinel errors:

```go
import "go-micro.dev/v5/model"

// Check for not found
err := db.Read(ctx, "missing", &User{})
if errors.Is(err, model.ErrNotFound) {
    // record doesn't exist
}

// Check for duplicate key
err = db.Create(ctx, &User{ID: "1", Name: "Alice"})
err = db.Create(ctx, &User{ID: "1", Name: "Bob"})
if errors.Is(err, model.ErrDuplicateKey) {
    // key "1" already exists
}
```

## Swapping Backends

Follow the standard Go Micro pattern — use in-memory for development, swap to a real database for production:

```go
func main() {
    var db model.Model

    if os.Getenv("ENV") == "production" {
        db = postgres.New(os.Getenv("DATABASE_URL"))
    } else {
        db = model.NewModel()
    }

    service := micro.New("myservice", micro.Model(db))
    // ... same application code regardless of backend
}
```
