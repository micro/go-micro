module github.com/asim/go-micro/plugins/broker/grpc/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	go-micro.dev/v4 v4.1.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
	google.golang.org/grpc v1.38.0
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../plugins/registry/memory
	go-micro.dev/v4 => ../../../../go-micro
)
