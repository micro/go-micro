module github.com/asim/go-micro/plugins/client/grpc/v4

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	go-micro.dev/v4 v4.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.41.0
	google.golang.org/grpc/examples v0.0.0-20211020220737-f00baa6c3c84
)

replace go-micro.dev/v4 => ../../../../go-micro
