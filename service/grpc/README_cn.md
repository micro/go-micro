# Micro gRPC [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-micro/service/grpc?status.svg)](https://godoc.org/github.com/micro/go-micro/service/grpc) [![Travis CI](https://api.travis-ci.org/micro/go-micro/service/grpc.svg?branch=master)](https://travis-ci.org/micro/go-micro/service/grpc) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro/service/grpc)](https://goreportcard.com/report/github.com/micro/go-micro/service/grpc)

Micro gRPC是micro的gRPC框架插件，简化开发基于gRPC的服务。

## 概览

micro提供有基于Go的gRPC插件[go-micro](https://github.com/micro/go-micro)，该插件可以在内部集成gPRC，并与之无缝交互，让开发gRPC更简单，并支持[grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway)。

micro有面向gRPC的[客户端](https://github.com/micro/go-plugins/tree/master/client)和[服务端](https://github.com/micro/go-plugins/tree/master/server)插件，go-grpc库调用客户端/服务端插件生成micro需要的gRPC代码，而客户端/服务端插件都是从[github.com/grpc/grpc-go](https://github.com/grpc/grpc-go)扩展而来，也即是说，我们不需要去知道go-micro是如何编解码或传输就可以使用原生的gRPC。

## 特性

- **服务发现** - go-micro的服务发现基于其[注册](https://github.com/micro/go-plugins/tree/master/registry)与[选择器](https://github.com/micro/go-micro/tree/master/selector)接口，实现了可插拔的服务发现与客户端侧的负载均衡，不需要拨号连接，micro已经把所有都封装好，大家只管用。

- **消息发布订阅** - 因为gRPC只提供同步通信机制，而**Go gRPC**使用go-micro的[broker代理](https://github.com/micro/go-micro/tree/master/broker)提供异步消息，broker也是基于gRPC协议。

- **Micro生态系统** - Micro生态系统包含工具链中，比如api网关、web管理控制台、CLI命令行接口等等。我们通过使用micro来增强gRPC框架的易用性。

## 示例

示例请查看[examples/greeter](https://github.com/micro/go-micro/service/grpc/tree/master/examples/greeter)。

## 开始使用

我们提供相关文档[docs](https://micro.mu/docs/go-grpc_cn.html)，以便上手。