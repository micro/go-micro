module github.com/asim/go-micro/plugins/wrapper/monitoring/prometheus/v3

go 1.13

require (
	github.com/asim/go-micro/plugins/broker/memory/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-00010101000000-000000000000
	github.com/asim/go-micro/v3 v3.0.0-20210120135431-d94936f6c97c
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/client_model v0.2.0
	github.com/stretchr/testify v1.4.0
)

replace github.com/asim/go-micro/plugins/broker/memory/v3 => ../../../broker/memory

replace github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../registry/memory
