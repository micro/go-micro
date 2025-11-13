---
layout: default
---

# RPC Client

Call a running service using the Go Micro client.

```go
package main

import (
    "context"
    "fmt"
    "go-micro.dev/v5"
)

type Request struct { Name string }

type Response struct { Message string }

func main() {
    svc := micro.New("caller")
    svc.Init()

    req := svc.Client().NewRequest("helloworld", "Say.Hello", &Request{Name: "John"})
    var rsp Response

    if err := svc.Client().Call(context.TODO(), req, &rsp); err != nil {
        fmt.Println("error:", err)
        return
    }

    fmt.Println(rsp.Message)
}
```
