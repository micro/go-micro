# Nitro [![License](https://img.shields.io/badge/license-polyform:noncommercial-blue)](https://polyformproject.org/licenses/noncommercial/1.0.0/) [![Docs](https://img.shields.io/badge/godoc-reference-green)](https://go-nitro.dev/docs/v3) [![Micro](https://img.shields.io/badge/m3o-micro-red)](https://m3o.com) 

<img src="https://avatars2.githubusercontent.com/u/73709577" />

Nitro (formerly known as Go Micro) is a blazingly fast framework for distributed app development.

Go to [m3o.com](https://m3o.com) for Micro.

## Overview

Nitro provides the core requirements for distributed app development including RPC and Event driven communication. 
The **Nitro** model is in-memory defaults with a pluggable architecture. Blaze with pure in-memory development 
and swap out as needed to go multi-process or multi-host.

## Features

Nitro abstracts away the details of distributed systems. Here are the main features.

- **Authentication** - Auth is built in as a first class citizen. Authentication and authorization enable secure 
zero trust networking by providing every service an identity and certificates. This additionally includes rule 
based access control.

- **Dynamic Config** - Load and hot reload dynamic config from anywhere. The config interface provides a way to load application 
level config from any source such as env vars, file, etcd. You can merge the sources and even define fallbacks.

- **Data Storage** - A simple data store interface to read, write and delete records. It includes support for memory, file and 
CockroachDB by default. State and persistence becomes a core requirement beyond prototyping and Nitro looks to build that into the framework.

- **Service Discovery** - Automatic service registration and name resolution. Service discovery is at the core of micro service 
development. When service A needs to speak to service B it needs the location of that service. The default discovery mechanism is 
multicast DNS (mdns), a zeroconf system.

- **Load Balancing** - Client side load balancing built on service discovery. Once we have the addresses of any number of instances 
of a service we now need a way to decide which node to route to. We use random hashed load balancing to provide even distribution 
across the services and retry a different node if there's a problem. 

- **Message Encoding** - Dynamic message encoding based on content-type. The client and server will use codecs along with content-type 
to seamlessly encode and decode Go types for you. Any variety of messages could be encoded and sent from different clients. The client 
and server handle this by default. This includes protobuf and json by default.

- **RPC Communication** - Synchronous request/response with support for bidirectional streaming. We provide an abstraction for synchronous 
communication. A request made to a service will be automatically resolved, load balanced, dialled and streamed.

- **Async Messaging** - PubSub is built in as a first class citizen for asynchronous communication and event driven architectures. 
Event notifications are a core pattern in micro service development. The default messaging system is a HTTP event message broker.

- **Synchronization** - Distributed systems are often built in an eventually consistent manner. Support for distributed locking and 
leadership are built in as a Sync interface. When using an eventually consistent database or scheduling use the Sync interface.

- **Pluggable Interfaces** - Nitro makes use of Go interfaces for each package. Because of this these interfaces 
are pluggable and allows Nitro to be runtime agnostic. You can plugin any underlying technology. Find external third party (non stdlib) 
plugins in [github.com/asim/nitro-plugins](https://github.com/asim/nitro-plugins).

## Usage

Here's how to write a quick Nitro App

```go
package main

import (
        "context"
        "fmt"

        "github.com/asim/nitro/app/rpc"
)

// Define a request type
type Request struct {
        Name string
}

// Define a response type
type Response struct {
        Message string
}

// Create your public App Handler
type Handler struct {}

// Create a public Handler method which takes request, response and returns an error
func (h *Handler) Call(ctx context.Context, req *Request, rsp *Response) error {
        rsp.Message = "Hello " + req.Name
        return nil
}

func main() {
        // Create a new App
        app := rpc.NewApp()

        // Set the App name
        app.Name("helloworld")

        // Register the Handler
        app.Handle(new(Handler))

        // Run the App (blocking call)
        app.Run()
}
```

To call a Nitro App

```go
var rsp Response

// Call your app (or any other) by name
err := app.Call("helloworld", "Handler.Call", &Request{Name: "Alice"}, &rsp)
if err != nil {
	fmt.Println(err)
}

fmt.Println(rsp.Message)
```

## License

[Polyform Noncommercial](https://polyformproject.org/licenses/noncommercial/1.0.0/). 

## Docs

See [go-nitro.dev/docs/v3](https://go-nitro.dev/docs/v3/)

## FAQ

### What happened to Go Micro?

Go Micro has now been renamed to Nitro. Go Micro moved back to being a personal project. So no longer lives under the organisation github.com/micro. 
The company is now doubling down on Micro itself and has pulled in the needed interfaces to consolidate a Server, Framework and CLI into one tool. 
Go Micro is now no longer maintained by a company. Yet it continued to create confusion even as a personal repo. So for that reason, we're renaming 
to Nitro.

### Why has the license changed from Apache 2.0 to Polyform Noncommercial

Go Micro was largely a solo maintained effort for the entirety of its lifetime. It has enabled the creation of a company called Micro Services, Inc. which 
now focuses on [Micro](https://github.com/micro/micro) as a Service and has consolidated any interfaces here into a service library in that project. For 
the most part, Go Micro was unfunded and in some ways under appreciated. In version 3.0, going back to something of a personal project of more than 6 years 
I have made the hard decision to relicense as a noncommercial project. For any commercial applications I am looking for [github sponsorship](https://github.com/sponsors/asim) 
so that I can then use those funds for maintenance and support efforts.

### Where are all the plugins?

The plugins now live in [github.com/asim/nitro-plugins](https://github.com/asim/nitro-plugins). This was to reduce the overall size and scope of Go Micro to purely 
a set of interfaces and standard library implementations. Go Plugins is Apache 2.0 licensed but relies on Nitro interfaces and so again can only be used in 
noncommercial setting without a commercial license.

### What's the new direction of Nitro?

Nitro will now focus on distributed app development using the Go standard library. It will continue to define abstractions for distributed systems 
but will only do so without external dependencies. All those external dependencies will live in Nitro Plugins. In this manner the hope is Nitro can be 
picked up with minimal overhead for all sorts of new applications that have a low memory or low resource footprint. The assumption is there are places 
which would like to use distributed apps just as embedded systems or web assembly, unikernels, and related targets that would benefit from a framework 
that defined these as primitives for such use.

### Where is the top level Service definition?

The top level service definition has been moved to the [app](https://github.com/asim/nitro/tree/master/app) package. Nitro exploded in terms 
of the interfaces it offered. While originally it was a small library, this increase in packages has meant the top level can't really provide full scope 
for everything. It's unclear at this time whether the top level definition should return.

### Where are the default initialised interfaces?

The defaults are gone. This proved to be a bad design pattern which meant one definition of an interface needed to live along side it. Over time it became 
quite complex and switching out meant you had a pre-initialised implementation there with a lot of cleanup that wasn't possible. So removing it feels 
as though a cleaner approach to interface design and modularisation of packages.

### Where is the cmd package, flag parsing etc?

These are also gone. The complexity of this code was quite honestly horrible. The command package had to make assumptions about how to load plugins because 
every package depends on other package. As a whole system this became impossible to maintain and even in Micro we'll be looking to scrap it for simpler 
initialisation. The flag parsing, plugin loading, etc is all gone in favour of users self defining it. What we find is most plugins require more initialisiation 
than what we can provide as hard coded values. For this reason we may look into [github.com/google/wire](https://github.com/google/wire) as a better 
alternative.

### How do Nitro and Micro now differ?

Micro is a platform for cloud native development. A complete experience that includes a server, framework and multi-language clients. Beyond that it also 
include environments, multi-tenancy and many more features which push it towards being a hosted Micro as a Service offering. It is a complete platform.

Nitro is more of a pluggable framework for distributed app development and now once again a purely personal project maintained by me and 
perhaps others who still find use for it commercially or noncommercially. It's of sentimental value and something I'd like to carry on for personal projects 
such as things related to edge, IoT, embedded systems, p2p, web assembly, etc.

### I used Go Micro to build microservices. What should I do now?

You should quite honestly go look at [Micro](https://github.com/micro/micro) and then consider the hosted offering at [m3o.com](https://m3o.com) which 
starts as a purely free Dev environment in the cloud. Micro continues to address many of the concerns and requirements you had if not more. It is likely 
you managed metrics, tracing, logging and much other boilerplate that needed to be plugged in. Micro will now take this complete platform story approach 
and help you in that journey e.g you're probably running managed kubernetes on a major cloud provider with many other things. We're doing that for you 
instead as a company and platform team.

### I want to use Go Micro version 2.0 for my company. Can I still do that?

Yes. Go Micro 2.0 is still Apache 2.0 licensed which means you can still freely use it for everything you were using before. If you're a new user 
you can do the same. These things are using go modules so you're import path is simply `github.com/micro/go-micro/v2` as it was before. Because 
GitHub handles redirects this should not break. Please continue to use it if you like, but my own support for 2.0 is now end of life. I will be 
focusing on some side hacking on 3.0 as I find time.
