module github.com/asim/go-micro/plugins/config/source/grpc/v3

go 1.16

require (
	go-micro.dev/v4 v4.1.0
	github.com/golang/protobuf v1.5.2
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.38.0
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
