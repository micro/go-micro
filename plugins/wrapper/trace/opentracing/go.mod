module github.com/asim/go-micro/plugins/wrapper/trace/opentracing/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.5.1
	github.com/asim/go-micro/v3 v3.5.1
	github.com/opentracing/opentracing-go v1.2.0
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../../go-micro
)
