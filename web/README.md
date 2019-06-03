# Go Web [![GoDoc](https://godoc.org/github.com/micro/go-micro/web?status.svg)](https://godoc.org/github.com/micro/go-micro/web) [![Travis CI](https://travis-ci.org/micro/go-micro/web.svg?branch=master)](https://travis-ci.org/micro/go-micro/web) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro/web)](https://goreportcard.com/report/github.com/micro/go-micro/web)

**Go Web** is a framework for micro service web development.

## Overview

Go Web provides a tiny HTTP web server library which leverages [go-micro](https://github.com/micro/go-micro) to create 
micro web services as first class citizens in a microservice world. It wraps go-micro to give you service discovery, 
heartbeating and the ability to create web apps as microservices.

## Features

- **Service Discovery** - Services are automatically registered in service discovery on startup. Go Web includes 
a http.Client with pre-initialised roundtripper which makes use of service discovery so you can use service names.

- **Heartbeating** - Go Web apps will periodically heartbeat with service discovery to provide liveness updates. 
In the event a service fails it will be removed from the registry after a pre-defined expiry time.

- **Custom Handlers** - Specify your own http router for handling requests. This allows you to maintain full 
control over how you want to route to internal handlers.

- **Static Serving** - Go Web automatically detects a local static `html` dir and serves files if no route handler 
is specified. A quick solution for those who want to write JS web apps as microservices.

## Getting Started

- [Dependencies](#dependencies)
- [Usage](#usage)
- [Set Handler](#set-handler)
- [Call Service](#call-service)
- [Static Files](#static-files)

## Dependencies

Go Web makes use of Go Micro which means it needs service discovery

See the [go-micro](https://github.com/micro/go-micro#service-discovery) for install instructions

For a quick start use consul

```
# install
brew install consul

# run
consul agent -dev
```

## Usage

```go
service := web.NewService(
	web.Name("example.com"),
)

service.HandleFunc("/foo", fooHandler)

if err := service.Init(); err != nil {
	log.Fatal(err)
}

if err := service.Run(); err != nil {
	log.Fatal(err)
}
```

## Set Handler

You might have a preference for a HTTP handler, so use something else. This loses the ability to register endpoints in discovery 
but we'll fix that soon.

```go
import "github.com/gorilla/mux"

r := mux.NewRouter()
r.HandleFunc("/", indexHandler)
r.HandleFunc("/objects/{object}", objectHandler)

service := web.NewService(
	web.Handler(r)
)
```

## Call Service

Go-web includes a http.Client with a custom http.RoundTripper that uses service discovery

```go
c := service.Client()

rsp, err := c.Get("http://example.com/foo")
```

This will lookup service discovery for the service `example.com` and route to one of the available nodes.

## Static Files

Go web was always meant as a way to register web apps where the majority of the code would be written in JS. To enable that by default, if no handler is registered on "/" and we find a local "html" directory then static files will be served.

You will see a log output like so.

```
2019/05/12 14:55:47 Enabling static file serving from /tmp/foo/html
```

If you want to set this path manually use the StaticDir option. If a relative path is specified we will use os.Getwd() and prefix this.

```
service := web.NewService(
	web.Name("example.com"),
	web.StaticDir("/tmp/example.com/html"),
)

```
