# gRPC Source

The gRPC source reads from a gRPC server

## Server

A gRPC source server should implement the [`Source`](https://github.com/micro/go-micro/config/blob/master/source/grpc/proto/grpc.proto#L3L6) proto interface.

```
service Source {
	rpc Read(ReadRequest) returns (ReadResponse) {};
	rpc Watch(WatchRequest) returns (stream WatchResponse) {};
}
```

## New Source

Specify source with address and path

```go
source := grpc.NewSource(
	// optionally specify server address; default to localhost:8080
	grpc.WithAddress("10.0.0.10:8500"),
	// optionally specify a path; defaults to /micro/config
	grpc.WithPath("/my/config/path"),
)
```

## Load Source

Load the source into config

```go
// Create new config
conf := config.NewConfig()

// Load file source
conf.Load(grpcSource)
```
