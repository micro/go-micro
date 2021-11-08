module github.com/asim/go-micro/plugins/client/grpc/v4

go 1.16

require (
	go-micro.dev/v4 v4.3.0
	google.golang.org/grpc v1.42.0
	google.golang.org/grpc/examples v0.0.0-20211102180624-670c133e568e
	google.golang.org/protobuf v1.27.1
)

replace go-micro.dev/v4 => ../../../../go-micro
