module github.com/asim/go-micro/plugins/config/source/grpc/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	github.com/golang/protobuf v1.5.2
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	google.golang.org/grpc v1.38.0
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
