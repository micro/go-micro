module github.com/asim/go-micro/plugins/selector/label/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1
	github.com/asim/go-micro/v3 v3.5.1
)

replace (
	github.com/asim/go-micro/v3 => ../../../../go-micro
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1 => ../../../plugins/registry/memory
)
