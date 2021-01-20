module github.com/asim/go-micro/plugins/server/grpc/v3

go 1.15

require (
	github.com/asim/go-micro/plugins/broker/memory/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/client/grpc/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/transport/grpc/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/golang/protobuf v1.4.3
	golang.org/x/net v0.0.0-20201224014010-6772e930b67b
	google.golang.org/genproto v0.0.0-20201214200347-8c77b98c765d
	google.golang.org/grpc v1.27.0
)

replace github.com/asim/go-micro/plugins/client/grpc/v3 => ../../client/grpc

replace github.com/asim/go-micro/plugins/transport/grpc/v3 => ../../transport/grpc

replace github.com/asim/go-micro/plugins/broker/memory/v3 => ../../broker/memory

replace github.com/asim/go-micro/plugins/registry/memory/v3 => ../../registry/memory

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0
