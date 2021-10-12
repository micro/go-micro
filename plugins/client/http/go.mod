module github.com/asim/go-micro/plugins/client/http/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210630062103-c13bb07171bc
	go-micro.dev/v4 v4.0.0
	github.com/golang/protobuf v1.5.2
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../go-micro
)
