module github.com/asim/go-micro/plugins/client/grpc/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210630062103-c13bb07171bc
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/golang/protobuf v1.5.2
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	google.golang.org/grpc v1.38.0
	google.golang.org/grpc/examples v0.0.0-20210902184326-c93e472777b9
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../go-micro
)
