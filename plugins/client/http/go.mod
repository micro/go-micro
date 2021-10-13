module github.com/asim/go-micro/plugins/client/http/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v4 master
	go-micro.dev/v4 v4.1.0
	github.com/golang/protobuf v1.5.2
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../plugins/registry/memory
	go-micro.dev/v4 => ../../../../go-micro
)
