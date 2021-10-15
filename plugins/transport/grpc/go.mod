module github.com/asim/go-micro/plugins/transport/grpc/v4

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	go-micro.dev/v4 v4.1.0
	google.golang.org/grpc v1.38.0
)

replace go-micro.dev/v4 => ../../../../go-micro
