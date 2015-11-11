# Go Micro [![GoDoc](https://godoc.org/github.com/myodc/go-micro?status.svg)](https://godoc.org/github.com/myodc/go-micro) [![Travis CI](https://travis-ci.org/myodc/go-micro.svg?branch=master)](https://travis-ci.org/myodc/go-micro)

Go Micro is a microservices library which provides the fundamental building blocks for writing fault tolerant distributed systems at scale. It is part of the [Micro](https://github.com/myodc/micro) toolchain.

An example server can be found in examples/server.

- [Mailing List](https://groups.google.com/forum/#!forum/micro-services) 
- [Slack](https://micro-services.slack.com) : [auto-invite](http://micro-invites.herokuapp.com/)

## Features

Feature		| Package	|	Description
-------		| -------	|	---------
Discovery	| [Registry](https://godoc.org/github.com/myodc/go-micro/registry)	|	A way of locating services to communicate with
Client		| [Client](https://godoc.org/github.com/myodc/go-micro/client)	|	Used to make RPC requests to a service
Server		| [Server](https://godoc.org/github.com/myodc/go-micro/server)	|	Listens and serves RPC requests
Pub/Sub		| [Broker](https://godoc.org/github.com/myodc/go-micro/broker)	|	Publish and Subscribe to events

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
$ go run examples/server/main.go --logtostderr
I1108 11:08:19.926071   11358 server.go:96] Starting server go.micro.srv.example id go.micro.srv.example-04de4cf0-8609-11e5-bf3a-68a86d0d36b6
I1108 11:08:19.926407   11358 rpc_server.go:233] Listening on [::]:54080
I1108 11:08:19.926500   11358 http_broker.go:80] Broker Listening on [::]:54081
I1108 11:08:19.926632   11358 rpc_server.go:158] Registering node: go.micro.srv.example-04de4cf0-8609-11e5-bf3a-68a86d0d36b6
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

Compile proto `protoc -I$GOPATH/src --go_out=$GOPATH/src $GOPATH/src/github.com/myodc/go-micro/examples/server/proto/example/example.proto`

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
	log.Infof("Received Example.Call request with metadata: %v", md)
	rsp.Msg = server.Config().Id() + ": Hello " + req.Name
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
	server.Handle(
		server.NewHandler(
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
$ go run examples/server/main.go --logtostderr
I1108 11:08:19.926071   11358 server.go:96] Starting server go.micro.srv.example id go.micro.srv.example-04de4cf0-8609-11e5-bf3a-68a86d0d36b6
I1108 11:08:19.926407   11358 rpc_server.go:233] Listening on [::]:54080
I1108 11:08:19.926500   11358 http_broker.go:80] Broker Listening on [::]:54081
I1108 11:08:19.926632   11358 rpc_server.go:158] Registering node: go.micro.srv.example-04de4cf0-8609-11e5-bf3a-68a86d0d36b6
```
