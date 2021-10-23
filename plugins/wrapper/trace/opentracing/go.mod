module github.com/asim/go-micro/plugins/wrapper/trace/opentracing/v4

go 1.16

require (
	github.com/opentracing/opentracing-go v1.2.0
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
