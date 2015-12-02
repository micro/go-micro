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

greeter.proto
```shell
syntax = "proto3";

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
	c client.Client
}

func NewSayClient(c client.Client) SayClient {
	if c == nil {
		c = client.NewClient()
	}
	return &sayClient{
		c: c,
	}
}

func (c *sayClient) Hello(ctx context.Context, in *Request) (*Response, error) {
	req := c.c.NewRequest("go.micro.srv.greeter", "Say.Hello", in)
	out := new(Response)
	err := c.c.Call(ctx, req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Say service

type SayServer interface {
	Hello(context.Context, *Request, *Response) error
}

func RegisterSayServer(s server.Server, srv SayServer) {
	s.Handle(s.NewHandler(srv))
}
```

### Use the client
```golang

import (
	"fmt"

	"golang.org/x/net/context"
	"github.com/micro/go-micro/client"
	hello "path/to/hello/proto"
)

func main() {
	cl := hello.NewSayClient(client.DefaultClient)

	rsp, err := cl.Hello(contex.Background(), &hello.Request{"Name": "John"})
	if err != nil {
		fmt.Println(err)
	}
}
```
