---
layout: default
---

# API Gateway with Backend Services

A complete example showing an API gateway routing to multiple backend microservices.

## Architecture

```
                  ┌─────────────┐
   Client ───────>│ API Gateway │
                  └──────┬──────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
    ┌─────▼────┐   ┌────▼─────┐  ┌────▼─────┐
    │  Users   │   │  Orders  │  │ Products │
    │ Service  │   │ Service  │  │ Service  │
    └──────────┘   └──────────┘  └──────────┘
          │              │              │
          └──────────────┼──────────────┘
                         │
                  ┌──────▼──────┐
                  │  PostgreSQL │
                  └─────────────┘
```

## Services

### 1. Users Service

```go
// services/users/main.go
package main

import (
    "context"
    "database/sql"
    "go-micro.dev/v5"
    "go-micro.dev/v5/server"
    _ "github.com/lib/pq"
)

type User struct {
    ID    int64  `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

type UsersService struct {
    db *sql.DB
}

type GetUserRequest struct {
    ID int64 `json:"id"`
}

type GetUserResponse struct {
    User *User `json:"user"`
}

func (s *UsersService) Get(ctx context.Context, req *GetUserRequest, rsp *GetUserResponse) error {
    var u User
    err := s.db.QueryRow("SELECT id, email, name FROM users WHERE id = $1", req.ID).
        Scan(&u.ID, &u.Email, &u.Name)
    if err != nil {
        return err
    }
    rsp.User = &u
    return nil
}

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/users?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    svc := micro.NewService(
        micro.Name("users"),
        micro.Version("1.0.0"),
    )

    svc.Init()

    server.RegisterHandler(svc.Server(), &UsersService{db: db})

    if err := svc.Run(); err != nil {
        panic(err)
    }
}
```

### 2. Orders Service

```go
// services/orders/main.go
package main

import (
    "context"
    "database/sql"
    "time"
    "go-micro.dev/v5"
    "go-micro.dev/v5/client"
    "go-micro.dev/v5/server"
)

type Order struct {
    ID        int64     `json:"id"`
    UserID    int64     `json:"user_id"`
    ProductID int64     `json:"product_id"`
    Amount    float64   `json:"amount"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

type OrdersService struct {
    db     *sql.DB
    client client.Client
}

type CreateOrderRequest struct {
    UserID    int64   `json:"user_id"`
    ProductID int64   `json:"product_id"`
    Amount    float64 `json:"amount"`
}

type CreateOrderResponse struct {
    Order *Order `json:"order"`
}

func (s *OrdersService) Create(ctx context.Context, req *CreateOrderRequest, rsp *CreateOrderResponse) error {
    // Verify user exists
    userReq := s.client.NewRequest("users", "UsersService.Get", &struct{ ID int64 }{ID: req.UserID})
    userRsp := &struct{ User interface{} }{}
    if err := s.client.Call(ctx, userReq, userRsp); err != nil {
        return err
    }

    // Verify product exists
    prodReq := s.client.NewRequest("products", "ProductsService.Get", &struct{ ID int64 }{ID: req.ProductID})
    prodRsp := &struct{ Product interface{} }{}
    if err := s.client.Call(ctx, prodReq, prodRsp); err != nil {
        return err
    }

    // Create order
    var o Order
    err := s.db.QueryRow(`
        INSERT INTO orders (user_id, product_id, amount, status, created_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, user_id, product_id, amount, status, created_at
    `, req.UserID, req.ProductID, req.Amount, "pending", time.Now()).
        Scan(&o.ID, &o.UserID, &o.ProductID, &o.Amount, &o.Status, &o.CreatedAt)

    if err != nil {
        return err
    }

    rsp.Order = &o
    return nil
}

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@localhost/orders?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    svc := micro.NewService(
        micro.Name("orders"),
        micro.Version("1.0.0"),
    )

    svc.Init()

    server.RegisterHandler(svc.Server(), &OrdersService{
        db:     db,
        client: svc.Client(),
    })

    if err := svc.Run(); err != nil {
        panic(err)
    }
}
```

### 3. API Gateway

```go
// gateway/main.go
package main

import (
    "encoding/json"
    "net/http"
    "strconv"
    "go-micro.dev/v5"
    "go-micro.dev/v5/client"
)

type Gateway struct {
    client client.Client
}

func (g *Gateway) GetUser(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Query().Get("id")
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    req := g.client.NewRequest("users", "UsersService.Get", &struct{ ID int64 }{ID: id})
    rsp := &struct{ User interface{} }{}

    if err := g.client.Call(r.Context(), req, rsp); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rsp)
}

func (g *Gateway) CreateOrder(w http.ResponseWriter, r *http.Request) {
    var body struct {
        UserID    int64   `json:"user_id"`
        ProductID int64   `json:"product_id"`
        Amount    float64 `json:"amount"`
    }

    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    req := g.client.NewRequest("orders", "OrdersService.Create", body)
    rsp := &struct{ Order interface{} }{}

    if err := g.client.Call(r.Context(), req, rsp); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(rsp)
}

func main() {
    svc := micro.NewService(
        micro.Name("api.gateway"),
    )
    svc.Init()

    gw := &Gateway{client: svc.Client()}

    http.HandleFunc("/users", gw.GetUser)
    http.HandleFunc("/orders", gw.CreateOrder)

    http.ListenAndServe(":8080", nil)
}
```

## Running the Example

### Development (Local)

```bash
# Terminal 1: Users service
cd services/users
go run main.go

# Terminal 2: Products service
cd services/products
go run main.go

# Terminal 3: Orders service
cd services/orders
go run main.go

# Terminal 4: API Gateway
cd gateway
go run main.go
```

### Testing

```bash
# Get user
curl http://localhost:8080/users?id=1

# Create order
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"user_id": 1, "product_id": 100, "amount": 99.99}'
```

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"

  users:
    build: ./services/users
    environment:
      MICRO_REGISTRY: nats
      MICRO_REGISTRY_ADDRESS: nats://nats:4222
      DATABASE_URL: postgres://postgres:secret@postgres/users
    depends_on:
      - postgres
      - nats

  products:
    build: ./services/products
    environment:
      MICRO_REGISTRY: nats
      MICRO_REGISTRY_ADDRESS: nats://nats:4222
      DATABASE_URL: postgres://postgres:secret@postgres/products
    depends_on:
      - postgres
      - nats

  orders:
    build: ./services/orders
    environment:
      MICRO_REGISTRY: nats
      MICRO_REGISTRY_ADDRESS: nats://nats:4222
      DATABASE_URL: postgres://postgres:secret@postgres/orders
    depends_on:
      - postgres
      - nats

  gateway:
    build: ./gateway
    ports:
      - "8080:8080"
    environment:
      MICRO_REGISTRY: nats
      MICRO_REGISTRY_ADDRESS: nats://nats:4222
    depends_on:
      - users
      - products
      - orders

  nats:
    image: nats:latest
    ports:
      - "4222:4222"
```

Run with:
```bash
docker-compose up
```

## Key Patterns

1. **Service isolation**: Each service owns its database
2. **Service communication**: Via Go Micro client
3. **Gateway pattern**: Single entry point for clients
4. **Error handling**: Proper HTTP status codes
5. **Registry**: mDNS for local, NATS for Docker

## Production Considerations

- Add authentication/authorization
- Implement request tracing
- Add circuit breakers for service calls
- Use connection pooling
- Add rate limiting
- Implement proper logging
- Use health checks
- Add metrics collection

See [Production Patterns](../realworld/) for more details.
