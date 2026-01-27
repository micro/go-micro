# Go Micro [![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/go-micro.dev/v5?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-micro/go-micro)](https://goreportcard.com/report/github.com/go-micro/go-micro) 

Go Micro is a framework for distributed systems development.

**[📖 Documentation](https://go-micro.dev/docs/)** | [Sponsor the project](https://github.com/sponsors/micro)

## Overview

Go Micro provides the core requirements for distributed systems development including RPC and Event driven communication.
The Go Micro philosophy is sane defaults with a pluggable architecture. We provide defaults to get you started quickly
but everything can be easily swapped out.

## Features

Go Micro abstracts away the details of distributed systems. Here are the main features.

- **Authentication** - Auth is built in as a first class citizen. Authentication and authorization enable secure
  zero trust networking by providing every service an identity and certificates. This additionally includes rule
  based access control.

- **Dynamic Config** - Load and hot reload dynamic config from anywhere. The config interface provides a way to load application
  level config from any source such as env vars, file, etcd. You can merge the sources and even define fallbacks.

- **Data Storage** - A simple data store interface to read, write and delete records. It includes support for many storage backends
in the plugins repo. State and persistence becomes a core requirement beyond prototyping and Micro looks to build that into the framework.

- **Service Discovery** - Automatic service registration and name resolution. Service discovery is at the core of micro service
  development. When service A needs to speak to service B it needs the location of that service. The default discovery mechanism is
  multicast DNS (mdns), a zeroconf system.

- **Load Balancing** - Client side load balancing built on service discovery. Once we have the addresses of any number of instances
  of a service we now need a way to decide which node to route to. We use random hashed load balancing to provide even distribution
  across the services and retry a different node if there's a problem.

- **Message Encoding** - Dynamic message encoding based on content-type. The client and server will use codecs along with content-type
  to seamlessly encode and decode Go types for you. Any variety of messages could be encoded and sent from different clients. The client
  and server handle this by default. This includes protobuf and json by default.

- **RPC Client/Server** - RPC based request/response with support for bidirectional streaming. We provide an abstraction for synchronous
  communication. A request made to a service will be automatically resolved, load balanced, dialled and streamed.

- **Async Messaging** - PubSub is built in as a first class citizen for asynchronous communication and event driven architectures.
  Event notifications are a core pattern in micro service development. The default messaging system is a HTTP event message broker.

- **Pluggable Interfaces** - Go Micro makes use of Go interfaces for each distributed system abstraction. Because of this these interfaces
  are pluggable and allows Go Micro to be runtime agnostic. You can plugin any underlying technology.

## Getting Started

To make use of Go Micro 

```bash
go get go-micro.dev/v5@latest
```

Create a service and register a handler

```go
package main

import (
        "go-micro.dev/v5"
)

type Request struct {
        Name string `json:"name"`
}

type Response struct {
        Message string `json:"message"`
}

type Say struct{}

func (h *Say) Hello(ctx context.Context, req *Request, rsp *Response) error {
        rsp.Message = "Hello " + req.Name
        return nil
}

func main() {
        // create the service
        service := micro.New("helloworld")

        // register handler
        service.Handle(new(Say))

        // run the service
        service.Run()
}
```

Set a fixed address

```go
service := micro.NewService(
    micro.Name("helloworld"),
    micro.Address(":8080"),
)
```

Call it via curl

```bash
curl -XPOST \
     -H 'Content-Type: application/json' \
     -H 'Micro-Endpoint: Say.Hello' \
     -d '{"name": "alice"}' \
      http://localhost:8080
```

## Experimental

There's a new `genai` package for generative AI capabilities.

## Protobuf

Install the code generator and see usage in the docs:

```bash
go install go-micro.dev/v5/cmd/protoc-gen-micro@latest
```

Docs: [`internal/website/docs/getting-started.md`](internal/website/docs/getting-started.md)

## Command Line

Install the CLI:

```
go install go-micro.dev/v5/cmd/micro@latest
```

### Quick Start

```bash
micro new helloworld   # Create a new service
cd helloworld
micro run              # Run with API gateway
```

Then open http://localhost:8080 to see your service and call it from the browser.

### micro run

`micro run` starts your services with:
- **API Gateway** - HTTP to RPC proxy at `/api/{service}/{method}`
- **Web Dashboard** - Browse and call services at `/`
- **Health Checks** - Aggregated health at `/health`
- **Hot Reload** - Auto-rebuild on file changes

```bash
micro run                    # Gateway on :8080
micro run --address :3000    # Custom gateway port
micro run --no-gateway       # Services only
micro run --env production   # Use production environment
```

### Configuration

For multi-service projects, create a `micro.mu` file:

```
service users
    path ./users
    port 8081

service posts
    path ./posts
    port 8082
    depends users

env development
    DATABASE_URL sqlite://./dev.db
```

The gateway runs on :8080 by default, so services should use other ports.

### Deployment

Deploy to any Linux server with systemd:

```bash
# On your server (one-time setup)
curl -fsSL https://go-micro.dev/install.sh | sh
sudo micro init --server

# From your laptop
micro deploy user@your-server
```

The deploy command:
1. Builds binaries for Linux
2. Copies via SSH to the server
3. Sets up systemd services
4. Verifies services are healthy

Manage deployed services:
```bash
micro status --remote user@server    # Check status
micro logs --remote user@server      # View logs
micro logs myservice --remote user@server -f  # Follow specific service
```

No Docker required. No Kubernetes. Just systemd.

See [docs/deployment.md](docs/deployment.md) for full deployment guide.

See [cmd/micro/README.md](cmd/micro/README.md) for full CLI documentation.

Docs: [`internal/website/docs`](internal/website/docs)

Package reference: https://pkg.go.dev/go-micro.dev/v5

Selected topics:
- Getting Started: [`internal/website/docs/getting-started.md`](internal/website/docs/getting-started.md)
- Plugins overview: [`internal/website/docs/plugins.md`](internal/website/docs/plugins.md)
- Learn by Example: [`internal/website/docs/examples/index.md`](internal/website/docs/examples/index.md)

## Adopters

- [Sourse](https://sourse.eu) - Work in the field of earth observation, including embedded Kubernetes running onboard aircraft, and we’ve built a mission management SaaS platform using Go Micro.
