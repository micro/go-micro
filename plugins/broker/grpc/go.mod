module github.com/asim/go-micro/plugins/broker/grpc/v3

go 1.13

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.5
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/grpc v1.26.0
)

replace github.com/asim/go-micro/plugins/registry/memory/v3 => ../../registry/memory
