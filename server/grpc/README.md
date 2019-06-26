# GRPC Server

The grpc server is a [micro.Server](https://godoc.org/github.com/micro/go-micro/server#Server) compatible server.

## Overview

The server makes use of the [google.golang.org/grpc](google.golang.org/grpc) framework for the underlying server 
but continues to use micro handler signatures and protoc-gen-micro generated code.

## Usage

Specify the server to your micro service

```go
import (
        "github.com/micro/go-micro"
        "github.com/micro/go-plugins/server/grpc"
)

func main() {
        service := micro.NewService(
                // This needs to be first as it replaces the underlying server
                // which causes any configuration set before it
                // to be discarded
                micro.Server(grpc.NewServer()),
                micro.Name("greeter"),
        )
}
```
**NOTE**: Setting the gRPC server and/or client causes the underlying the server/client to be replaced which causes any previous configuration set on that server/client to be discarded. It is therefore recommended to set gRPC server/client before any other configuration