# GRPC Client

The grpc client is a [micro.Client](https://godoc.org/github.com/micro/go-micro/client#Client) compatible client.

## Overview

The client makes use of the [google.golang.org/grpc](google.golang.org/grpc) framework for the underlying communication mechanism.

## Usage

Specify the client to your micro service

```go
import (
	"github.com/micro/go-micro"
	"github.com/asim/go-micro/plugins/server/grpc/v4"
)

func main() {
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Client(grpc.NewClient()),
	)
}
```
