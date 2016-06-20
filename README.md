# Go Micro [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-micro?status.svg)](https://godoc.org/github.com/micro/go-micro) [![Travis CI](https://api.travis-ci.org/micro/go-micro.svg?branch=master)](https://travis-ci.org/micro/go-micro) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro)](https://goreportcard.com/report/github.com/micro/go-micro)

Go Micro is a pluggable RPC framework which provides the fundamental building blocks for writing microservices. It is part of [Micro](https://github.com/micro/micro), the microservice toolkit.

The **Micro** philosophy is sane defaults with a pluggable architecture. We provide defaults to get you started quickly but everything can be easily swapped out. It comes with built in support for {json,proto}-rpc encoding, consul or multicast dns for service discovery, http for communication and random hashed client side load balancing.

Everything in go-micro is **pluggable**. You can find and contribute to plugins at [github.com/micro/go-plugins](https://github.com/micro/go-plugins).

An example service can be found in [**examples/service**](https://github.com/micro/go-micro/tree/master/examples/service). The [**examples**](https://github.com/micro/go-micro/tree/master/examples) directory contains many more examples for using things such as middleware/wrappers, selector filters, pub/sub and code generation.

Check out the blog post to learn how to write go-micro services [https://blog.micro.mu/2016/03/28/go-micro.html](https://blog.micro.mu/2016/03/28/go-micro.html).

Join the community to learn more:
- [Mailing List](https://groups.google.com/forum/#!forum/micro-services) 
- [Slack](https://micro-services.slack.com) : [Invite](http://slack.micro.mu/)

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
[geocode-srv](https://github.com/micro/geocode-srv)	|	A geocoding service using the Google Geocoding API
[hailo-srv](https://github.com/micro/hailo-srv)	|	A service for the hailo taxi service developer api
[place-srv](https://github.com/micro/place-srv)	|	A microservice to store and retrieve places (includes Google Place Search API)
[slack-srv](https://github.com/micro/slack-srv)	|	The slack bot API as a go-micro RPC service
[twitter-srv](https://github.com/micro/twitter-srv)	|	A microservice for the twitter API
[user-srv](https://github.com/micro/user-srv)	|	A microservice for user management and authentication

## Go Plugins

By default go-micro only provides a single implementation of each interface. Plugins can be found at [github.com/micro/go-plugins](https://github.com/micro/go-plugins). Contributions welcome!

## How does it work?

<p align="center">
  <img src="go-micro.png" />
</p>

Go Micro is a framework that addresses the fundamental requirements to write microservices. 

Let's dig into the core components.

### Registry

The registry provides a service discovery mechanism to resolve names to addresses. It can be backed by consul, etcd, zookeeper, dns, gossip, etc. 
Services should register using the registry on startup and deregister on shutdown. Services can optionally provide an expiry TTL and reregister 
on an interval to ensure liveness and that the service is cleaned up if it dies.

### Selector

The selector is a load balancing abstraction which builds on the registry. It allows services to be "filtered" using filter functions and "selected" 
using a choice of algorithms such as random, roundrobin, leastconn, etc. The selector is leveraged by the Client when making requests. The client 
will use the selector rather than the registry as it provides that built in mechanism of load balancing. 

### Transport

The transport is the interface for synchronous request/response communication between services. It's akin to the golang net package but provides 
a higher level abstraction which allows us to switch out communication mechanisms e.g http, rabbitmq, websockets, NATS. The transport also 
supports bidirectional streaming. This is powerful for client side push to the server.

### Broker

The broker provides an interface to a message broker for asynchronous pub/sub communication. This is one of the fundamental requirements of an event 
driven architecture and microservices. By default we use an inbox style point to point HTTP system to minimise the number of dependencies required 
to get started. However there are many message broker implementations available in go-plugins e.g RabbitMQ, NATS, NSQ, Google Cloud Pub Sub.

### Codec

The codec is used for encoding and decoding messages before transporting them across the wire. This could be json, protobuf, bson, msgpack, etc. 
Where this differs from most other codecs is that we actually support the RPC format here as well. So we have JSON-RPC, PROTO-RPC, BSON-RPC, etc. 
It separates encoding from the client/server and provides a powerful method for integrating other systems such as gRPC, Vanadium, etc.

### Server

The server is the building block for writing a service. Here you can name your service, register request handlers, add middeware, etc. The service 
builds on the above packages to provide a unified interface for serving requests. The built in server is an RPC system. In the future there maybe 
other implementations. The server also allows you to define multiple codecs to serve different encoded messages.

### Client

The client provides an interface to make requests to services. Again like the server, it builds on the other packages to provide a unified interface 
for finding services by name using the registry, load balancing using the selector, making synchronous requests with the transport and asynchronous 
messaging using the broker. 


The  above components are combined at the top-level of micro as a **Service**.

## Getting Started

This is a quick getting started guide with the greeter service example.

### Prerequisites

There's just one prerequisite. We need a service discovery system to resolve service names to their address. 
The default discovery mechanism used in go-micro is Consul. Discovery is however pluggable so you can used 
etcd, kubernetes, zookeeper, etc. Other implementations can be found in [go-plugins](https://github.com/micro/go-plugins).

Alternatively we can use multicast DNS with the built in MDNS registry for a zero dependency configuration. Just pass `--registry=mdns` to the below commands.

### Install Consul
[https://www.consul.io/intro/getting-started/install.html](https://www.consul.io/intro/getting-started/install.html)

### Run Consul
```
$ consul agent -dev -advertise=127.0.0.1
```

### Run Service
```
$ go run examples/service/main.go
2016/03/14 10:59:14 Listening on [::]:50137
2016/03/14 10:59:14 Broker Listening on [::]:50138
2016/03/14 10:59:14 Registering node: greeter-ca62b017-e9d3-11e5-9bbb-68a86d0d36b6
```

### Test Service
```
$ go run examples/service/main.go --client
Hello John
```

## Writing a service

### Create request/response proto

One of the key requirements of microservices is strongly defined interfaces so we utilised protobuf to define the handler and request/response. 
Here's a definition for the Greeter handler with the method Hello which takes a HelloRequest and HelloResponse both with one string arguments.

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
go get github.com/micro/protobuf/{proto,protoc-gen-go}
```

There's still a need for proto compiler to generate Go stub code from our proto file. You can either use the micro fork above or the official repo `github.com/golang/protobuf`.

### Compile the protobuf file

```
`protoc -I$GOPATH/src --go_out=plugins=micro:$GOPATH/src $GOPATH/src/github.com/micro/go-micro/examples/service/proto/greeter.proto`
```

### Define the service

Below is the code sample for the Greeter service. It basically implements the interface defined above for the Greeter handler, 
initialises the service, registers the handler and then runs itself. Simple as that.

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
go run examples/service/main.go
2016/03/14 10:59:14 Listening on [::]:50137
2016/03/14 10:59:14 Broker Listening on [::]:50138
2016/03/14 10:59:14 Registering node: greeter-ca62b017-e9d3-11e5-9bbb-68a86d0d36b6
```

### Define a client

Below is the client code to query the greeter service. Notice we're using the code generated client interface `proto.NewGreeterClient`. 
This reduces the amount of boiler plate code we need to write. The greeter client can be reused throughout the code if need be.

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

## Sponsors

<a href="https://www.sixt.com"><img src="https://micro.mu/sixt_logo.png" width=150px height="auto" /></a>

## Next steps

- [Examples Directory](https://github.com/micro/go-micro/tree/master/examples)
- [Example Services](https://github.com/micro/go-micro#example-services)
- [Micro Toolkit](https://github.com/micro/micro)
- Join the [Slack](https://micro-services.slack.com)! - [Invite Here](http://micro-invites.herokuapp.com/)


## Contributing

- Checkout the issues list [github.com/micro/go-micro/issues](https://github.com/micro/go-micro/issues)
- Join the Slack to discuss the roadmap
- PR plugins to [github.com/micro/go-plugins](https://github.com/micro/go-plugins)
- Write example services for others to use
