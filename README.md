# Go Micro [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-micro?status.svg)](https://godoc.org/github.com/micro/go-micro) [![Travis CI](https://api.travis-ci.org/micro/go-micro.svg?branch=master)](https://travis-ci.org/micro/go-micro) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro)](https://goreportcard.com/report/github.com/micro/go-micro)

Go Micro is a pluggable RPC framework for **microservices**. It is part of the [Micro](https://github.com/micro/micro) toolkit.

The **Micro** philosophy is sane defaults with a pluggable architecture. We provide defaults to get you started quickly but everything can be easily swapped out. It comes with built in support for {json,proto}-rpc encoding, consul or multicast dns for service discovery, http for communication and random hashed client side load balancing.

Everything in go-micro is **pluggable**. You can find and contribute to plugins at [github.com/micro/go-plugins](https://github.com/micro/go-plugins).

Follow us on [Twitter](https://twitter.com/microhq) or join the [Slack](http://slack.micro.mu/) community.

## Features

Go Micro abstracts away the details of distributed systems. Here are the main features.

- **Service Discovery** - Applications are automatically registered with service discovery so they can find each other.
- **Load Balancing** - Smart client side load balancing is used to balance requests between instances of a service.
- **Synchronous Communication** - Request-response is provided as a bidirectional streaming transport layer.
- **Asynchronous Communication** - Microservices should promote an event driven architecture. Publish and Subscribe semantics are built in.
- **Message Encoding** - Micro services can encode requests in a number of encoding formats and seamlessly decode based on the Content-Type header.
- **RPC Client/Server** - The client and server leverage the above features and provide a clean simple interface for building microservices.

Go Micro supports both the Service and Function programming models. Read on to learn more.

## Docs

For more detailed information on the architecture, installation and use of go-micro checkout the [docs](https://micro.mu/docs).

## Learn By Example

An example service can be found in [**examples/service**](https://github.com/micro/examples/tree/master/service) and function in [**examples/function**](https://github.com/micro/examples/tree/master/function). The [**examples**](https://github.com/micro/examples) directory contains many more examples for using things such as middleware/wrappers, selector filters, pub/sub and code generation. 
For the complete greeter example look at [**examples/greeter**](https://github.com/micro/examples/tree/master/greeter). Other examples can be found throughout the GitHub repository.

Check out the blog post to learn how to write go-micro services [https://micro.mu/blog/2016/03/28/go-micro.html](https://micro.mu/blog/2016/03/28/go-micro.html) or watch the talk from the [Golang UK Conf 2016](https://www.youtube.com/watch?v=xspaDovwk34).

## Getting Started

This is a quick getting started guide with the greeter service example.

### Prerequisites: Service Discovery

There's just one prerequisite. We need a service discovery system to resolve service names to their address. 
The default discovery mechanism used in go-micro is Consul. Discovery is however pluggable so you can used 
etcd, kubernetes, zookeeper, etc. Plugins can be found in [micro/go-plugins](https://github.com/micro/go-plugins).

### Multicast DNS

We can use multicast DNS with the built in MDNS registry for a zero dependency configuration. 

Just pass `--registry=mdns` to any command
```
$ go run main.go --registry=mdns
```

### Consul

Alternatively we can use the default discovery system which is Consul.

**Mac OS**
```
brew install consul
consul agent -dev
```

**Docker**
```
docker run consul
```

[Further installation instructions](https://www.consul.io/intro/getting-started/install.html)

### Run Service

```
$ go get github.com/micro/examples/service && service
2016/03/14 10:59:14 Listening on [::]:50137
2016/03/14 10:59:14 Broker Listening on [::]:50138
2016/03/14 10:59:14 Registering node: greeter-ca62b017-e9d3-11e5-9bbb-68a86d0d36b6
```

### Call Service
```
$ service --run_client
Hello John
```

## Writing a service

### Create service proto

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

### Install protobuf

We use a protobuf plugin for code generation. This is completely optional. Look at [examples/server](https://github.com/micro/examples/blob/master/server/main.go) 
and [examples/client](https://github.com/micro/examples/blob/master/client/main.go) for examples without code generation.

```shell
go get github.com/micro/protobuf/{proto,protoc-gen-go}
```

There's still a need for proto compiler to generate Go stub code from our proto file. You can either use the micro fork above or the official repo `github.com/golang/protobuf`.

### Compile the proto

```shell
protoc -I$GOPATH/src --go_out=plugins=micro:$GOPATH/src \
	$GOPATH/src/github.com/micro/examples/service/proto/greeter.proto
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
	proto "github.com/micro/examples/service/proto"
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
	)

	// Init will parse the command line flags.
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
	proto "github.com/micro/examples/service/proto"
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

## Writing a Function

Go Micro includes the Function programming model. This is the notion of a one time executing Service which operates much like a service except exiting 
after completing a request. A function is defined much like a service and called in exactly the same way.

### Defining a Function

```go
package main

import (
	proto "github.com/micro/examples/function/proto"
	"github.com/micro/go-micro"
	"golang.org/x/net/context"
)

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	// create a new function
	fnc := micro.NewFunction(
		micro.Name("go.micro.fnc.greeter"),
	)

	// init the command line
	fnc.Init()

	// register a handler
	fnc.Handle(new(Greeter))

	// run the function
	fnc.Run()
}
```

It's that simple.

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

## Plugins

By default go-micro only provides a few implementation of each interface at the core but it's completely pluggable. There's already dozens of plugins which are available at [github.com/micro/go-plugins](https://github.com/micro/go-plugins). Contributions are welcome!

### Build with plugins

If you want to integrate plugins simply link them in a separate file and rebuild

Create a plugins.go file

```go
import (
        // etcd v3 registry
        _ "github.com/micro/go-plugins/registry/etcdv3"
        // nats transport
        _ "github.com/micro/go-plugins/transport/nats"
        // kafka broker
        _ "github.com/micro/go-plugins/broker/kafka"
)
```

Build binary

```shell
// For local use
go build -i -o service ./main.go ./plugins.go
```

Flag usage of plugins
```shell
service --registry=etcdv3 --transport=nats --broker=kafka
```

## Other Languages

Check out [ja-micro](https://github.com/Sixt/ja-micro) to write services in Java

## Sponsors

Open source development of Micro is sponsored by Sixt

<a href="https://micro.mu/blog/2016/04/25/announcing-sixt-sponsorship.html"><img src="https://micro.mu/sixt_logo.png" width=150px height="auto" /></a>
