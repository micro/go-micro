# Go Micro [![GoDoc](https://godoc.org/github.com/micro/go-micro?status.svg)](https://godoc.org/github.com/micro/go-micro) [![Travis CI](https://travis-ci.org/micro/go-micro.svg?branch=master)](https://travis-ci.org/micro/go-micro)

Go Micro is a pluggable RPC based microservice library which provides the fundamental building blocks for writing distributed applications. It is part of the [Micro](https://github.com/micro/micro) toolchain. It supports Proto-RPC and JSON-RPC as the request/response protocol out of the box and defaults to Consul for discovery.

Every aspect of go-micro is pluggable.

An example service can be found in [**examples/service**](https://github.com/micro/go-micro/tree/master/examples/service). The [**examples**](https://github.com/micro/go-micro/tree/master/examples) directory contains many more examples for using things such as middleware/wrappers, selector filters, pub/sub and code generation.

- [Mailing List](https://groups.google.com/forum/#!forum/micro-services) 
- [Slack](https://micro-services.slack.com) : [auto-invite](http://micro-invites.herokuapp.com/)

## Features

Feature		| Package	|	Built-in Plugin		|	Description
-------		| -------	|	---------		|	-----------
Discovery	| [Registry](https://godoc.org/github.com/micro/go-micro/registry)	| consul	| A way of locating services to communicate with
Client		| [Client](https://godoc.org/github.com/micro/go-micro/client)	| rpc	| Used to make RPC requests to a service
Codec		| [Codec](https://godoc.org/github.com/micro/go-micro/codec)	| proto,json	| Encoding/Decoding handler for requests
Balancer	| [Selector](https://godoc.org/github.com/micro/go-micro/selector)	| random	| Service node filter and pool 
Server		| [Server](https://godoc.org/github.com/micro/go-micro/server)	| rpc	| Listens and serves RPC requests
Pub/Sub		| [Broker](https://godoc.org/github.com/micro/go-micro/broker)	| http	| Publish and Subscribe to events
Transport	| [Transport](https://godoc.org/github.com/micro/go-micro/transport)	| http	| Communication mechanism between services

## Example Services
Project		|	Description
-----		|	------
[greeter](https://github.com/micro/micro/tree/master/examples/greeter)	|	A greeter service (includes Go, Ruby, Python examples)
[geo-srv](https://github.com/micro/geo-srv)	|	Geolocation tracking service using hailocab/go-geoindex
[geo-api](https://github.com/micro/geo-api)	|	A HTTP API handler for geo location tracking and search
[discovery-srv](https://github.com/micro/discovery-srv)	|	A discovery in the micro platform
[geocode-srv](https://github.com/micro/geocode-srv)	|	A geocoding service using the Google Geocoding API
[hailo-srv](https://github.com/micro/hailo-srv)	|	A service for the hailo taxi service developer api
[monitoring-srv](https://github.com/micro/monitoring-srv)	|	A monitoring service for Micro services
[place-srv](https://github.com/micro/place-srv)	|	A microservice to store and retrieve places (includes Google Place Search API)
[slack-srv](https://github.com/micro/slack-srv)	|	The slack bot API as a go-micro RPC service
[trace-srv](https://github.com/micro/trace-srv)	|	A distributed tracing microservice in the realm of dapper, zipkin, etc
[twitter-srv](https://github.com/micro/twitter-srv)	|	A microservice for the twitter API
[user-srv](https://github.com/micro/user-srv)	|	A microservice for user management and authentication

## Go Plugins

By default go-micro only provides a single implementation of each interface. Plugins can be found at [github.com/micro/go-plugins](https://github.com/micro/go-plugins). Contributions welcome!

## Prerequisites

Consul is the default discovery mechanism provided in go-micro. Discovery is however pluggable so you can used etcd, kubernetes, zookeeper, etc.

### Install Consul
[https://www.consul.io/intro/getting-started/install.html](https://www.consul.io/intro/getting-started/install.html)

## Getting Started

### Run Consul
```
$ consul agent -server -bootstrap-expect 1 -data-dir /tmp/consul
```

### Run Service
```
$ go run examples/service/main.go --logtostderr
I0102 00:22:26.413467   12018 rpc_server.go:297] Listening on [::]:62492
I0102 00:22:26.413803   12018 http_broker.go:115] Broker Listening on [::]:62493
I0102 00:22:26.414009   12018 rpc_server.go:212] Registering node: greeter-e6b2fc6f-b0e6-11e5-a42f-68a86d0d36b6
```

### Test Service
```
$ go run examples/service/main.go --client
Hello John
```

## Writing a service

### Create request/response proto
`go-micro/examples/service/proto/greeter.proto`:

```proto
syntax = "proto3";

service Greeter {
	rpc Hello(HelloRequest) returns (HelloResponse) {}
}

message HelloRequest {
	string name = 1;
}

message HelloResponse {
	string greeting = 2;
}
```

### Install protobuf for code generation

We use a protobuf plugin for code generation. This is completely optional. Look at [examples/server](https://github.com/micro/go-micro/blob/master/examples/server/main.go) 
and [examples/client](https://github.com/micro/go-micro/blob/master/examples/client/main.go) for examples without code generation.

```shell
go get github.com/micro/protobuf
```

Compile proto `protoc -I$GOPATH/src --go_out=plugins=micro:$GOPATH/src $GOPATH/src/github.com/micro/go-micro/examples/service/proto/greeter.proto`

### Define the service
`go-micro/examples/service/main.go`:

```go
package main

import (
	"fmt"

	micro "github.com/micro/go-micro"
	proto "github.com/micro/go-micro/examples/service/proto"
	"golang.org/x/net/context"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	// Create a new service. Optionally include some options here.
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Version("latest"),
		micro.Metadata(map[string]string{
			"type": "helloworld",
		}),
	)

	// Init will parse the command line flags. Any flags set will
	// override the above settings. Options defined here will
	// override anything set on the command line.
	service.Init()

	// Register handler
	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
```

### Run service
```
go run examples/service/main.go --logtostderr
I0102 00:22:26.413467   12018 rpc_server.go:297] Listening on [::]:62492
I0102 00:22:26.413803   12018 http_broker.go:115] Broker Listening on [::]:62493
I0102 00:22:26.414009   12018 rpc_server.go:212] Registering node: greeter-e6b2fc6f-b0e6-11e5-a42f-68a86d0d36b6
```

### Define a client

`client.go`

```go
package main

import (
	"fmt"

	micro "github.com/micro/go-micro"
	proto "github.com/micro/go-micro/examples/service/proto"
	"golang.org/x/net/context"
)


func main() {
	// Create a new service. Optionally include some options here.
	service := micro.NewService(micro.Name("greeter.client"))

	// Create new greeter client
	greeter := proto.NewGreeterClient("greeter", service.Client())

	// Call the greeter
	rsp, err := greeter.Hello(context.TODO(), &proto.HelloRequest{Name: "John"})
	if err != nil {
		fmt.Println(err)
	}

	// Print response
	fmt.Println(rsp.Greeting)
}
```

### Run the client

```shell
go run client.go
Hello John
```
