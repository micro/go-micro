module github.com/asim/go-micro/plugins/client/grpc/v4

go 1.16

require (
	go-micro.dev/v4 v4.2.1
	google.golang.org/grpc v1.41.0
	google.golang.org/grpc/examples v0.0.0-20211020220737-f00baa6c3c84
	google.golang.org/protobuf v1.26.0
)

replace go-micro.dev/v4 => ../../../../go-micro
