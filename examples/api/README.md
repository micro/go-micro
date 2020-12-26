# API

This repo contains examples for using the micro api.

## Overview

The [micro api](https://github.com/micro/micro/tree/master/api) is an API gateway which serves HTTP and routes dynamically based on service discovery.

The micro api by default serves the namespace go.micro.api. Our service names include this plus a unique name e.g go.micro.api.example. 
You can change the namespace via the flag `--namespace=`.

The micro api has a number of different handlers which lets you define what kind of API services you want. See examples below. The handler 
can be set via the flag `--handler=`. The default handler is "rpc".

## Contents

- api - an rpc handler that provides the entire http headers and request
- proxy - use the api as a http reverse proxy
- rpc - make an rpc request to a go-micro app
- meta - specify which handler to use via configuration in code

## Request Mapping

### API/RPC

Micro maps http paths to rpc services. The mapping table can be seen below.

The default namespace for the api is **go.micro.api** but you can set your own namespace via `--namespace`.

URLs are mapped as follows:

Path	|	Service	|	Method
----	|	----	|	----
/foo/bar	|	go.micro.api.foo	|	Foo.Bar
/foo/bar/baz	|	go.micro.api.foo	|	Bar.Baz
/foo/bar/baz/cat	|	go.micro.api.foo.bar	|	Baz.Cat

Versioned API URLs can easily be mapped to service names:

Path	|	Service	|	Method
----	|	----	|	----
/foo/bar	|	go.micro.api.foo	|	Foo.Bar
/v1/foo/bar	|	go.micro.api.v1.foo	|	Foo.Bar
/v1/foo/bar/baz	|	go.micro.api.v1.foo	|	Bar.Baz
/v2/foo/bar	|	go.micro.api.v2.foo	|	Foo.Bar
/v2/foo/bar/baz	|	go.micro.api.v2.foo	|	Bar.Baz

### Proxy Mapping

Starting the API with `--handler=http` will reverse proxy requests to backend services within the served API namespace (default: go.micro.api). 

Example

Path	|	Service	|	Service Path
---	|	---	|	---
/greeter	|	go.micro.api.greeter	|	/greeter
/greeter/:name	|	go.micro.api.greeter	|	/greeter/:name
