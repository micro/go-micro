# Micro gRPC [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-micro/service/grpc?status.svg)](https://godoc.org/github.com/micro/go-micro/service/grpc) [![Travis CI](https://api.travis-ci.org/micro/go-micro/service/grpc.svg?branch=master)](https://travis-ci.org/micro/go-micro/service/grpc) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro/service/grpc)](https://goreportcard.com/report/github.com/micro/go-micro/service/grpc)

A micro gRPC framework. A simplified experience for building gRPC services. 

## Overview

**Go gRPC** makes use of [go-micro](https://github.com/micro/go-micro) plugins to create a simpler framework for gRPC development. 
It interoperates with standard gRPC services seamlessly, including the [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway). 
The go-grpc library uses the go-micro broker, client and server plugins which make use of 
[github.com/grpc/grpc-go](https://github.com/grpc/grpc-go) internally. 
This means we ignore the go-micro codec and transport but provide a native grpc experience.

<img src="https://micro.mu/docs/images/go-grpc.svg" />

## Features

- **Service Discovery** - We make use of go-micro's registry and selector interfaces to provide pluggable discovery 
and client side load balancing. There's no need to dial connections, we'll do everything beneath the covers for you.

- **PubSub Messaging** - Where gRPC only provides you synchronous communication, **Go gRPC** uses the go-micro broker 
to provide asynchronous messaging while using the gRPC protocol.

- **Micro Ecosystem** - Make use of the existing micro ecosystem of tooling including our api gateway, web dashboard, 
command line interface and much more. We're enhancing gRPC with a simplified experience using micro.

## Examples

Find an example greeter service in [examples/greeter](https://github.com/micro/go-micro/service/grpc/tree/master/examples/greeter).

## Getting Started

See the [docs](https://micro.mu/docs/go-grpc.html) to get started.

## I18n

### [中文](README_cn.md)
