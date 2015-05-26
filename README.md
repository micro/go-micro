# Go Micro

Go Micro is a microservices library which provides the fundamental building blocks for writing fault tolerant distributed systems at scale. It is part of the [Micro](https://github.com/myodc/micro) toolchain.

An example server can be found in examples/server.

[![GoDoc](http://img.shields.io/badge/go-documentation-brightgreen.svg?style=flat-square)](https://godoc.org/github.com/myodc/go-micro)

## Features
- Discovery
- Client
- Server
- Pub/Sub

## Prerequisites

Consul is the default discovery mechanism provided in go-micro. Discovery is however pluggable.

### Install Consul
[https://www.consul.io/intro/getting-started/install.html](https://www.consul.io/intro/getting-started/install.html)

## Getting Started

### Run Consul
```
$ consul agent -server -bootstrap-expect 1 -data-dir /tmp/consul
```

### Run Service
```
$ go run examples/server/main.go
I0525 18:06:14.471489   83304 server.go:117] Starting server go.micro.srv.example id go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
I0525 18:06:14.474960   83304 rpc_server.go:126] Listening on [::]:62216
I0525 18:06:14.474997   83304 server.go:99] Registering node: go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
```

### Test Service
```
$ go run examples/client/main.go 
go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6: Hello John
```

## Writing a service

### Create request/response proto
`go-micro/examples/server/proto/example/example.proto`:

```
syntax = "proto3";

message Request {
        string name = 1;
}

message Response {
        string msg = 1;
}
```

Compile proto `protoc -I$GOPATH/src --go_out=$GOPATH/src $GOPATH/src/github.com/myodc/go-micro/template/proto/example/example.proto`

### Create request handler
`go-micro/examples/server/handler/example.go`:

```go
package handler

import (
        log "github.com/golang/glog"
        c "github.com/myodc/go-micro/context"
        example "github.com/myodc/go-micro/examples/server/proto/example"
        "github.com/myodc/go-micro/server"

        "golang.org/x/net/context"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
        md, _ := c.GetMetadata(ctx)
        log.Info("Received Example.Call request with metadata: %v", md)
        rsp.Msg = server.Id + ": Hello " + req.Name
        return nil
}
```

### Init server
`go-micro/examples/server/main.go`:

```go
package main

import (
	log "github.com/golang/glog"
	"github.com/myodc/go-micro/cmd"
	"github.com/myodc/go-micro/examples/server/handler"
	"github.com/myodc/go-micro/server"
)

func main() {
	// optionally setup command line usage
	cmd.Init()

	// Initialise Server
	server.Init(
		server.Name("go.micro.srv.example"),
	)

	// Register Handlers
	server.Register(
		server.NewReceiver(
			new(handler.Example),
		),
	)

	// Run server
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Run service
```
$ go run examples/server/main.go
I0525 18:06:14.471489   83304 server.go:117] Starting server go.micro.srv.example id go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
I0525 18:06:14.474960   83304 rpc_server.go:126] Listening on [::]:62216
I0525 18:06:14.474997   83304 server.go:99] Registering node: go.micro.srv.example-59b6e0ab-0300-11e5-b696-68a86d0d36b6
```
