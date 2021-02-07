module github.com/asim/go-micro/plugins/proxy/http/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1
	github.com/asim/go-micro/v3 v3.5.1
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../go-micro
)
