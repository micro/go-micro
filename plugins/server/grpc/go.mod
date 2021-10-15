module github.com/asim/go-micro/plugins/server/grpc/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/broker/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/client/grpc/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/registry/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/transport/grpc/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/golang/protobuf v1.5.2
	go-micro.dev/v4 v4.1.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/genproto v0.0.0-20200806141610-86f49bd18e98
	google.golang.org/grpc v1.38.0
)

replace (
	github.com/asim/go-micro/plugins/broker/memory/v4 => ../../../plugins/broker/memory
	github.com/asim/go-micro/plugins/client/grpc/v4 => ../../../plugins/client/grpc
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../plugins/registry/memory
	github.com/asim/go-micro/plugins/transport/grpc/v4 => ../../../plugins/transport/grpc
	go-micro.dev/v4 => ../../../../go-micro
)
