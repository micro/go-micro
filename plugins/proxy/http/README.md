# HTTP Proxy

This is a http proxy plugin which converts RPC to HTTP request

## Overview

`NewService` returns a new http proxy. It acts as a micro service and proxies to a http backend.
Routes are dynamically set e.g Foo.Bar routes to /foo/bar. The default backend is http:localhost:9090.
Optionally specify the backend endpoint url or the router. Also choose to register specific endpoints.

## Usage

```
service := NewService(
      micro.Name("greeter"),
      // Sets the default http endpoint
      http.WithBackend("http:localhost:10001"),
)

// Set fixed backend endpoints
// register an endpoint
http.RegisterEndpoint("Hello.World", "/helloworld")

service := NewService(
      micro.Name("greeter"),
      // Set the http endpoint
      http.WithBackend("http:localhost:10001"),
)
```
