module github.com/asim/go-micro/plugins/client/grpc/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1
	github.com/asim/go-micro/v3 v3.5.1
	github.com/golang/protobuf v1.4.3
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	google.golang.org/grpc v1.38.0
	google.golang.org/grpc/examples v0.0.0-20210628165121-83f9def5feb3
)

replace (
	github.com/asim/go-micro/v3 => ../../../../go-micro
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../plugins/registry/memory
)
