# Model Package

The `model` package provides a typed data model layer with CRUD operations, query filtering, and multiple database backends. It uses Go generics for type-safe access.

Unlike the `store` package (which is a raw KV abstraction), `model` provides structured data access with schema awareness, WHERE queries, ordering, pagination, and indexes.

## Quick Start

```go
import (
    "context"
    "go-micro.dev/v5/model"
    "go-micro.dev/v5/model/memory"
)

// Define your model with struct tags
type User struct {
    ID    string `json:"id" model:"key"`
    Name  string `json:"name" model:"index"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

// Create a database and model
db := memory.New()
users := model.New[User](db)

ctx := context.Background()

// Create
users.Create(ctx, &User{ID: "1", Name: "Alice", Email: "alice@example.com", Age: 30})

// Read
user, _ := users.Read(ctx, "1")
fmt.Println(user.Name) // "Alice"

// Update
user.Name = "Alice Smith"
users.Update(ctx, user)

// Delete
users.Delete(ctx, "1")
```

## Struct Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `model:"key"` | Primary key field | `ID string \`model:"key"\`` |
| `model:"index"` | Create an index on this field | `Name string \`model:"index"\`` |
| `json:"name"` | Column name in the database | `Name string \`json:"name"\`` |

If no `model:"key"` tag is found, the package defaults to a field with `json:"id"` or column name `id`.

## Querying

```go
// Filter by field value
users.List(ctx, model.Where("name", "Alice"))

// Comparison operators
users.List(ctx, model.WhereOp("age", ">", 25))
users.List(ctx, model.WhereOp("name", "LIKE", "Ali%"))

// Ordering
users.List(ctx, model.OrderAsc("name"))
users.List(ctx, model.OrderDesc("age"))

// Pagination
users.List(ctx, model.Limit(10), model.Offset(20))

// Combine
users.List(ctx,
    model.Where("status", "active"),
    model.WhereOp("age", ">=", 18),
    model.OrderDesc("created_at"),
    model.Limit(25),
)

// Count
total, _ := users.Count(ctx)
active, _ := users.Count(ctx, model.Where("status", "active"))
```

## Backends

### Memory (Development & Testing)

```go
import "go-micro.dev/v5/model/memory"

db := memory.New()
```

In-memory storage. No persistence. Fast. Good for tests and prototyping.

### SQLite (Development & Single-Node Production)

```go
import "go-micro.dev/v5/model/sqlite"

db := sqlite.New("app.db")       // File-based
db := sqlite.New(":memory:")     // In-memory (testing)
```

Embedded SQL database. Zero external dependencies. Supports WHERE, indexes, ordering natively.

### Postgres (Production)

```go
import "go-micro.dev/v5/model/postgres"

db := postgres.New("postgres://user:pass@localhost/mydb?sslmode=disable")
```

Full PostgreSQL support. Best for production with rich query capabilities.

## Table Names

By default, the table name is the lowercase struct name + "s" (e.g., `User` → `users`). Override with `WithTable`:

```go
users := model.New[User](db, model.WithTable("app_users"))
```

## Database Interface

All backends implement the `model.Database` interface:

```go
type Database interface {
    Init(...Option) error
    NewTable(schema *Schema) error
    Create(ctx context.Context, schema *Schema, key string, fields map[string]any) error
    Read(ctx context.Context, schema *Schema, key string) (map[string]any, error)
    Update(ctx context.Context, schema *Schema, key string, fields map[string]any) error
    Delete(ctx context.Context, schema *Schema, key string) error
    List(ctx context.Context, schema *Schema, opts ...QueryOption) ([]map[string]any, error)
    Count(ctx context.Context, schema *Schema, opts ...QueryOption) (int64, error)
    Close() error
    String() string
}
```

## Model vs Store

| Feature | `store` | `model` |
|---------|---------|---------|
| Data format | Raw `[]byte` | Typed Go structs |
| Queries | Key prefix/suffix only | WHERE, operators, LIKE |
| Ordering | None | ORDER BY field ASC/DESC |
| Pagination | Limit/Offset on keys | Limit/Offset on results |
| Indexes | None | Via `model:"index"` tag |
| Schema | None (schemaless KV) | Auto-created from struct |
| Backends | Memory, File, MySQL, Postgres, NATS | Memory, SQLite, Postgres |
| Use case | Config, sessions, cache | Application data, entities |

## Testing

```bash
go test ./model/...
```
