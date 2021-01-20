# Code Generation [Experimental]

We're experimenting with code generation to reduce the amount of boiler plate code written.

## Example

Going from this
```golang
req := client.NewRequest("go.micro.srv.example", "Example.Call", &example.Request{
	Name: "John",
})

rsp := &example.Response{}

if err := client.Call(context.Background(), req, rsp); err != nil {
	return err
}
```

To

```golang
rsp, err := cl.Call(context.Background(), &example.Request{Name: "John"})
if err != nil {
	return err
}
```

## Generation of stub code for the example service

```shell
go get github.com/micro/protobuf/protoc-gen-go
cd examples/server/proto/example
protoc --go_out=plugins=micro:. example.proto
```

Look at examples/server/proto/example/example.pb.go 
to see the generated code.

## Guide

### Download the protoc-gen-go code

```shell
go get github.com/micro/protobuf/protoc-gen-go
```

### Define your proto service.

hello.proto
```shell
syntax = "proto3";

// package name is used as the service name for discovery
// if service name is not passed in when initialising the 
// client
package go.micro.srv.greeter;

service Say {
	rpc Hello(Request) returns (Response) {}
}

message Request {
	optional string name = 1;
}

message Response {
	optional string msg = 1;
}
```

**Note: Remember to set package name in the proto, it's used to generate 
the service for discovery.**

### Generate code

```shell
protoc --go_out=plugins=micro:. hello.proto
```

### Generated code

```shell
// Client API for Say service

type SayClient interface {
        Hello(ctx context.Context, in *Request) (*Response, error)
}

type sayClient struct {
        c           client.Client
        serviceName string
}

func NewSayClient(serviceName string, c client.Client) SayClient {
        if c == nil {
                c = client.NewClient()
        }
        if len(serviceName) == 0 {
                serviceName = "go.micro.srv.greeter"
        }
        return &sayClient{
                c:           c,
                serviceName: serviceName,
        }
}

func (c *sayClient) Hello(ctx context.Context, in *Request) (*Response, error) {
        req := c.c.NewRequest(c.serviceName, "Say.Hello", in)
        out := new(Response)
        err := c.c.Call(ctx, req, out)
        if err != nil {
                return nil, err
        }
        return out, nil
}

// Server API for Say service

type SayHandler interface {
        Hello(context.Context, *Request, *Response) error
}

func RegisterSayHandler(s server.Server, hdlr SayHandler) {
        s.Handle(s.NewHandler(hdlr))
}
```

### Use the client
```golang

import (
	"fmt"

	"context"
	"github.com/asim/go-micro/v3/client"
	hello "path/to/hello/proto"
)

func main() {
	cl := hello.NewSayClient("go.micro.srv.greeter", client.DefaultClient)
	// alternative initialisation
	// cl := hello.NewSayClient("", nil)

	rsp, err := cl.Hello(contex.Background(), &hello.Request{"Name": "John"})
	if err != nil {
		fmt.Println(err)
	}
}
```
