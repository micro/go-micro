---
layout: default
---

# Getting Started

To make use of Go Micro 

```bash
go get go-micro.dev/v5@latest
```

## Create a service

This is a basic example of how you'd create a service and register a handler in pure Go.

```bash
mkdir helloworld
cd helloworld
go mod init
go get go-micro.dev/v5@latest
```

Write the following into `main.go`

```go
package main

import (
        "go-micro.dev/v5"
)

type Request struct {
        Name string `json:"name"`
}

type Response struct {
        Message string `json:"message"`
}

type Say struct{}

func (h *Say) Hello(ctx context.Context, req *Request, rsp *Response) error {
        rsp.Message = "Hello " + req.Name
        return nil
}

func main() {
        // create the service
        service := micro.New("helloworld")

        // initialise service
        service.Init()

        // register handler
        service.Handle(new(Say))

        // run the service
        service.Run()
}
```

Now run the service

```bash
go run main.go
```

Take a note of the address with the log line

```text
Transport [http] Listening on [::]:35823
```

Now you can call the service

```bash
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:35823
```

## Set a fixed address

To set a fixed address by specifying it as an option to service, note the change from `New` to `NewService`

```go
service := micro.NewService(
    micro.Name("helloworld"),
    micro.Address(":8080"),
)
```

Alternatively use `MICRO_SERVER_ADDRESS=:8080` as an env var

```bash
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:8080
```

## Protobuf

If you want to define services with protobuf you can use protoc-gen-micro (go-micro.dev/v5/cmd/protoc-gen-micro).

Install the generator:

```bash
go install go-micro.dev/v5/cmd/protoc-gen-micro@latest
```

```bash
cd helloworld
mkdir proto
```

Edit a file `proto/helloworld.proto`

```proto
syntax = "proto3";

package greeter;
option go_package = "/proto;helloworld";

service Say {
        rpc Hello(Request) returns (Response) {}
}

message Request {
        string name = 1;
}

message Response {
        string message = 1;
}
```

You can now generate a client/server like so (ensure `$GOBIN` is on your `$PATH` so `protoc` can find `protoc-gen-micro`):

```bash
protoc --proto_path=. --micro_out=. --go_out=. helloworld.proto
```

In your `main.go` update the code to reference the generated code

```go
package main

import (
        "go-micro.dev/v5"

        pb "github.com/micro/helloworld/proto"
)

type Say struct{}

func (h *Say) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
        rsp.Message = "Hello " + req.Name
        return nil
}

func main() {
        // create the service
        service := micro.New("helloworld")

        // initialise service
        service.Init()

        // register handler
        pb.RegisterSayHandler(service.Server(), &Say{})

        // run the service
        service.Run()
}
```

Now I can run this again

```bash
go run main.go
```

## Call via a client

The generated code provides us a client

```go
package main

import (
        "context"
        "fmt"

        "go-micro.dev/v5"
        pb "github.com/micro/helloworld/proto"
)

func main() {
        service := micro.New("helloworld")
        service.Init()

        say := pb.NewSayService("helloworld", service.Client())

        rsp, err := say.Hello(context.TODO(), &pb.Request{
            Name: "John",
        })
        if err != nil {
                fmt.Println(err)
                return
        }

        fmt.Println(rsp.Message)
}
```

## Command line

Install the Micro CLI:

```
go install go-micro.dev/v5/cmd/micro@latest
```

Call a running service via RPC:

```
micro call helloworld Say.Hello '{"name": "John"}'
```

Alternative using the dynamic CLI commands:

```
micro helloworld say hello --name="John"
```
