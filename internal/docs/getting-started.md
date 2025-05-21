# Getting Started

To make use of Go Micro 

```golang
go get "go-micro.dev/v5"
```

## Create a service

This is a basic example of how you'd create a service and register a handler in pure Go.

```
mkdir helloworld
cd helloworld
go mod init
go get go-micro.dev/v5@latest"
```

Write the following into `main.go`

```golang
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

```
go run main.go
```

Take a note of the address with the log line

```
Transport [http] Listening on [::]:35823
```

Now you can call the service

```
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:35823
```

## Set a fixed address

To set a fixed address by specifying it as an option to service, note the change from `New` to `NewService`

```golang
service := micro.NewService(
    micro.Name("helloworld"),
    micro.Address(":8080"),
)
```

Alternatively use `MICRO_SERVER_ADDRESS=:8080` as an env var

```
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:8080
```

## Protobuf

If you want to define services with protobuf you can use [protoc-gen-micro](https://github.com/micro/micro/tree/master/cmd/protoc-gen-micro)

```
cd helloworld
mkdir proto
```

Edit a file `proto/helloworld.proto`

```
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

You can now generate a client/server like so

```
protoc --proto_path=. --micro_out=. --go_out=. helloworld.proto
```

In your `main.go` update the code to reference the generated code

```
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
        proto.RegisterSayHandler(service.Server(), &Say{})

        // run the service
        service.Run()
}
```

Now I can run this again

```
go run main.go
```

## Call via a client

The generated code provides us a client

```
package main

import (
        "context"
        "fmt"

        "go-micro.dev/v5"
        pb "github.com/micro/helloworld/proto"
)

package main() {
        service := micro.New("helloworld)
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

If you'd like to use a command line checkout [Micro](https://github.com/micro/micro)

```
go get github.com/micro/micro/v5@latest
```

```
micro call helloworld Say.Hello '{"name": "John"}'
```

Alternative using the dynamic CLI commands

```
micro helloworld say hello --name="John"
```
