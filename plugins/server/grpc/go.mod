module github.com/asim/go-micro/plugins/server/grpc/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/client/grpc/v4 v4.0.0-20211019191242-9edc569e68bb
	github.com/asim/go-micro/plugins/transport/grpc/v4 v4.0.0-20211019191242-9edc569e68bb
	github.com/golang/protobuf v1.5.2
	go-micro.dev/v4 v4.2.1
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	google.golang.org/genproto v0.0.0-20211020151524-b7c3a969101a
	google.golang.org/grpc v1.41.0
)

replace (
	github.com/asim/go-micro/plugins/client/grpc/v4 => ../../../plugins/client/grpc
	github.com/asim/go-micro/plugins/transport/grpc/v4 => ../../../plugins/transport/grpc
	go-micro.dev/v4 => ../../../../go-micro
)
