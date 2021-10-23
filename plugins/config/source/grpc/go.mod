module github.com/asim/go-micro/plugins/config/source/grpc/v4

go 1.16

require (
	github.com/golang/protobuf v1.5.2
	go-micro.dev/v4 v4.2.1
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.38.0
)

replace go-micro.dev/v4 => ../../../../../go-micro
