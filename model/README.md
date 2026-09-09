# Model Package

The `model` package provides a structured data storage interface with CRUD operations, query filtering, and multiple database backends.

Unlike the `store` package (which is a raw KV abstraction), `model` provides structured data access with schema awareness, WHERE queries, ordering, pagination, and indexes.

## Quick Start

```go
import (
    "context"
    "go-micro.dev/v5/model"
)

// Define your model with struct tags
type User struct {
    ID    string `json:"id" model:"key"`
    Name  string `json:"name" model:"index"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

// Create a model and register your type
db := model.NewModel()
db.Register(&User{})

ctx := context.Background()

// Create
db.Create(ctx, &User{ID: "1", Name: "Alice", Email: "alice@example.com", Age: 30})

// Read
user := &User{}
db.Read(ctx, "1", user)
fmt.Println(user.Name) // "Alice"

// Update
user.Name = "Alice Smith"
db.Update(ctx, user)

// Delete
db.Delete(ctx, "1", &User{})
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
var users []*User
db.List(ctx, &users, model.Where("name", "Alice"))

// Comparison operators
db.List(ctx, &users, model.WhereOp("age", ">", 25))
db.List(ctx, &users, model.WhereOp("name", "LIKE", "Ali%"))

// Ordering
db.List(ctx, &users, model.OrderAsc("name"))
db.List(ctx, &users, model.OrderDesc("age"))

// Pagination
db.List(ctx, &users, model.Limit(10), model.Offset(20))

// Combine
db.List(ctx, &users,
    model.Where("status", "active"),
    model.WhereOp("age", ">=", 18),
    model.OrderDesc("created_at"),
    model.Limit(25),
)

// Count
total, _ := db.Count(ctx, &User{})
active, _ := db.Count(ctx, &User{}, model.Where("status", "active"))
```

## Backends

### Memory (Development & Testing)

```go
import "go-micro.dev/v5/model"

db := model.NewModel()
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

By default, the table name is the lowercase struct name + "s" (e.g., `User` → `users`). Override with `model.WithTable`:

```go
db.Register(&User{}, model.WithTable("app_users"))
```

## Model Interface

All backends implement the `model.Model` interface:

```go
type Model interface {
    Init(...Option) error
    Register(v interface{}, opts ...RegisterOption) error
    Create(ctx context.Context, v interface{}) error
    Read(ctx context.Context, key string, v interface{}) error
    Update(ctx context.Context, v interface{}) error
    Delete(ctx context.Context, key string, v interface{}) error
    List(ctx context.Context, result interface{}, opts ...QueryOption) error
    Count(ctx context.Context, v interface{}, opts ...QueryOption) (int64, error)
    Close() error
    String() string
}
```

## Model vs Store

| Feature | `store` | `model` |
|---------|---------|---------|
| Data format | Raw `[]byte` | Go structs |
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
