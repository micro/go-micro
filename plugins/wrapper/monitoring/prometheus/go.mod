module github.com/asim/go-micro/plugins/wrapper/monitoring/prometheus/v4

go 1.16

require (
	github.com/asim/go-micro/plugins/broker/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/asim/go-micro/plugins/registry/memory/v4 v4.0.0-20211013123123-62801c3d6883
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.1.0
)

replace (
	github.com/asim/go-micro/plugins/broker/memory/v4 => ../../../../plugins/broker/memory
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../../plugins/registry/memory
	go-micro.dev/v4 => ../../../../../go-micro
)
