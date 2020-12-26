# Examples

This is a repository for micro examples. Feel free to contribute.

## Contents

- [api](api) - Provides API usage examples
- [booking](booking) - A booking.com demo application
- [broker](broker) - A example of using Broker for Publish and Subscribing.
- [client](client) - Usage of the Client package to call a service.
- [command](command) - An example of bot commands as micro services
- [config](config) - Using Go Config for dynamic config
- [event](event) - Using the API Gateway event handler
- [filter](filter) - Filter nodes of a service when requesting
- [flags](flags) - Using command line flags with a service
- [form](form) - How to parse a form behind the micro api
- [function](function) - Example of using Function programming model
- [getip](getip) - Get the local and remote ip from metadata
- [graceful](graceful) - Demonstrates graceful shutdown of a service
- [greeter](greeter) - A complete greeter example (includes python, ruby examples)
- [heartbeat](heartbeat) - Make services heartbeat with discovery for high availability
- [helloworld](helloworld) - Hello world using micro
- [kubernetes](kubernetes) - Examples of using the k8s registry and grpc
- [metadata](metadata) - Extracting metadata from context of a request
- [mocking](mocking) - Demonstrate mocking helloworld service
- [noproto](noproto) - Use micro without protobuf or code generation, only go types
- [options](options) - Setting options in the go-micro framework
- [plugins](plugins) - How to use plugins
- [pubsub](pubsub) - Example of using pubsub at the client/server level
- [grpc](grpc) - Examples of how to use [go-micro/service/grpc](https://github.com/micro/go-micro/service/grpc)
- [redirect](redirect) - An example of how to http redirect using an API service
- [roundrobin](roundrobin) - A stateful client wrapper for true round robin of requests
- [secure](secure) - Demonstrates use of transport secure option for self signed certs
- [server](server) - Use of the Server package directly to server requests.
- [service](service) - Example of the top level Service in go-micro.
- [sharding](sharding) - An example of how to shard requests or use session affinity
- [shutdown](shutdown) - Demonstrates graceful shutdown via context cancellation
- [stream](stream) - An example of a streaming service and client
- [template](template) - Api, web and srv service templates generated with the 'micro new' command
- [tunnel](tunnel) - How to use connection tunneling with the tunnel package
- [waitgroup](waitgroup) - Demonstrates how to use a waitgroup with a service
- [wrapper](wrapper) - A simple example of using a log wrapper

## Community

Find contributions from the community via the [explorer](https://micro.mu/projects/)

## Install

Install [protoc](https://github.com/google/protobuf) for your environment. Then:

```shell
# install protoc-gen-go
go get github.com/golang/protobuf/{proto,protoc-gen-go}
# install protoc-gen-micro
go get github.com/micro/micro/v2/cmd/protoc-gen-micro@master
```

To recompile any proto after changes:

```shell
protoc --proto_path=$GOPATH/src:. --micro_out=. --go_out=. path/to/proto
```
