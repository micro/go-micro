---
layout: default
---

# Hello Service

A minimal HTTP service using Go Micro, with a single endpoint.

## Service

```go
package main

import (
    "context"
    "go-micro.dev/v5"
)

type Request struct { Name string `json:"name"` }

type Response struct { Message string `json:"message"` }

type Say struct{}

func (h *Say) Hello(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Message = "Hello " + req.Name
    return nil
}

func main() {
    svc := micro.New("helloworld")
    svc.Init()
    svc.Handle(new(Say))
    svc.Run()
}
```

Run it:

```bash
go run main.go
```

Call it:

```bash
curl -XPOST \
  -H 'Content-Type: application/json' \
  -H 'Micro-Endpoint: Say.Hello' \
  -d '{"name": "Alice"}' \
  http://127.0.0.1:8080
```

Set a fixed address:

```go
svc := micro.NewService(
    micro.Name("helloworld"),
    micro.Address(":8080"),
)
```
