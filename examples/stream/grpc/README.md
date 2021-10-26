# Description

This example is translate [go grpc example](https://grpc.io/docs/languages/go/basics/) into go-micro.
You can also find the orignal codes in [github.com/grpc/grpc-go](https://github.com/grpc/grpc-go/tree/master/examples/route_guide).

# Run the sample code

## Protobuf

```shell
protoc --go_out=proto --micro_out=proto proto/route_guide.proto
```

## Server

```shell
cd stream/gprc/server
go run .
```

## Client

```shell
cd stream/client
go run main.go
```
