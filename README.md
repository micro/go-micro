# Go Micro

Go Micro is a microservices client/server library utilising HTTP/RPC and protobuf. It is part of the [Micro](https://github.com/myodc/micro) toolchain.

An example server can be found in go-micro/template.

[![GoDoc](http://img.shields.io/badge/go-documentation-brightgreen.svg?style=flat-square)](https://godoc.org/github.com/myodc/go-micro)

## Features
- Discovery
- Client/Server
- Pub/Sub
- Key/Value store

### Planned
- Metrics
- Tracing
- Logging
- Stats

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
$ go run go-micro/template/main.go

1416690099281057746 [Debug] Rpc handler /_rpc
1416690099281092588 [Debug] Starting server go.micro.service.template id go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
1416690099281192941 [Debug] Listening on [::]:58264
1416690099281215346 [Debug] Registering go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
```

### Test Service
```
$ go run go-micro/examples/service_client.go

go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6: Hello John
```

## Writing a service

### Create request/response proto
`go-micro/template/proto/example/example.proto`:

```
package go.micro.service.template.example;

message Request {
	required string name = 1;
}

message Response {
	required string msg = 1;
}
```

Compile proto `protoc -I$GOPATH/src --go_out=$GOPATH/src $GOPATH/src/github.com/myodc/go-micro/template/proto/example/example.proto`

### Create request handler
`go-micro/template/handler/example.go`:

```go
package handler

import (
	"code.google.com/p/go.net/context"
	"github.com/golang/protobuf/proto"

	"github.com/myodc/go-micro/server"
	example "github.com/myodc/go-micro/template/proto/example"
	log "github.com/golang/glog"
)

type Example struct{}

func (e *Example) Call(ctx context.Context, req *example.Request, rsp *example.Response) error {
	log.Info("Received Example.Call request")

	rsp.Msg = proto.String(server.Id + ": Hello " + req.GetName())

	return nil
}
```

### Init server
`go-micro/template/main.go`:

```go
package main

import (
	"github.com/myodc/go-micro/server"
	"github.com/myodc/go-micro/template/handler"
	log "github.com/golang/glog"
)

func main() {
	server.Name = "go.micro.service.template"

	// Initialise Server
	server.Init()

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
$ go run go-micro/template/main.go

1416690099281057746 [Debug] Rpc handler /_rpc
1416690099281092588 [Debug] Starting server go.micro.service.template id go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
1416690099281192941 [Debug] Listening on [::]:58264
1416690099281215346 [Debug] Registering go.micro.service.template-c0bfcb44-728a-11e4-b099-68a86d0d36b6
```
